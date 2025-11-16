package rewardcatalog

import (
	"context"
	"fmt"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Service handles reward catalog-related business logic
type Service struct {
	queries *db.Queries
}

// NewService creates a new reward catalog service
func NewService(queries *db.Queries) *Service {
	return &Service{
		queries: queries,
	}
}

// CreateReward creates a new reward in the catalog
func (s *Service) CreateReward(ctx context.Context, params db.CreateRewardParams) (db.RewardCatalog, error) {
	return s.queries.CreateReward(ctx, params)
}

// GetRewardByID retrieves a reward by ID
func (s *Service) GetRewardByID(ctx context.Context, id, tenantID pgtype.UUID) (db.RewardCatalog, error) {
	reward, err := s.queries.GetRewardByID(ctx, db.GetRewardByIDParams{
		ID:       id,
		TenantID: tenantID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.RewardCatalog{}, fmt.Errorf("reward not found")
		}
		return db.RewardCatalog{}, fmt.Errorf("failed to get reward: %w", err)
	}
	return reward, nil
}

// ListRewards retrieves all rewards for a tenant
func (s *Service) ListRewards(ctx context.Context, tenantID pgtype.UUID, activeOnly bool) ([]db.RewardCatalog, error) {
	if activeOnly {
		return s.queries.ListActiveRewards(ctx, tenantID)
	}
	return s.queries.ListRewards(ctx, tenantID)
}

// UpdateRewardStatus updates the active status of a reward
func (s *Service) UpdateRewardStatus(ctx context.Context, id, tenantID pgtype.UUID, active bool) error {
	err := s.queries.UpdateRewardStatus(ctx, db.UpdateRewardStatusParams{
		ID:       id,
		TenantID: tenantID,
		Active:   active,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("reward not found")
		}
		return fmt.Errorf("failed to update reward status: %w", err)
	}
	return nil
}
