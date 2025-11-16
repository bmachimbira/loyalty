package reward

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/bmachimbira/loyalty/api/internal/reward/handlers"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service handles reward issuance and state management
type Service struct {
	pool     *pgxpool.Pool
	queries  *db.Queries
	handlers map[string]handlers.RewardHandler
}

// NewService creates a new reward service with all handlers registered
func NewService(pool *pgxpool.Pool, queries *db.Queries) *Service {
	s := &Service{
		pool:     pool,
		queries:  queries,
		handlers: make(map[string]handlers.RewardHandler),
	}

	// Register all reward type handlers
	s.RegisterHandler("discount", &handlers.DiscountHandler{})
	s.RegisterHandler("voucher_code", handlers.NewVoucherCodeHandler(queries))
	s.RegisterHandler("external_voucher", handlers.NewExternalVoucherHandler())
	s.RegisterHandler("points_credit", handlers.NewPointsCreditHandler())
	s.RegisterHandler("physical_item", handlers.NewPhysicalItemHandler())
	s.RegisterHandler("webhook_custom", handlers.NewWebhookHandler())

	return s
}

// RegisterHandler registers a reward type handler
func (s *Service) RegisterHandler(rewardType string, handler handlers.RewardHandler) {
	s.handlers[rewardType] = handler
}

// GetHandler returns the handler for a reward type
func (s *Service) GetHandler(rewardType string) (handlers.RewardHandler, error) {
	handler, ok := s.handlers[rewardType]
	if !ok {
		return nil, fmt.Errorf("unknown reward type: %s", rewardType)
	}
	return handler, nil
}

// ProcessIssuance processes a reserved issuance, moving it from reserved → issued
// This is the main orchestration function that:
// 1. Validates the issuance is in reserved state
// 2. Gets the appropriate handler for the reward type
// 3. Calls the handler to process the reward
// 4. Updates the issuance with the result
// 5. Transitions to issued state
func (s *Service) ProcessIssuance(ctx context.Context, issuanceID pgtype.UUID) error {
	// Start a transaction for atomic processing
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	txQueries := s.queries.WithTx(tx)

	// Get the issuance with the tenant ID from context
	// In production, tenant_id would come from the context set by middleware
	// For now, we'll get it from the issuance itself
	var issuance db.Issuance
	err = tx.QueryRow(ctx, `
		SELECT id, tenant_id, customer_id, campaign_id, reward_id, status,
		       code, external_ref, currency, cost_amount, face_amount,
		       issued_at, expires_at, redeemed_at, event_id
		FROM issuances
		WHERE id = $1
	`, issuanceID).Scan(
		&issuance.ID,
		&issuance.TenantID,
		&issuance.CustomerID,
		&issuance.CampaignID,
		&issuance.RewardID,
		&issuance.Status,
		&issuance.Code,
		&issuance.ExternalRef,
		&issuance.Currency,
		&issuance.CostAmount,
		&issuance.FaceAmount,
		&issuance.IssuedAt,
		&issuance.ExpiresAt,
		&issuance.RedeemedAt,
		&issuance.EventID,
	)
	if err != nil {
		return fmt.Errorf("failed to get issuance: %w", err)
	}

	// Validate state
	currentState := State(issuance.Status)
	if currentState != StateReserved {
		return fmt.Errorf("invalid state for processing: %s (expected %s)", currentState, StateReserved)
	}

	// Get the reward details
	reward, err := txQueries.GetRewardByID(ctx, db.GetRewardByIDParams{
		ID:       issuance.RewardID,
		TenantID: issuance.TenantID,
	})
	if err != nil {
		return fmt.Errorf("failed to get reward: %w", err)
	}

	// Get the handler for this reward type
	handler, err := s.GetHandler(reward.Type)
	if err != nil {
		// Mark as failed if handler not found
		_ = s.updateStateInTx(ctx, tx, issuance.ID, issuance.TenantID, StateReserved, StateFailed)
		return err
	}

	// Process the reward
	result, err := handler.Process(ctx, &issuance, &reward)
	if err != nil {
		// Mark as failed if processing fails
		log.Printf("Failed to process issuance %s: %v", issuance.ID, err)
		_ = s.updateStateInTx(ctx, tx, issuance.ID, issuance.TenantID, StateReserved, StateFailed)
		return fmt.Errorf("reward processing failed: %w", err)
	}

	// Update issuance with result
	err = s.updateIssuanceWithResult(ctx, tx, issuance.ID, issuance.TenantID, result)
	if err != nil {
		return fmt.Errorf("failed to update issuance: %w", err)
	}

	// Transition to issued state
	err = s.updateStateInTx(ctx, tx, issuance.ID, issuance.TenantID, StateReserved, StateIssued)
	if err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Successfully processed issuance %s: %s -> %s", issuance.ID, StateReserved, StateIssued)
	return nil
}

