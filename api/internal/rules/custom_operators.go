package rules

import (
	"context"
	"time"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CustomOperators implements custom JsonLogic operators
type CustomOperators struct {
	pool *pgxpool.Pool
}

// NewCustomOperators creates a new custom operators instance
func NewCustomOperators(pool *pgxpool.Pool) *CustomOperators {
	return &CustomOperators{
		pool: pool,
	}
}

// NthEventInPeriod checks if this is the Nth occurrence of an event in a period
// Returns true if the count of events in the last periodDays equals n
func (c *CustomOperators) NthEventInPeriod(
	ctx context.Context,
	tenantID, customerID string,
	eventType string,
	n, periodDays int,
) (bool, error) {
	// Parse UUIDs
	var tenantUUID, customerUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		return false, err
	}
	if err := customerUUID.Scan(customerID); err != nil {
		return false, err
	}

	// Calculate cutoff date
	cutoff := time.Now().AddDate(0, 0, -periodDays)

	// Count events in the period
	count, err := c.countEventsSince(ctx, tenantUUID, customerUUID, eventType, cutoff)
	if err != nil {
		return false, err
	}

	return count == int64(n), nil
}

// DistinctVisitDays counts the number of distinct days with visit events
func (c *CustomOperators) DistinctVisitDays(
	ctx context.Context,
	tenantID, customerID string,
	periodDays int,
) (int, error) {
	// Parse UUIDs
	var tenantUUID, customerUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		return 0, err
	}
	if err := customerUUID.Scan(customerID); err != nil {
		return 0, err
	}

	// Calculate cutoff date
	cutoff := time.Now().AddDate(0, 0, -periodDays)

	// Count distinct days with visit events
	count, err := c.countDistinctDays(ctx, tenantUUID, customerUUID, "visit", cutoff)
	if err != nil {
		return 0, err
	}

	return int(count), nil
}

// countEventsSince counts events of a type since a cutoff date
func (c *CustomOperators) countEventsSince(
	ctx context.Context,
	tenantID, customerID pgtype.UUID,
	eventType string,
	since time.Time,
) (int64, error) {
	// Convert time to pgtype.Timestamptz
	var sinceTS pgtype.Timestamptz
	if err := sinceTS.Scan(since); err != nil {
		return 0, err
	}

	// Query using raw SQL since we don't have a specific sqlc query for this
	query := `
		SELECT COUNT(*)
		FROM events
		WHERE tenant_id = $1
		  AND customer_id = $2
		  AND event_type = $3
		  AND occurred_at >= $4
	`

	var count int64
	err := c.pool.QueryRow(ctx, query, tenantID, customerID, eventType, sinceTS).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// countDistinctDays counts distinct days with events of a type since a cutoff date
func (c *CustomOperators) countDistinctDays(
	ctx context.Context,
	tenantID, customerID pgtype.UUID,
	eventType string,
	since time.Time,
) (int64, error) {
	// Convert time to pgtype.Timestamptz
	var sinceTS pgtype.Timestamptz
	if err := sinceTS.Scan(since); err != nil {
		return 0, err
	}

	// Query for distinct dates
	query := `
		SELECT COUNT(DISTINCT DATE(occurred_at))
		FROM events
		WHERE tenant_id = $1
		  AND customer_id = $2
		  AND event_type = $3
		  AND occurred_at >= $4
	`

	var count int64
	err := c.pool.QueryRow(ctx, query, tenantID, customerID, eventType, sinceTS).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}
