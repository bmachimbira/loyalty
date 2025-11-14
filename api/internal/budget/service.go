package budget

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/bmachimbira/loyalty/api/internal/db"
)

var (
	// ErrInsufficientFunds is returned when a reservation would exceed the hard cap
	ErrInsufficientFunds = errors.New("insufficient budget funds")

	// ErrBudgetNotFound is returned when a budget doesn't exist
	ErrBudgetNotFound = errors.New("budget not found")

	// ErrCurrencyMismatch is returned when currency doesn't match budget currency
	ErrCurrencyMismatch = errors.New("currency mismatch")

	// ErrInvalidAmount is returned when amount is invalid (e.g. negative)
	ErrInvalidAmount = errors.New("invalid amount")

	// ErrAlreadyCharged is returned when trying to charge an already-charged reservation
	ErrAlreadyCharged = errors.New("reservation already charged")

	// ErrReservationNotFound is returned when a reservation is not found
	ErrReservationNotFound = errors.New("reservation not found")
)

// Service handles budget operations
type Service struct {
	queries *db.Queries
	pool    DBTX
	logger  *slog.Logger
}

// DBTX interface for database operations (matches sqlc's interface)
type DBTX interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgx.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

// NewService creates a new budget service
func NewService(pool DBTX, queries *db.Queries, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		queries: queries,
		pool:    pool,
		logger:  logger,
	}
}

