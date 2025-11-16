package issuance

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Service handles issuance-related business logic
type Service struct {
	queries *db.Queries
}

// NewService creates a new issuance service
func NewService(queries *db.Queries) *Service {
	return &Service{
		queries: queries,
	}
}

// GetIssuanceByID retrieves an issuance by ID
func (s *Service) GetIssuanceByID(ctx context.Context, id, tenantID pgtype.UUID) (db.Issuance, error) {
	issuance, err := s.queries.GetIssuanceByID(ctx, db.GetIssuanceByIDParams{
		ID:       id,
		TenantID: tenantID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Issuance{}, fmt.Errorf("issuance not found")
		}
		return db.Issuance{}, fmt.Errorf("failed to get issuance: %w", err)
	}
	return issuance, nil
}

// ListIssuancesByCustomer retrieves a paginated list of issuances for a customer
func (s *Service) ListIssuancesByCustomer(ctx context.Context, tenantID, customerID pgtype.UUID, limit, offset string) ([]db.Issuance, int, error) {
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

	issuances, err := s.queries.ListIssuancesByCustomer(ctx, db.ListIssuancesByCustomerParams{
		TenantID:   tenantID,
		CustomerID: customerID,
		Limit:      int32(limitInt),
		Offset:     int32(offsetInt),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list issuances: %w", err)
	}

	return issuances, len(issuances), nil
}

// ListActiveIssuances retrieves all active issuances for a customer
func (s *Service) ListActiveIssuances(ctx context.Context, tenantID, customerID pgtype.UUID) ([]db.Issuance, error) {
	return s.queries.ListActiveIssuances(ctx, db.ListActiveIssuancesParams{
		TenantID:   tenantID,
		CustomerID: customerID,
	})
}

// ReserveIssuance reserves an issuance
func (s *Service) ReserveIssuance(ctx context.Context, params db.ReserveIssuanceParams) (db.Issuance, error) {
	return s.queries.ReserveIssuance(ctx, params)
}

// UpdateIssuanceStatus updates the status of an issuance
func (s *Service) UpdateIssuanceStatus(ctx context.Context, id, tenantID pgtype.UUID, fromStatus, toStatus string) error {
	err := s.queries.UpdateIssuanceStatus(ctx, db.UpdateIssuanceStatusParams{
		ID:       id,
		TenantID: tenantID,
		Status:   fromStatus,
		Status_2: toStatus,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("issuance not found or status mismatch")
		}
		return fmt.Errorf("failed to update issuance status: %w", err)
	}
	return nil
}

// UpdateIssuanceDetails updates the details of an issuance
func (s *Service) UpdateIssuanceDetails(ctx context.Context, params db.UpdateIssuanceDetailsParams) error {
	err := s.queries.UpdateIssuanceDetails(ctx, params)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("issuance not found")
		}
		return fmt.Errorf("failed to update issuance details: %w", err)
	}
	return nil
}
