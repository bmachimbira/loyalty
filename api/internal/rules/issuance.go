package rules

import (
	"context"
	"fmt"
	"hash/fnv"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// issueReward creates a new issuance for a triggered rule
// Uses PostgreSQL advisory locks to prevent race conditions
func (e *Engine) issueReward(ctx context.Context, rule db.Rule, event db.Event) (*db.Issuance, error) {
	// Start transaction
	tx, err := e.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Create queries with transaction
	qtx := e.queries.WithTx(tx)

	// Acquire advisory lock on (tenant_id, rule_id, customer_id)
	// This prevents concurrent issuances for the same rule and customer
	lockKey := hashLock(event.TenantID.Bytes[:], rule.ID.Bytes[:], event.CustomerID.Bytes[:])
	_, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockKey)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire advisory lock: %w", err)
	}

	// Re-check caps inside transaction (protection against race conditions)
	passed, err := e.checkCaps(ctx, rule, event)
	if err != nil {
		return nil, fmt.Errorf("cap check failed in transaction: %w", err)
	}
	if !passed {
		return nil, fmt.Errorf("cap check failed: limits exceeded")
	}

	// Get reward details
	reward, err := qtx.GetRewardByID(ctx, db.GetRewardByIDParams{
		TenantID: event.TenantID,
		ID:       rule.RewardID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get reward: %w", err)
	}

	// Check and reserve budget if campaign has one
	if rule.CampaignID.Valid {
		campaign, err := qtx.GetCampaignByID(ctx, db.GetCampaignByIDParams{
			TenantID: event.TenantID,
			ID:       rule.CampaignID.UUID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get campaign: %w", err)
		}

		// If campaign has a budget, reserve funds
		if campaign.BudgetID.Valid {
			success, err := e.reserveBudget(ctx, tx, campaign.BudgetID.UUID, event.TenantID, reward.FaceValue)
			if err != nil {
				return nil, fmt.Errorf("failed to reserve budget: %w", err)
			}
			if !success {
				return nil, fmt.Errorf("budget capacity exceeded")
			}
		}
	}

	// Create issuance in 'reserved' state
	var currency pgtype.Text
	if reward.Currency.Valid {
		currency = reward.Currency
	} else {
		currency.String = "USD"
		currency.Valid = true
	}

	issuance, err := qtx.ReserveIssuance(ctx, db.ReserveIssuanceParams{
		TenantID:   event.TenantID,
		CustomerID: event.CustomerID,
		CampaignID: rule.CampaignID,
		RewardID:   reward.ID,
		Currency:   currency,
		FaceAmount: reward.FaceValue,
		CostAmount: reward.FaceValue,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create issuance: %w", err)
	}

	// Update issuance with event_id
	// Note: The ReserveIssuance query doesn't include event_id, so we'll skip this for now
	// In a production system, you might want to add event_id to the issuance

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Note: In a full implementation, you would trigger async processing here
	// to transition the issuance from 'reserved' to 'issued' state
	// For now, we'll return the reserved issuance

	return &issuance, nil
}

// reserveBudget reserves budget for an issuance
func (e *Engine) reserveBudget(ctx context.Context, tx pgx.Tx, budgetID, tenantID pgtype.UUID, amount pgtype.Numeric) (bool, error) {
	// Call database function to reserve budget
	query := `SELECT reserve_budget($1, $2, $3, $4, $5)`

	// We need to generate a temporary issuance ID for the reference
	// In a real implementation, this would be the actual issuance ID
	var tempRefID pgtype.UUID
	tempRefID.Scan("00000000-0000-0000-0000-000000000000")

	var success bool
	err := tx.QueryRow(ctx, query, tenantID, budgetID, amount, "USD", tempRefID).Scan(&success)
	if err != nil {
		return false, fmt.Errorf("reserve_budget function failed: %w", err)
	}

	return success, nil
}

// hashLock generates a consistent int64 hash for advisory locking
func hashLock(parts ...[]byte) int64 {
	h := fnv.New64a()
	for _, part := range parts {
		h.Write(part)
	}
	// Convert to int64 (PostgreSQL advisory locks use bigint)
	return int64(h.Sum64() & 0x7FFFFFFFFFFFFFFF) // Ensure positive
}

// IssuanceResult represents the result of processing an issuance
type IssuanceResult struct {
	Issuance *db.Issuance
	Rule     *db.Rule
	Reward   *db.RewardCatalog
	Error    error
}
