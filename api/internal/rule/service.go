package rule

import (
	"context"
	"fmt"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Service handles rule-related business logic
type Service struct {
	queries *db.Queries
}

// NewService creates a new rule service
func NewService(queries *db.Queries) *Service {
	return &Service{
		queries: queries,
	}
}

// CreateRule creates a new rule
func (s *Service) CreateRule(ctx context.Context, params db.CreateRuleParams) (db.Rule, error) {
	return s.queries.CreateRule(ctx, params)
}

// GetRuleByID retrieves a rule by ID
func (s *Service) GetRuleByID(ctx context.Context, id, tenantID pgtype.UUID) (db.Rule, error) {
	rule, err := s.queries.GetRuleByID(ctx, db.GetRuleByIDParams{
		ID:       id,
		TenantID: tenantID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Rule{}, fmt.Errorf("rule not found")
		}
		return db.Rule{}, fmt.Errorf("failed to get rule: %w", err)
	}
	return rule, nil
}

// ListRules retrieves all rules for a tenant
func (s *Service) ListRules(ctx context.Context, tenantID pgtype.UUID, activeOnly bool) ([]db.Rule, error) {
	if activeOnly {
		return s.queries.ListActiveRules(ctx, tenantID)
	}
	// If we need all rules, we'd need a ListAllRules query
	// For now, just return active rules
	return s.queries.ListActiveRules(ctx, tenantID)
}

// UpdateRuleStatus updates the active status of a rule
func (s *Service) UpdateRuleStatus(ctx context.Context, id, tenantID pgtype.UUID, active bool) error {
	err := s.queries.UpdateRuleStatus(ctx, db.UpdateRuleStatusParams{
		ID:       id,
		TenantID: tenantID,
		Active:   active,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("rule not found")
		}
		return fmt.Errorf("failed to update rule status: %w", err)
	}
	return nil
}

// DeactivateRule deactivates a rule (soft delete)
func (s *Service) DeactivateRule(ctx context.Context, id, tenantID pgtype.UUID) error {
	return s.UpdateRuleStatus(ctx, id, tenantID, false)
}
