package budget

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/bmachimbira/loyalty/api/internal/db"
)

// AlertType represents the type of budget alert
type AlertType string

const (
	// AlertTypeSoftCap is triggered when balance exceeds soft cap
	AlertTypeSoftCap AlertType = "soft_cap_exceeded"

	// AlertTypeHardCap is triggered when balance approaches hard cap
	AlertTypeHardCap AlertType = "hard_cap_approaching"

	// AlertTypeHardCapReached is triggered when a reservation is rejected due to hard cap
	AlertTypeHardCapReached AlertType = "hard_cap_reached"
)

// AlertLevel represents the severity of an alert
type AlertLevel string

const (
	// AlertLevelWarning is for soft cap alerts
	AlertLevelWarning AlertLevel = "warning"

	// AlertLevelCritical is for hard cap alerts
	AlertLevelCritical AlertLevel = "critical"
)

// Alert represents a budget alert
type Alert struct {
	Type            AlertType
	Level           AlertLevel
	BudgetID        pgtype.UUID
	BudgetName      string
	TenantID        pgtype.UUID
	Balance         float64
	SoftCap         float64
	HardCap         float64
	Utilization     float64 // Percentage (0-100)
	Message         string
	Timestamp       string
}

// AlertThresholds contains configurable thresholds for alerts
type AlertThresholds struct {
	SoftCapPercent float64 // Default: 80% - trigger when balance > soft_cap
	HardCapPercent float64 // Default: 95% - trigger warning when balance > 95% of hard_cap
}

// DefaultAlertThresholds returns the default alert thresholds
func DefaultAlertThresholds() AlertThresholds {
	return AlertThresholds{
		SoftCapPercent: 80.0,  // Alert when balance > soft cap
		HardCapPercent: 95.0,  // Alert when balance > 95% of hard cap
	}
}

// CheckSoftCapAlert checks if a budget has exceeded its soft cap and triggers an alert
func (s *Service) CheckSoftCapAlert(ctx context.Context, tenantID, budgetID pgtype.UUID) error {
	if !tenantID.Valid || !budgetID.Valid {
		return errors.New("tenant_id and budget_id are required")
	}

	// Get budget
	budget, err := s.queries.GetBudgetByID(ctx, db.GetBudgetByIDParams{
		ID:       budgetID,
		TenantID: tenantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBudgetNotFound
		}
		return fmt.Errorf("failed to get budget: %w", err)
	}

	// Convert numeric values
	balanceVal, err := budget.Balance.Float64Value()
	if err != nil {
		return fmt.Errorf("invalid balance value: %w", err)
	}
	balance := balanceVal.Float64

	softCapVal, err := budget.SoftCap.Float64Value()
	if err != nil {
		return fmt.Errorf("invalid soft cap value: %w", err)
	}
	softCap := softCapVal.Float64

	hardCapVal, err := budget.HardCap.Float64Value()
	if err != nil {
		return fmt.Errorf("invalid hard cap value: %w", err)
	}
	hardCap := hardCapVal.Float64

	// Check if soft cap exceeded
	if balance > softCap {
		utilization := (balance / hardCap) * 100

		alert := Alert{
			Type:        AlertTypeSoftCap,
			Level:       AlertLevelWarning,
			BudgetID:    budgetID,
			BudgetName:  budget.Name,
			TenantID:    tenantID,
			Balance:     balance,
			SoftCap:     softCap,
			HardCap:     hardCap,
			Utilization: utilization,
			Message: fmt.Sprintf(
				"Budget '%s' has exceeded soft cap. Balance: %.2f %s, Soft Cap: %.2f %s (%.1f%% utilized)",
				budget.Name, balance, budget.Currency, softCap, budget.Currency, utilization,
			),
		}

		return s.deliverAlert(ctx, alert)
	}

	return nil
}

// CheckHardCapAlert checks if a budget is approaching its hard cap
func (s *Service) CheckHardCapAlert(ctx context.Context, tenantID, budgetID pgtype.UUID) error {
	if !tenantID.Valid || !budgetID.Valid {
		return errors.New("tenant_id and budget_id are required")
	}

	thresholds := DefaultAlertThresholds()

	// Get budget
	budget, err := s.queries.GetBudgetByID(ctx, db.GetBudgetByIDParams{
		ID:       budgetID,
		TenantID: tenantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBudgetNotFound
		}
		return fmt.Errorf("failed to get budget: %w", err)
	}

	// Convert numeric values
	balanceVal, err := budget.Balance.Float64Value()
	if err != nil {
		return fmt.Errorf("invalid balance value: %w", err)
	}
	balance := balanceVal.Float64

	hardCapVal, err := budget.HardCap.Float64Value()
	if err != nil {
		return fmt.Errorf("invalid hard cap value: %w", err)
	}
	hardCap := hardCapVal.Float64

	utilization := (balance / hardCap) * 100

	// Check if approaching hard cap (>95% by default)
	if utilization >= thresholds.HardCapPercent {
		softCapVal, err := budget.SoftCap.Float64Value()
		if err != nil {
			return fmt.Errorf("invalid soft cap value: %w", err)
		}
		softCap := softCapVal.Float64

		alert := Alert{
			Type:        AlertTypeHardCap,
			Level:       AlertLevelCritical,
			BudgetID:    budgetID,
			BudgetName:  budget.Name,
			TenantID:    tenantID,
			Balance:     balance,
			SoftCap:     softCap,
			HardCap:     hardCap,
			Utilization: utilization,
			Message: fmt.Sprintf(
				"CRITICAL: Budget '%s' is approaching hard cap. Balance: %.2f %s, Hard Cap: %.2f %s (%.1f%% utilized)",
				budget.Name, balance, budget.Currency, hardCap, budget.Currency, utilization,
			),
		}

		return s.deliverAlert(ctx, alert)
	}

	return nil
}

