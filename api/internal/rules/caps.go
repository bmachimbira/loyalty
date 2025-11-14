package rules

import (
	"context"
	"fmt"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

// checkCaps verifies all cap constraints for a rule before issuance
// Returns true if all checks pass, false otherwise
func (e *Engine) checkCaps(ctx context.Context, rule db.Rule, event db.Event) (bool, error) {
	// Check per-user cap
	if rule.PerUserCap > 0 {
		passed, err := e.checkPerUserCap(ctx, rule, event)
		if err != nil {
			return false, fmt.Errorf("per-user cap check failed: %w", err)
		}
		if !passed {
			return false, nil
		}
	}

	// Check global cap
	if rule.GlobalCap.Valid && rule.GlobalCap.Int32 > 0 {
		passed, err := e.checkGlobalCap(ctx, rule, event)
		if err != nil {
			return false, fmt.Errorf("global cap check failed: %w", err)
		}
		if !passed {
			return false, nil
		}
	}

	// Check cooldown period
	if rule.CoolDownSec > 0 {
		passed, err := e.checkCooldown(ctx, rule, event)
		if err != nil {
			return false, fmt.Errorf("cooldown check failed: %w", err)
		}
		if !passed {
			return false, nil
		}
	}

	return true, nil
}

// checkPerUserCap verifies the per-user issuance cap
func (e *Engine) checkPerUserCap(ctx context.Context, rule db.Rule, event db.Event) (bool, error) {
	// Call database function to get customer issuance count
	query := `SELECT get_customer_rule_issuance_count($1, $2, $3)`

	var count int64
	err := e.pool.QueryRow(ctx, query, event.TenantID, event.CustomerID, rule.ID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to get customer issuance count: %w", err)
	}

	// Check if count is below cap
	return count < int64(rule.PerUserCap), nil
}

// checkGlobalCap verifies the global issuance cap for the rule
func (e *Engine) checkGlobalCap(ctx context.Context, rule db.Rule, event db.Event) (bool, error) {
	// Call database function to get global issuance count
	query := `SELECT get_rule_global_issuance_count($1, $2)`

	var count int64
	err := e.pool.QueryRow(ctx, query, event.TenantID, rule.ID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to get global issuance count: %w", err)
	}

	// Check if count is below cap
	return count < int64(rule.GlobalCap.Int32), nil
}

// checkCooldown verifies the customer is not within the cooldown period
func (e *Engine) checkCooldown(ctx context.Context, rule db.Rule, event db.Event) (bool, error) {
	// Call database function to check if within cooldown
	query := `SELECT is_within_cooldown($1, $2, $3, $4)`

	var withinCooldown bool
	err := e.pool.QueryRow(ctx, query, event.TenantID, event.CustomerID, rule.ID, rule.CoolDownSec).Scan(&withinCooldown)
	if err != nil {
		return false, fmt.Errorf("failed to check cooldown: %w", err)
	}

	// If within cooldown, cap check fails
	return !withinCooldown, nil
}

// getCampaignByID retrieves campaign details
func (e *Engine) getCampaignByID(ctx context.Context, tenantID, campaignID pgtype.UUID) (*db.Campaign, error) {
	campaign, err := e.queries.GetCampaignByID(ctx, db.GetCampaignByIDParams{
		TenantID: tenantID,
		ID:       campaignID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get campaign: %w", err)
	}
	return &campaign, nil
}

// checkBudgetCapacity verifies budget has capacity for the reward
func (e *Engine) checkBudgetCapacity(ctx context.Context, budgetID pgtype.UUID, amount pgtype.Numeric) (bool, error) {
	// Call database function to check budget capacity
	query := `SELECT check_budget_capacity($1, $2)`

	var hasCapacity bool
	err := e.pool.QueryRow(ctx, query, budgetID, amount).Scan(&hasCapacity)
	if err != nil {
		return false, fmt.Errorf("failed to check budget capacity: %w", err)
	}

	return hasCapacity, nil
}
