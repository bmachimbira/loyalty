package reward

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ExpireOldIssuances finds and expires issuances that have passed their expiry date
// This should be run as a background job (e.g., hourly cron)
// It:
// 1. Finds all issued issuances where expires_at < NOW()
// 2. Transitions them to expired state
// 3. Releases their budget reservations
func (s *Service) ExpireOldIssuances(ctx context.Context) error {
	log.Println("Starting expiry worker...")

	// Start transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Query for expired issuances
	rows, err := tx.Query(ctx, `
		SELECT i.id, i.tenant_id, i.campaign_id, i.cost_amount
		FROM issuances i
		WHERE i.status = 'issued'
		  AND i.expires_at IS NOT NULL
		  AND i.expires_at < NOW()
		FOR UPDATE SKIP LOCKED
	`)
	if err != nil {
		return fmt.Errorf("failed to query expired issuances: %w", err)
	}
	defer rows.Close()

	expiredCount := 0
	errorCount := 0

	for rows.Next() {
		var issuanceID, tenantID, campaignID pgtype.UUID
		var costAmount pgtype.Numeric

		if err := rows.Scan(&issuanceID, &tenantID, &campaignID, &costAmount); err != nil {
			log.Printf("Error scanning expired issuance: %v", err)
			errorCount++
			continue
		}

		// Transition to expired state
		err = s.updateStateInTx(ctx, tx, issuanceID, tenantID, StateIssued, StateExpired)
		if err != nil {
			log.Printf("Failed to expire issuance %s: %v", issuanceID, err)
			errorCount++
			continue
		}

		// Release the budget reservation
		err = s.releaseBudget(ctx, tx, campaignID, issuanceID)
		if err != nil {
			log.Printf("Failed to release budget for issuance %s: %v", issuanceID, err)
			errorCount++
			continue
		}

		expiredCount++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating expired issuances: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Expiry worker completed: %d expired, %d errors", expiredCount, errorCount)
	return nil
}

// releaseBudget releases a budget reservation for an expired or cancelled issuance
// This uses the release_budget database function to atomically:
// 1. Find the reserve ledger entry
// 2. Create a release ledger entry
// 3. Update the budget balance
func (s *Service) releaseBudget(ctx context.Context, tx pgx.Tx, campaignID, issuanceID pgtype.UUID) error {
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

	// Call the release_budget function
	// This function expects (p_tenant_id, p_budget_id, p_ref_type, p_ref_id)
	_, err = tx.Exec(ctx, `
		SELECT release_budget($1::uuid, $2::uuid, 'issuance'::text, $3::uuid)
	`, tenantID, budgetID, issuanceID)

	if err != nil {
		return fmt.Errorf("release_budget function failed: %w", err)
	}

	return nil
}

// RunExpiryWorker runs the expiry worker on a schedule
// This is a blocking function that should be run in a goroutine
func (s *Service) RunExpiryWorker(ctx context.Context, interval time.Duration) {
	log.Printf("Starting expiry worker with interval: %v", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run immediately on startup
	if err := s.ExpireOldIssuances(ctx); err != nil {
		log.Printf("Expiry worker error: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Expiry worker shutting down")
			return
		case <-ticker.C:
			if err := s.ExpireOldIssuances(ctx); err != nil {
				log.Printf("Expiry worker error: %v", err)
			}
		}
	}
}

// ExpireIssuance manually expires a single issuance
// Useful for testing or manual intervention
func (s *Service) ExpireIssuance(ctx context.Context, issuanceID, tenantID pgtype.UUID) error {
	// Start transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get issuance
	var campaignID pgtype.UUID
	var status string

	err = tx.QueryRow(ctx, `
		SELECT campaign_id, status
		FROM issuances
		WHERE id = $1 AND tenant_id = $2
		FOR UPDATE
	`, issuanceID, tenantID).Scan(&campaignID, &status)

	if err != nil {
		return fmt.Errorf("failed to get issuance: %w", err)
	}

	// Can only expire issued issuances
	if status != string(StateIssued) {
		return fmt.Errorf("cannot expire issuance in state: %s", status)
	}

	// Transition to expired
	err = s.updateStateInTx(ctx, tx, issuanceID, tenantID, StateIssued, StateExpired)
	if err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	// Release budget
	err = s.releaseBudget(ctx, tx, campaignID, issuanceID)
	if err != nil {
		return fmt.Errorf("failed to release budget: %w", err)
	}

	// Commit
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
