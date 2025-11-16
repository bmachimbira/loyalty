package customer

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Service handles customer-related business logic
type Service struct {
	queries *db.Queries
}

// NewService creates a new customer service
func NewService(queries *db.Queries) *Service {
	return &Service{
		queries: queries,
	}
}

// CreateCustomer creates a new customer
func (s *Service) CreateCustomer(ctx context.Context, params db.CreateCustomerParams) (db.Customer, error) {
	return s.queries.CreateCustomer(ctx, params)
}

// GetCustomerByID retrieves a customer by ID
func (s *Service) GetCustomerByID(ctx context.Context, id, tenantID pgtype.UUID) (db.Customer, error) {
	return s.queries.GetCustomerByID(ctx, db.GetCustomerByIDParams{
		ID:       id,
		TenantID: tenantID,
	})
}

// GetCustomerByPhone retrieves a customer by phone number
func (s *Service) GetCustomerByPhone(ctx context.Context, tenantID pgtype.UUID, phoneE164 string) (db.Customer, error) {
	return s.queries.GetCustomerByPhone(ctx, db.GetCustomerByPhoneParams{
		TenantID:  tenantID,
		PhoneE164: pgtype.Text{String: phoneE164, Valid: true},
	})
}

// GetCustomerByExternalRef retrieves a customer by external reference
func (s *Service) GetCustomerByExternalRef(ctx context.Context, tenantID pgtype.UUID, externalRef string) (db.Customer, error) {
	return s.queries.GetCustomerByExternalRef(ctx, db.GetCustomerByExternalRefParams{
		TenantID:    tenantID,
		ExternalRef: pgtype.Text{String: externalRef, Valid: true},
	})
}

// ListCustomers retrieves a paginated list of customers
func (s *Service) ListCustomers(ctx context.Context, tenantID pgtype.UUID, limit, offset string) ([]db.Customer, int, error) {
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

	customers, err := s.queries.ListCustomers(ctx, db.ListCustomersParams{
		TenantID: tenantID,
		Limit:    int32(limitInt),
		Offset:   int32(offsetInt),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list customers: %w", err)
	}

	return customers, len(customers), nil
}

// UpdateCustomerStatus updates the status of a customer
func (s *Service) UpdateCustomerStatus(ctx context.Context, id, tenantID pgtype.UUID, status string) error {
	err := s.queries.UpdateCustomerStatus(ctx, db.UpdateCustomerStatusParams{
		ID:       id,
		TenantID: tenantID,
		Status:   status,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("customer not found")
		}
		return fmt.Errorf("failed to update customer status: %w", err)
	}
	return nil
}