// TriggerHardCapReachedAlert is called when a reservation is rejected due to hard cap
func (s *Service) TriggerHardCapReachedAlert(ctx context.Context, tenantID, budgetID pgtype.UUID, attemptedAmount float64) error {
	if !tenantID.Valid || !budgetID.Valid {
		return errors.New("tenant_id and budget_id are required")
	}

	// Get budget
	budget, err := s.queries.GetBudgetByID(ctx, db.GetBudgetByIDParams{
		ID:       budgetID,
		TenantID: tenantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBudgetNotFound
		}
		return fmt.Errorf("failed to get budget: %w", err)
	}

	// Convert numeric values
	balanceVal, err := budget.Balance.Float64Value()
	if err != nil {
		return fmt.Errorf("invalid balance value: %w", err)
	}
	balance := balanceVal.Float64

	softCapVal, err := budget.SoftCap.Float64Value()
	if err != nil {
		return fmt.Errorf("invalid soft cap value: %w", err)
	}
	softCap := softCapVal.Float64

	hardCapVal, err := budget.HardCap.Float64Value()
	if err != nil {
		return fmt.Errorf("invalid hard cap value: %w", err)
	}
	hardCap := hardCapVal.Float64

	utilization := (balance / hardCap) * 100

	alert := Alert{
		Type:        AlertTypeHardCapReached,
		Level:       AlertLevelCritical,
		BudgetID:    budgetID,
		BudgetName:  budget.Name,
		TenantID:    tenantID,
		Balance:     balance,
		SoftCap:     softCap,
		HardCap:     hardCap,
		Utilization: utilization,
		Message: fmt.Sprintf(
			"CRITICAL: Budget '%s' hard cap reached. Reservation rejected. "+
				"Attempted: %.2f %s, Balance: %.2f %s, Hard Cap: %.2f %s (%.1f%% utilized)",
			budget.Name, attemptedAmount, budget.Currency, balance, budget.Currency,
			hardCap, budget.Currency, utilization,
		),
	}

	return s.deliverAlert(ctx, alert)
}

// deliverAlert delivers an alert through configured channels
// For Phase 2, this logs the alert. Phase 4 will add webhook delivery.
func (s *Service) deliverAlert(ctx context.Context, alert Alert) error {
	// Log the alert
	logFunc := s.logger.Warn
	if alert.Level == AlertLevelCritical {
		logFunc = s.logger.Error
	}

	logFunc("budget alert triggered",
		"type", alert.Type,
		"level", alert.Level,
		"budget_id", alert.BudgetID,
		"budget_name", alert.BudgetName,
		"tenant_id", alert.TenantID,
		"balance", alert.Balance,
		"soft_cap", alert.SoftCap,
		"hard_cap", alert.HardCap,
		"utilization", alert.Utilization,
		"message", alert.Message)

	// TODO: Phase 4 - Send webhook notification to tenant
	// This would call the webhook service to notify the tenant
	// about budget alerts via their configured webhook endpoints

	return nil
}

// GetBudgetUtilization returns the current utilization of a budget
func (s *Service) GetBudgetUtilization(ctx context.Context, tenantID, budgetID pgtype.UUID) (*BudgetUtilization, error) {
	if !tenantID.Valid || !budgetID.Valid {
		return nil, errors.New("tenant_id and budget_id are required")
	}

	// Get budget
	budget, err := s.queries.GetBudgetByID(ctx, db.GetBudgetByIDParams{
		ID:       budgetID,
		TenantID: tenantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBudgetNotFound
		}
		return nil, fmt.Errorf("failed to get budget: %w", err)
	}

	// Convert numeric values
	balanceVal, err := budget.Balance.Float64Value()
	if err != nil {
		return nil, fmt.Errorf("invalid balance value: %w", err)
	}
	balance := balanceVal.Float64

	softCapVal, err := budget.SoftCap.Float64Value()
	if err != nil {
		return nil, fmt.Errorf("invalid soft cap value: %w", err)
	}
	softCap := softCapVal.Float64

	hardCapVal, err := budget.HardCap.Float64Value()
	if err != nil {
		return nil, fmt.Errorf("invalid hard cap value: %w", err)
	}
	hardCap := hardCapVal.Float64

	available := hardCap - balance
	utilization := 0.0
	if hardCap > 0 {
		utilization = (balance / hardCap) * 100
	}

	return &BudgetUtilization{
		BudgetID:           budgetID,
		BudgetName:         budget.Name,
		Currency:           budget.Currency,
		Balance:            balance,
		Available:          available,
		SoftCap:            softCap,
		HardCap:            hardCap,
		Utilization:        utilization,
		SoftCapExceeded:    balance > softCap,
		HardCapApproaching: utilization >= 95.0,
	}, nil
}

// BudgetUtilization contains budget utilization information
type BudgetUtilization struct {
	BudgetID           pgtype.UUID
	BudgetName         string
	Currency           string
	Balance            float64 // Currently reserved
	Available          float64 // Available for new reservations
	SoftCap            float64
	HardCap            float64
	Utilization        float64 // Percentage (0-100)
	SoftCapExceeded    bool
	HardCapApproaching bool
}
