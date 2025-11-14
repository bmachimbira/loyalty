package budget

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/bmachimbira/loyalty/api/internal/db"
)

// PeriodType represents the budget period type
type PeriodType string

const (
	// PeriodRolling means budget does not reset
	PeriodRolling PeriodType = "rolling"

	// PeriodMonthly means budget resets monthly
	PeriodMonthly PeriodType = "monthly"

	// PeriodQuarterly means budget resets quarterly
	PeriodQuarterly PeriodType = "quarterly"

	// PeriodYearly means budget resets yearly
	PeriodYearly PeriodType = "yearly"
)

// ResetResult contains the result of a budget reset
type ResetResult struct {
	BudgetID     pgtype.UUID
	BudgetName   string
	PreviousBalance float64
	NewBalance   float64
	RolloverAmount float64
	ResetAt      time.Time
}

// ResetBudget resets a budget's balance to zero and optionally creates a rollover entry
func (s *Service) ResetBudget(ctx context.Context, tenantID, budgetID pgtype.UUID, createRollover bool) (*ResetResult, error) {
	// Start transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Create queries with transaction
	qtx := s.queries.WithTx(tx)

	// Get budget
	budget, err := qtx.GetBudgetByID(ctx, db.GetBudgetByIDParams{
		ID:       budgetID,
		TenantID: tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get budget: %w", err)
	}

	var previousBalance float64
	budget.Balance.Float(&previousBalance)

	// If there's a balance and we want to create a rollover entry
	if createRollover && previousBalance > 0 {
		// Create a release entry for the existing balance
		releaseAmount := pgtype.Numeric{}
		if err := releaseAmount.Scan(fmt.Sprintf("%.2f", -previousBalance)); err != nil {
			return nil, fmt.Errorf("failed to convert balance: %w", err)
		}

		_, err = qtx.InsertLedgerEntry(ctx, db.InsertLedgerEntryParams{
			TenantID:  tenantID,
			BudgetID:  budgetID,
			EntryType: "release",
			Currency:  budget.Currency,
			Amount:    releaseAmount,
			RefType:   pgtype.Text{String: "period_reset", Valid: true},
			RefID:     pgtype.UUID{}, // No specific reference
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create rollover entry: %w", err)
		}
	}

	// Reset balance to 0
	resetAmount := pgtype.Numeric{}
	if err := resetAmount.Scan(fmt.Sprintf("%.2f", -previousBalance)); err != nil {
		return nil, fmt.Errorf("failed to convert reset amount: %w", err)
	}

	err = qtx.UpdateBudgetBalance(ctx, db.UpdateBudgetBalanceParams{
		ID:       budgetID,
		TenantID: tenantID,
		Balance:  resetAmount,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to reset balance: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	result := &ResetResult{
		BudgetID:       budgetID,
		BudgetName:     budget.Name,
		PreviousBalance: previousBalance,
		NewBalance:     0,
		RolloverAmount: 0, // In a rolling budget, we might want to track this
		ResetAt:        time.Now(),
	}

	s.logger.Info("budget reset",
		"budget_id", budgetID,
		"budget_name", budget.Name,
		"previous_balance", previousBalance,
		"create_rollover", createRollover)

	return result, nil
}

// ResetMonthlyBudgets resets all monthly budgets at the start of a new month
// This should be called by a cron job on the 1st of each month
func (s *Service) ResetMonthlyBudgets(ctx context.Context) ([]ResetResult, error) {
	s.logger.Info("starting monthly budget reset")

	results := []ResetResult{}

	// Get all budgets with period = 'monthly'
	// Note: We need to query across all tenants, so we'll need to modify the query
	// For now, we'll implement this assuming we get tenants separately

	// Get all tenants
	// This is a simplified version - in production, you'd iterate through tenants
	// For now, we'll return an error indicating this needs tenant-specific calls

	return results, fmt.Errorf("ResetMonthlyBudgets should be called per-tenant, use ResetMonthlyBudgetsForTenant instead")
}

// ResetMonthlyBudgetsForTenant resets all monthly budgets for a specific tenant
func (s *Service) ResetMonthlyBudgetsForTenant(ctx context.Context, tenantID pgtype.UUID) ([]ResetResult, error) {
	if !tenantID.Valid {
		return nil, fmt.Errorf("tenant_id is required")
	}

	s.logger.Info("starting monthly budget reset for tenant", "tenant_id", tenantID)

	// Get all budgets for tenant
	budgets, err := s.queries.ListBudgets(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list budgets: %w", err)
	}

	results := []ResetResult{}
	resetCount := 0
	errorCount := 0

	// Reset each monthly budget
	for _, budget := range budgets {
		// Only reset budgets with period = "monthly"
		if budget.Period != string(PeriodMonthly) {
			continue
		}

		result, err := s.ResetBudget(ctx, tenantID, budget.ID, true)
		if err != nil {
			s.logger.Error("failed to reset budget",
				"budget_id", budget.ID,
				"budget_name", budget.Name,
				"error", err)
			errorCount++
			continue
		}

		results = append(results, *result)
		resetCount++
	}

	s.logger.Info("completed monthly budget reset for tenant",
		"tenant_id", tenantID,
		"total_budgets", len(budgets),
		"reset_count", resetCount,
		"error_count", errorCount)

	return results, nil
}

// ShouldResetBudget checks if a budget should be reset based on its period
func (s *Service) ShouldResetBudget(budget *db.Budget, now time.Time) bool {
	period := PeriodType(budget.Period)

	switch period {
	case PeriodMonthly:
		// Reset on the 1st of each month
		return now.Day() == 1

	case PeriodQuarterly:
		// Reset on the 1st of Jan, Apr, Jul, Oct
		month := now.Month()
		return now.Day() == 1 && (month == time.January || month == time.April || month == time.July || month == time.October)

	case PeriodYearly:
		// Reset on Jan 1st
		return now.Month() == time.January && now.Day() == 1

	case PeriodRolling:
		// Never reset rolling budgets
		return false

	default:
		return false
	}
}

// GetNextResetDate returns the next reset date for a budget
func (s *Service) GetNextResetDate(budget *db.Budget, from time.Time) *time.Time {
	period := PeriodType(budget.Period)

	switch period {
	case PeriodMonthly:
		// Next 1st of month
		next := time.Date(from.Year(), from.Month()+1, 1, 0, 0, 0, 0, from.Location())
		return &next

	case PeriodQuarterly:
		// Next quarter start (Jan, Apr, Jul, Oct)
		month := from.Month()
		var nextMonth time.Month
		year := from.Year()

		if month < time.April {
			nextMonth = time.April
		} else if month < time.July {
			nextMonth = time.July
		} else if month < time.October {
			nextMonth = time.October
		} else {
			nextMonth = time.January
			year++
		}

		next := time.Date(year, nextMonth, 1, 0, 0, 0, 0, from.Location())
		return &next

	case PeriodYearly:
		// Next Jan 1st
		year := from.Year()
		if from.Month() >= time.January && from.Day() > 1 {
			year++
		}
		next := time.Date(year, time.January, 1, 0, 0, 0, 0, from.Location())
		return &next

	case PeriodRolling:
		// Rolling budgets never reset
		return nil

	default:
		return nil
	}
}

// ScheduleMonthlyReset returns a note about scheduling monthly resets
// This is a placeholder for the actual cron job implementation
func ScheduleMonthlyReset() string {
	return `
To schedule monthly budget resets, add the following cron job:

# Reset monthly budgets on the 1st of each month at midnight
0 0 1 * * /path/to/your/app reset-monthly-budgets

Or use a container scheduler like:
- Kubernetes CronJob
- AWS EventBridge
- Cloud Scheduler (GCP)

The cron job should call the ResetMonthlyBudgetsForTenant function for each tenant.
`
}
