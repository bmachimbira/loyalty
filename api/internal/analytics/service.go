package analytics

import (
	"context"
	"log/slog"
	"time"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

// Service handles analytics operations
type Service struct {
	queries *db.Queries
	logger  *slog.Logger
}

// NewService creates a new analytics service
func NewService(queries *db.Queries, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		queries: queries,
		logger:  logger,
	}
}

// DashboardStats represents the dashboard statistics
type DashboardStats struct {
	ActiveCustomers      int64
	EventsToday          int64
	RewardsIssuedToday   int64
	RewardsRedeemedToday int64
	RedemptionRate       float64
}

// GetDashboardStats retrieves dashboard statistics for a tenant
func (s *Service) GetDashboardStats(ctx context.Context, tenantID pgtype.UUID) (*DashboardStats, error) {
	// Get start of today in UTC
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	
	var todayTimestamp pgtype.Timestamptz
	todayTimestamp.Time = startOfDay
	todayTimestamp.Valid = true

	stats, err := s.queries.GetDashboardStats(ctx, db.GetDashboardStatsParams{
		TenantID:   tenantID,
		OccurredAt: todayTimestamp,
	})
	if err != nil {
		s.logger.Error("Failed to fetch dashboard stats",
			"tenant_id", tenantID,
			"error", err,
		)
		return nil, err
	}

	// Calculate redemption rate
	var redemptionRate float64
	if stats.RewardsIssuedToday > 0 {
		redemptionRate = (float64(stats.RewardsRedeemedToday) / float64(stats.RewardsIssuedToday)) * 100
	}

	return &DashboardStats{
		ActiveCustomers:      stats.ActiveCustomers,
		EventsToday:          stats.EventsToday,
		RewardsIssuedToday:   stats.RewardsIssuedToday,
		RewardsRedeemedToday: stats.RewardsRedeemedToday,
		RedemptionRate:       redemptionRate,
	}, nil
}
