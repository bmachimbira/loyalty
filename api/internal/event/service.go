package event

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Service handles event-related business logic
type Service struct {
	queries *db.Queries
}

// NewService creates a new event service
func NewService(queries *db.Queries) *Service {
	return &Service{
		queries: queries,
	}
}

// ListEventsByCustomer retrieves a paginated list of events for a customer
func (s *Service) ListEventsByCustomer(ctx context.Context, tenantID, customerID pgtype.UUID, limit, offset string) ([]db.Event, int, error) {
	// Parse limit and offset
	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt < 1 {
		limitInt = 50
	}
	if limitInt > 100 {
		limitInt = 100
	}

	offsetInt, err := strconv.Atoi(offset)
	if err != nil || offsetInt < 0 {
		offsetInt = 0
	}

	events, err := s.queries.ListEventsByCustomer(ctx, db.ListEventsByCustomerParams{
		TenantID:   tenantID,
		CustomerID: customerID,
		Limit:      int32(limitInt),
		Offset:     int32(offsetInt),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list events: %w", err)
	}

	return events, len(events), nil
}

// GetEventByIdemKey retrieves an event by its idempotency key
func (s *Service) GetEventByIdemKey(ctx context.Context, tenantID pgtype.UUID, idempotencyKey string) (db.Event, error) {
	event, err := s.queries.GetEventByIdemKey(ctx, db.GetEventByIdemKeyParams{
		TenantID:       tenantID,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Event{}, fmt.Errorf("event not found")
		}
		return db.Event{}, fmt.Errorf("failed to get event: %w", err)
	}
	return event, nil
}

// InsertEvent creates a new event
func (s *Service) InsertEvent(ctx context.Context, params db.InsertEventParams) (db.Event, error) {
	return s.queries.InsertEvent(ctx, params)
}

// GetActiveRulesForEvent retrieves active rules for a given event type
func (s *Service) GetActiveRulesForEvent(ctx context.Context, tenantID pgtype.UUID, eventType string) ([]db.Rule, error) {
	return s.queries.GetActiveRulesForEvent(ctx, db.GetActiveRulesForEventParams{
		TenantID:  tenantID,
		EventType: eventType,
	})
}
