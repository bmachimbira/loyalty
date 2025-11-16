package campaign

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Service handles campaign-related business logic
type Service struct {
	queries *db.Queries
}

// NewService creates a new campaign service
func NewService(queries *db.Queries) *Service {
	return &Service{
		queries: queries,
	}
}

// CreateCampaign creates a new campaign
func (s *Service) CreateCampaign(ctx context.Context, params db.CreateCampaignParams) (db.Campaign, error) {
	return s.queries.CreateCampaign(ctx, params)
}

// GetCampaignByID retrieves a campaign by ID
func (s *Service) GetCampaignByID(ctx context.Context, id, tenantID pgtype.UUID) (db.Campaign, error) {
	return s.queries.GetCampaignByID(ctx, db.GetCampaignByIDParams{
		ID:       id,
		TenantID: tenantID,
	})
}

// ListCampaigns retrieves a paginated list of campaigns
func (s *Service) ListCampaigns(ctx context.Context, tenantID pgtype.UUID, limit, offset string, status string) ([]db.Campaign, int, error) {
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

	var campaigns []db.Campaign

	// If status filter is provided, use GetCampaignsByStatus
	if status != "" {
		campaigns, err = s.queries.GetCampaignsByStatus(ctx, db.GetCampaignsByStatusParams{
			TenantID: tenantID,
			Status:   status,
			Limit:    int32(limitInt),
			Offset:   int32(offsetInt),
		})
	} else {
		// Otherwise use ListCampaigns
		campaigns, err = s.queries.ListCampaigns(ctx, db.ListCampaignsParams{
			TenantID: tenantID,
			Limit:    int32(limitInt),
			Offset:   int32(offsetInt),
		})
	}

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list campaigns: %w", err)
	}

	return campaigns, len(campaigns), nil
}

// UpdateCampaign updates a campaign
func (s *Service) UpdateCampaign(ctx context.Context, params db.UpdateCampaignParams) error {
	err := s.queries.UpdateCampaign(ctx, params)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("campaign not found")
		}
		return fmt.Errorf("failed to update campaign: %w", err)
	}
	return nil
}

// UpdateCampaignStatus updates the status of a campaign
func (s *Service) UpdateCampaignStatus(ctx context.Context, id, tenantID pgtype.UUID, status string) error {
	err := s.queries.UpdateCampaignStatus(ctx, db.UpdateCampaignStatusParams{
		ID:       id,
		TenantID: tenantID,
		Status:   status,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("campaign not found")
		}
		return fmt.Errorf("failed to update campaign status: %w", err)
	}
	return nil
}

// ListActiveCampaigns retrieves all currently active campaigns
func (s *Service) ListActiveCampaigns(ctx context.Context, tenantID pgtype.UUID) ([]db.Campaign, error) {
	return s.queries.ListActiveCampaigns(ctx, tenantID)
}