// updateIssuanceWithResult updates the issuance record with processing results
func (s *Service) updateIssuanceWithResult(ctx context.Context, tx pgx.Tx, issuanceID, tenantID pgtype.UUID, result *handlers.ProcessResult) error {
	// Prepare code field
	var code pgtype.Text
	if result.Code != "" {
		code = pgtype.Text{String: result.Code, Valid: true}
	}

	// Prepare external_ref field
	var externalRef pgtype.Text
	if result.ExternalRef != "" {
		externalRef = pgtype.Text{String: result.ExternalRef, Valid: true}
	}

	// Prepare expires_at field
	var expiresAt pgtype.Timestamptz
	if result.ExpiresAt != nil {
		expiresAt = pgtype.Timestamptz{Time: *result.ExpiresAt, Valid: true}
	}

	// Update issuance
	_, err := tx.Exec(ctx, `
		UPDATE issuances
		SET code = $3,
		    external_ref = $4,
		    expires_at = $5
		WHERE id = $1 AND tenant_id = $2
	`, issuanceID, tenantID, code, externalRef, expiresAt)

	return err
}

// updateState transitions an issuance to a new state with validation
func (s *Service) updateState(ctx context.Context, issuanceID, tenantID pgtype.UUID, fromState, toState State) error {
	// Start transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	err = s.updateStateInTx(ctx, tx, issuanceID, tenantID, fromState, toState)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// updateStateInTx transitions an issuance to a new state within an existing transaction
func (s *Service) updateStateInTx(ctx context.Context, tx pgx.Tx, issuanceID, tenantID pgtype.UUID, fromState, toState State) error {
	// Validate transition
	if err := fromState.ValidateTransition(toState); err != nil {
		return err
	}

	// Update status
	result, err := tx.Exec(ctx, `
		UPDATE issuances
		SET status = $3,
		    redeemed_at = CASE WHEN $3 = 'redeemed' THEN NOW() ELSE redeemed_at END
		WHERE id = $1 AND tenant_id = $2 AND status = $4
	`, issuanceID, tenantID, toState.String(), fromState.String())

	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Check if any rows were affected
	if result.RowsAffected() == 0 {
		return fmt.Errorf("issuance not found or state mismatch")
	}

	return nil
}

// CancelIssuance cancels an issuance and releases its budget
func (s *Service) CancelIssuance(ctx context.Context, issuanceID, tenantID pgtype.UUID) error {
	// Get current issuance
	issuance, err := s.queries.GetIssuanceByID(ctx, db.GetIssuanceByIDParams{
		ID:       issuanceID,
		TenantID: tenantID,
	})
	if err != nil {
		return fmt.Errorf("failed to get issuance: %w", err)
	}

	currentState := State(issuance.Status)

	// Can only cancel from reserved or issued states
	if currentState != StateReserved && currentState != StateIssued {
		return fmt.Errorf("cannot cancel issuance in state: %s", currentState)
	}

	// Update state to cancelled
	return s.updateState(ctx, issuanceID, tenantID, currentState, StateCancelled)
}

// GetIssuance retrieves an issuance by ID
func (s *Service) GetIssuance(ctx context.Context, issuanceID, tenantID pgtype.UUID) (*IssuanceDetails, error) {
	issuance, err := s.queries.GetIssuanceByID(ctx, db.GetIssuanceByIDParams{
		ID:       issuanceID,
		TenantID: tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get issuance: %w", err)
	}

	// Get reward details
	reward, err := s.queries.GetRewardByID(ctx, db.GetRewardByIDParams{
		ID:       issuance.RewardID,
		TenantID: tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get reward: %w", err)
	}

	details := &IssuanceDetails{
		Issuance: issuance,
		Reward:   reward,
	}

	return details, nil
}

// IssuanceDetails contains issuance and related reward information
type IssuanceDetails struct {
	Issuance db.Issuance
	Reward   db.RewardCatalog
}

// MarshalJSON provides custom JSON serialization
func (d *IssuanceDetails) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"id":           d.Issuance.ID,
		"customer_id":  d.Issuance.CustomerID,
		"campaign_id":  d.Issuance.CampaignID,
		"reward_id":    d.Issuance.RewardID,
		"status":       d.Issuance.Status,
		"code":         d.Issuance.Code,
		"external_ref": d.Issuance.ExternalRef,
		"currency":     d.Issuance.Currency,
		"cost_amount":  d.Issuance.CostAmount,
		"face_amount":  d.Issuance.FaceAmount,
		"issued_at":    d.Issuance.IssuedAt,
		"expires_at":   d.Issuance.ExpiresAt,
		"redeemed_at":  d.Issuance.RedeemedAt,
		"reward": map[string]interface{}{
			"name": d.Reward.Name,
			"type": d.Reward.Type,
		},
	})
}
