package reward

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// RedeemIssuance redeems an issued reward
// This function:
// 1. Validates the issuance is in issued state
// 2. Verifies the OTP/code if provided
// 3. Checks expiry
// 4. Transitions to redeemed state
// 5. Charges the budget (moves from reserved to charged in ledger)
func (s *Service) RedeemIssuance(ctx context.Context, issuanceID, tenantID pgtype.UUID, code string) error {
	// Start transaction for atomic redemption
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get issuance
	var issuance struct {
		ID         pgtype.UUID
		TenantID   pgtype.UUID
		CustomerID pgtype.UUID
		CampaignID pgtype.UUID
		RewardID   pgtype.UUID
		Status     string
		Code       pgtype.Text
		ExpiresAt  pgtype.Timestamptz
		CostAmount pgtype.Numeric
		Currency   pgtype.Text
	}

	err = tx.QueryRow(ctx, `
		SELECT id, tenant_id, customer_id, campaign_id, reward_id, status,
		       code, expires_at, cost_amount, currency
		FROM issuances
		WHERE id = $1 AND tenant_id = $2
		FOR UPDATE
	`, issuanceID, tenantID).Scan(
		&issuance.ID,
		&issuance.TenantID,
		&issuance.CustomerID,
		&issuance.CampaignID,
		&issuance.RewardID,
		&issuance.Status,
		&issuance.Code,
		&issuance.ExpiresAt,
		&issuance.CostAmount,
		&issuance.Currency,
	)
	if err != nil {
		return fmt.Errorf("failed to get issuance: %w", err)
	}

	// Validate state
	currentState := State(issuance.Status)
	if currentState != StateIssued {
		return fmt.Errorf("cannot redeem issuance in state: %s (must be issued)", currentState)
	}

	// Verify code if provided and if issuance has a code
	if code != "" && issuance.Code.Valid {
		// Normalize codes for comparison (case-insensitive, trim whitespace)
		normalizedProvided := strings.ToUpper(strings.TrimSpace(code))
		normalizedStored := strings.ToUpper(strings.TrimSpace(issuance.Code.String))

		if normalizedProvided != normalizedStored {
			return fmt.Errorf("invalid redemption code")
		}
	}

	// Check expiry
	if issuance.ExpiresAt.Valid {
		if time.Now().After(issuance.ExpiresAt.Time) {
			// Mark as expired
			_ = s.updateStateInTx(ctx, tx, issuanceID, tenantID, StateIssued, StateExpired)
			return fmt.Errorf("reward has expired")
		}
	}

	// Transition to redeemed state
	err = s.updateStateInTx(ctx, tx, issuanceID, tenantID, StateIssued, StateRedeemed)
	if err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	// Charge the budget by calling the charge_budget database function
	// This moves the ledger entry from 'reserve' to 'charge'
	err = s.chargeBudget(ctx, tx, issuance.CampaignID, issuance.ID, issuance.CostAmount)
	if err != nil {
		return fmt.Errorf("failed to charge budget: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// chargeBudget charges the budget for a redeemed issuance
// This uses the charge_budget database function to atomically:
// 1. Find the reserve ledger entry
// 2. Create a charge ledger entry
// 3. Update the budget balance
func (s *Service) chargeBudget(ctx context.Context, tx pgx.Tx, campaignID, issuanceID pgtype.UUID, amount pgtype.Numeric) error {
	// Get the budget ID from the campaign
	var budgetID pgtype.UUID
	var tenantID pgtype.UUID

	err := tx.QueryRow(ctx, `
		SELECT budget_id, tenant_id
		FROM campaigns
		WHERE id = $1
	`, campaignID).Scan(&budgetID, &tenantID)

	if err != nil {
		return fmt.Errorf("failed to get campaign budget: %w", err)
	}

	// Call the charge_budget function
	// This function expects (p_tenant_id, p_budget_id, p_ref_type, p_ref_id)
	_, err = tx.Exec(ctx, `
		SELECT charge_budget($1::uuid, $2::uuid, 'issuance'::text, $3::uuid)
	`, tenantID, budgetID, issuanceID)

	if err != nil {
		return fmt.Errorf("charge_budget function failed: %w", err)
	}

	return nil
}

// VerifyRedemptionCode checks if a code is valid for redemption without actually redeeming
// Useful for preview/validation before final redemption
func (s *Service) VerifyRedemptionCode(ctx context.Context, issuanceID, tenantID pgtype.UUID, code string) (bool, error) {
	issuance, err := s.queries.GetIssuanceByID(ctx, db.GetIssuanceByIDParams{
		ID:       issuanceID,
		TenantID: tenantID,
	})
	if err != nil {
		return false, fmt.Errorf("failed to get issuance: %w", err)
	}

	// Check state
	if issuance.Status != string(StateIssued) {
		return false, nil
	}

	// Check expiry
	if issuance.ExpiresAt.Valid && time.Now().After(issuance.ExpiresAt.Time) {
		return false, nil
	}

	// Verify code
	if !issuance.Code.Valid {
		// No code required
		return true, nil
	}

	normalizedProvided := strings.ToUpper(strings.TrimSpace(code))
	normalizedStored := strings.ToUpper(strings.TrimSpace(issuance.Code.String))

	return normalizedProvided == normalizedStored, nil
}