// ReserveBudget reserves an amount from a budget for a future charge
// This is called when a reward is issued (reserved state)
func (s *Service) ReserveBudget(ctx context.Context, params ReserveBudgetParams) (*ReservationResult, error) {
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

	// Get budget to verify currency and check soft cap
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

	// Call reserve_budget database function
	// This function will check capacity and update balance atomically
	var success bool
	err = tx.QueryRow(ctx,
		"SELECT reserve_budget($1, $2, $3, $4, $5)",
		params.TenantID,
		params.BudgetID,
		params.Amount,
		params.Currency,
		params.RefID,
	).Scan(&success)

	if err != nil {
		return nil, fmt.Errorf("failed to reserve budget: %w", err)
	}

	if !success {
		return nil, ErrInsufficientFunds
	}

	// Get updated budget for soft cap check
	updatedBudget, err := qtx.GetBudgetByID(ctx, db.GetBudgetByIDParams{
		ID:       params.BudgetID,
		TenantID: params.TenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get updated budget: %w", err)
	}

	// Check soft cap (warning only)
	softCapExceeded := false
	var balance, softCap, hardCap float64
	updatedBudget.Balance.Float(&balance)
	updatedBudget.SoftCap.Float(&softCap)
	updatedBudget.HardCap.Float(&hardCap)

	if balance > softCap {
		softCapExceeded = true
		// Trigger soft cap alert (non-blocking)
		go func() {
			alertCtx := context.Background()
			if err := s.CheckSoftCapAlert(alertCtx, params.TenantID, params.BudgetID); err != nil {
				s.logger.Error("failed to trigger soft cap alert",
					"error", err,
					"budget_id", params.BudgetID,
					"tenant_id", params.TenantID)
			}
		}()
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	result := &ReservationResult{
		ReservationID:   params.RefID,
		BudgetID:        params.BudgetID,
		Amount:          params.Amount,
		Currency:        params.Currency,
		NewBalance:      balance,
		SoftCapExceeded: softCapExceeded,
		Utilization:     (balance / hardCap) * 100,
	}

	s.logger.Info("budget reserved",
		"budget_id", params.BudgetID,
		"amount", params.Amount,
		"currency", params.Currency,
		"ref_id", params.RefID,
		"new_balance", balance,
		"soft_cap_exceeded", softCapExceeded)

	return result, nil
}

// ChargeReservation converts a reservation to a charge (on redemption)
// This is called when a reward is redeemed
func (s *Service) ChargeReservation(ctx context.Context, params ChargeReservationParams) error {
	// Validate parameters
	if err := params.Validate(); err != nil {
		return err
	}

	// Start transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check if already charged
	qtx := s.queries.WithTx(tx)
	entries, err := qtx.GetLedgerEntryByRef(ctx, db.GetLedgerEntryByRefParams{
		TenantID: params.TenantID,
		RefType:  pgtype.Text{String: "issuance", Valid: true},
		RefID:    params.RefID,
	})
	if err != nil {
		return fmt.Errorf("failed to check existing entries: %w", err)
	}

	// Check if already charged
	for _, entry := range entries {
		if entry.EntryType == "charge" {
			return ErrAlreadyCharged
		}
	}

	// Call charge_budget database function
	var success bool
	err = tx.QueryRow(ctx,
		"SELECT charge_budget($1, $2, $3, $4, $5)",
		params.TenantID,
		params.BudgetID,
		params.Amount,
		params.Currency,
		params.RefID,
	).Scan(&success)

	if err != nil {
		return fmt.Errorf("failed to charge budget: %w", err)
	}

	if !success {
		return errors.New("charge budget function returned false")
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("budget charged",
		"budget_id", params.BudgetID,
		"amount", params.Amount,
		"currency", params.Currency,
		"ref_id", params.RefID)

	return nil
}

// ReleaseReservation returns reserved funds (on expiry/cancel)
// This is called when a reward expires or is cancelled before redemption
func (s *Service) ReleaseReservation(ctx context.Context, params ReleaseReservationParams) error {
	// Validate parameters
	if err := params.Validate(); err != nil {
		return err
	}

	// Start transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check if reservation exists and hasn't been released
	qtx := s.queries.WithTx(tx)
	entries, err := qtx.GetLedgerEntryByRef(ctx, db.GetLedgerEntryByRefParams{
		TenantID: params.TenantID,
		RefType:  pgtype.Text{String: "issuance", Valid: true},
		RefID:    params.RefID,
	})
	if err != nil {
		return fmt.Errorf("failed to check existing entries: %w", err)
	}

	if len(entries) == 0 {
		return ErrReservationNotFound
	}

	// Check if already released or charged
	for _, entry := range entries {
		if entry.EntryType == "release" {
			return errors.New("reservation already released")
		}
		if entry.EntryType == "charge" {
			return errors.New("cannot release charged reservation")
		}
	}

	// Call release_budget database function
	var success bool
	err = tx.QueryRow(ctx,
		"SELECT release_budget($1, $2, $3, $4, $5)",
		params.TenantID,
		params.BudgetID,
		params.Amount,
		params.Currency,
		params.RefID,
	).Scan(&success)

	if err != nil {
		return fmt.Errorf("failed to release budget: %w", err)
	}

	if !success {
		return errors.New("release budget function returned false")
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("budget released",
		"budget_id", params.BudgetID,
		"amount", params.Amount,
		"currency", params.Currency,
		"ref_id", params.RefID)

	return nil
}

// ReserveBudgetParams contains parameters for reserving budget
type ReserveBudgetParams struct {
	TenantID pgtype.UUID
	BudgetID pgtype.UUID
	Amount   string // String to avoid floating point precision issues
	Currency string
	RefID    pgtype.UUID // Reference to issuance
}

// Validate validates the reserve budget parameters
func (p ReserveBudgetParams) Validate() error {
	if !p.TenantID.Valid {
		return errors.New("tenant_id is required")
	}
	if !p.BudgetID.Valid {
		return errors.New("budget_id is required")
	}
	if !p.RefID.Valid {
		return errors.New("ref_id is required")
	}
	if p.Amount == "" {
		return ErrInvalidAmount
	}
	if !IsValidCurrency(p.Currency) {
		return ErrCurrencyMismatch
	}
	return nil
}

// ChargeReservationParams contains parameters for charging a reservation
type ChargeReservationParams struct {
	TenantID pgtype.UUID
	BudgetID pgtype.UUID
	Amount   string
	Currency string
	RefID    pgtype.UUID // Reference to issuance
}

// Validate validates the charge reservation parameters
func (p ChargeReservationParams) Validate() error {
	if !p.TenantID.Valid {
		return errors.New("tenant_id is required")
	}
	if !p.BudgetID.Valid {
		return errors.New("budget_id is required")
	}
	if !p.RefID.Valid {
		return errors.New("ref_id is required")
	}
	if p.Amount == "" {
		return ErrInvalidAmount
	}
	if !IsValidCurrency(p.Currency) {
		return ErrCurrencyMismatch
	}
	return nil
}

// ReleaseReservationParams contains parameters for releasing a reservation
type ReleaseReservationParams struct {
	TenantID pgtype.UUID
	BudgetID pgtype.UUID
	Amount   string
	Currency string
	RefID    pgtype.UUID // Reference to issuance
}

// Validate validates the release reservation parameters
func (p ReleaseReservationParams) Validate() error {
	if !p.TenantID.Valid {
		return errors.New("tenant_id is required")
	}
	if !p.BudgetID.Valid {
		return errors.New("budget_id is required")
	}
	if !p.RefID.Valid {
		return errors.New("ref_id is required")
	}
	if p.Amount == "" {
		return ErrInvalidAmount
	}
	if !IsValidCurrency(p.Currency) {
		return ErrCurrencyMismatch
	}
	return nil
}

// ReservationResult contains the result of a budget reservation
type ReservationResult struct {
	ReservationID   pgtype.UUID
	BudgetID        pgtype.UUID
	Amount          string
	Currency        string
	NewBalance      float64
	SoftCapExceeded bool
	Utilization     float64 // Percentage (0-100)
}
