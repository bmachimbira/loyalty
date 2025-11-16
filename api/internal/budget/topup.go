package budget

import (
	"context"
	"errors"
	"fmt"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// TopupBudgetParams contains parameters for topping up a budget
type TopupBudgetParams struct {
	TenantID pgtype.UUID
	BudgetID pgtype.UUID
	Amount   string // String to avoid floating point precision issues
	Currency string
}

// Validate validates the topup budget parameters
func (p TopupBudgetParams) Validate() error {
	if !p.TenantID.Valid {
		return errors.New("tenant_id is required")
	}
	if !p.BudgetID.Valid {
		return errors.New("budget_id is required")
	}
	if p.Amount == "" || p.Amount == "0" {
		return errors.New("amount must be greater than 0")
	}
	if !IsValidCurrency(p.Currency) {
		return ErrCurrencyMismatch
	}
	return nil
}

// TopupResult contains the result of a budget topup
type TopupResult struct {
	BudgetID   pgtype.UUID
	Amount     string
	Currency   string
	NewBalance float64
}

// TopupBudget adds funds to a budget
// This increases both the available balance for reservations
func (s *Service) TopupBudget(ctx context.Context, params TopupBudgetParams) (*TopupResult, error) {
	// Validate parameters
	if err := params.Validate(); err != nil {
		return nil, err
	}

	// Start transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Create queries with transaction
	qtx := s.queries.WithTx(tx)

	// Get budget to verify currency
	budget, err := qtx.GetBudgetByID(ctx, db.GetBudgetByIDParams{
		ID:       params.BudgetID,
		TenantID: params.TenantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBudgetNotFound
		}
		return nil, fmt.Errorf("failed to get budget: %w", err)
	}

	// Verify currency matches
	if budget.Currency != params.Currency {
		return nil, ErrCurrencyMismatch
	}

	// Call fund_budget database function
	// This function will update balance and create ledger entry
	var success bool
	err = tx.QueryRow(ctx,
		"SELECT fund_budget($1, $2, $3, $4)",
		params.TenantID,
		params.BudgetID,
		params.Amount,
		params.Currency,
	).Scan(&success)

	if err != nil {
		return nil, fmt.Errorf("failed to fund budget: %w", err)
	}

	if !success {
		return nil, errors.New("fund budget function returned false")
	}

	// Get updated budget
	updatedBudget, err := qtx.GetBudgetByID(ctx, db.GetBudgetByIDParams{
		ID:       params.BudgetID,
		TenantID: params.TenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get updated budget: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	balanceVal, err := updatedBudget.Balance.Float64Value()
	if err != nil {
		return nil, fmt.Errorf("invalid balance value: %w", err)
	}
	newBalance := balanceVal.Float64

	result := &TopupResult{
		BudgetID:   params.BudgetID,
		Amount:     params.Amount,
		Currency:   params.Currency,
		NewBalance: newBalance,
	}

	s.logger.Info("budget topped up",
		"budget_id", params.BudgetID,
		"amount", params.Amount,
		"currency", params.Currency,
		"new_balance", newBalance)

	return result, nil
}
