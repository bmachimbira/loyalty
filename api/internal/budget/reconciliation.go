package budget

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/bmachimbira/loyalty/api/internal/db"
)

// ReconciliationResult contains the result of a budget reconciliation
type ReconciliationResult struct {
	BudgetID           pgtype.UUID
	CurrentBalance     float64
	CalculatedBalance  float64
	Discrepancy        float64
	HasDiscrepancy     bool
	LedgerEntryCount   int64
	TotalFunded        float64
	TotalReserved      float64
	TotalCharged       float64
	TotalReleased      float64
	ExpectedReserved   float64
}

// ReconciliationReport is a detailed reconciliation report
type ReconciliationReport struct {
	Results []ReconciliationResult
	Summary ReconciliationSummary
}

// ReconciliationSummary contains summary statistics
type ReconciliationSummary struct {
	TotalBudgets          int
	BudgetsWithDiscrepancy int
	TotalDiscrepancy      float64
}

// ReconcileBudget verifies that a budget's balance matches the ledger entries
// This should be run periodically (e.g., daily) to detect any inconsistencies
func (s *Service) ReconcileBudget(ctx context.Context, tenantID, budgetID pgtype.UUID) (*ReconciliationResult, error) {
	if !tenantID.Valid || !budgetID.Valid {
		return nil, errors.New("tenant_id and budget_id are required")
	}

	// Get current budget to verify it exists
	_, err := s.queries.GetBudgetByID(ctx, db.GetBudgetByIDParams{
		ID:       budgetID,
		TenantID: tenantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBudgetNotFound
		}
		return nil, fmt.Errorf("failed to get budget: %w", err)
	}

	// Call reconcile_budget database function
	var currentBalance, calculatedBalance, discrepancy float64
	err = s.pool.QueryRow(ctx,
		"SELECT current_balance, calculated_balance, discrepancy FROM reconcile_budget($1)",
		budgetID,
	).Scan(&currentBalance, &calculatedBalance, &discrepancy)

	if err != nil {
		return nil, fmt.Errorf("failed to reconcile budget: %w", err)
	}

	// Get ledger summary for detailed information
	// Use a wide date range to get all entries
	entries, err := s.queries.GetLedgerEntriesByDateRangeOnly(ctx, db.GetLedgerEntriesByDateRangeOnlyParams{
		TenantID: tenantID,
		BudgetID: budgetID,
		Limit:    10000, // Get all entries
		Offset:   0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger entries: %w", err)
	}

	// Calculate totals by entry type
	var totalFunded, totalReserved, totalCharged, totalReleased float64
	for _, entry := range entries {
		amountVal, err := entry.Amount.Float64Value()
		if err != nil {
			return nil, fmt.Errorf("invalid amount value: %w", err)
		}
		amount := amountVal.Float64

		switch entry.EntryType {
		case "fund":
			totalFunded += amount
		case "reserve":
			totalReserved += amount
		case "charge":
			totalCharged += amount
		case "release":
			// Release entries are stored as negative amounts
			totalReleased += -amount
		}
	}

	// Expected reserved = funded + reserved - released
	// (charged entries don't affect balance, just record the charge)
	expectedReserved := totalFunded + totalReserved + totalReleased // totalReleased is already negative

	result := &ReconciliationResult{
		BudgetID:           budgetID,
		CurrentBalance:     currentBalance,
		CalculatedBalance:  calculatedBalance,
		Discrepancy:        discrepancy,
		HasDiscrepancy:     discrepancy != 0,
		LedgerEntryCount:   int64(len(entries)),
		TotalFunded:        totalFunded,
		TotalReserved:      totalReserved,
		TotalCharged:       totalCharged,
		TotalReleased:      totalReleased,
		ExpectedReserved:   expectedReserved,
	}

	// Log discrepancy if found
	if result.HasDiscrepancy {
		s.logger.Error("budget reconciliation discrepancy detected",
			"budget_id", budgetID,
			"tenant_id", tenantID,
			"current_balance", currentBalance,
			"calculated_balance", calculatedBalance,
			"discrepancy", discrepancy,
			"total_funded", totalFunded,
			"total_reserved", totalReserved,
			"total_charged", totalCharged,
			"total_released", totalReleased)
	} else {
		s.logger.Info("budget reconciliation successful",
			"budget_id", budgetID,
			"tenant_id", tenantID,
			"balance", currentBalance,
			"entry_count", result.LedgerEntryCount)
	}

	return result, nil
}

// ReconcileAllBudgets reconciles all budgets for a tenant
func (s *Service) ReconcileAllBudgets(ctx context.Context, tenantID pgtype.UUID) (*ReconciliationReport, error) {
	if !tenantID.Valid {
		return nil, errors.New("tenant_id is required")
	}

	// Get all budgets for tenant
	budgets, err := s.queries.ListBudgets(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list budgets: %w", err)
	}

	report := &ReconciliationReport{
		Results: make([]ReconciliationResult, 0, len(budgets)),
		Summary: ReconciliationSummary{
			TotalBudgets: len(budgets),
		},
	}

	// Reconcile each budget
	for _, budget := range budgets {
		result, err := s.ReconcileBudget(ctx, tenantID, budget.ID)
		if err != nil {
			s.logger.Error("failed to reconcile budget",
				"budget_id", budget.ID,
				"error", err)
			continue
		}

		report.Results = append(report.Results, *result)

		if result.HasDiscrepancy {
			report.Summary.BudgetsWithDiscrepancy++
			report.Summary.TotalDiscrepancy += result.Discrepancy
		}
	}

	s.logger.Info("reconciled all budgets",
		"tenant_id", tenantID,
		"total_budgets", report.Summary.TotalBudgets,
		"budgets_with_discrepancy", report.Summary.BudgetsWithDiscrepancy,
		"total_discrepancy", report.Summary.TotalDiscrepancy)

	return report, nil
}

// FixDiscrepancy attempts to fix a budget balance discrepancy
// This should be used with caution and only after manual review
func (s *Service) FixDiscrepancy(ctx context.Context, tenantID, budgetID pgtype.UUID, notes string) error {
	// Start transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get reconciliation result
	result, err := s.ReconcileBudget(ctx, tenantID, budgetID)
	if err != nil {
		return fmt.Errorf("failed to reconcile: %w", err)
	}

	if !result.HasDiscrepancy {
		return errors.New("no discrepancy to fix")
	}

	// Create a "reverse" entry to fix the discrepancy
	qtx := s.queries.WithTx(tx)

	// Convert discrepancy to numeric
	var discrepancyNumeric pgtype.Numeric
	if err := discrepancyNumeric.Scan(fmt.Sprintf("%.2f", result.Discrepancy)); err != nil {
		return fmt.Errorf("failed to convert discrepancy: %w", err)
	}

	// Get budget currency
	budget, err := qtx.GetBudgetByID(ctx, db.GetBudgetByIDParams{
		ID:       budgetID,
		TenantID: tenantID,
	})
	if err != nil {
		return fmt.Errorf("failed to get budget: %w", err)
	}

	// Create reverse entry
	_, err = qtx.InsertLedgerEntry(ctx, db.InsertLedgerEntryParams{
		TenantID:  tenantID,
		BudgetID:  budgetID,
		EntryType: "reverse",
		Currency:  budget.Currency,
		Amount:    discrepancyNumeric,
		RefType:   pgtype.Text{String: "reconciliation", Valid: true},
		RefID:     pgtype.UUID{}, // No specific reference
	})
	if err != nil {
		return fmt.Errorf("failed to create reverse entry: %w", err)
	}

	// Update budget balance
	err = qtx.UpdateBudgetBalance(ctx, db.UpdateBudgetBalanceParams{
		ID:       budgetID,
		TenantID: tenantID,
		Balance:  discrepancyNumeric,
	})
	if err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Warn("budget discrepancy fixed",
		"budget_id", budgetID,
		"tenant_id", tenantID,
		"discrepancy", result.Discrepancy,
		"notes", notes)

	return nil
}
