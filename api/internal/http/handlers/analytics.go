package handlers

import (
	"github.com/bmachimbira/loyalty/api/internal/analytics"
	"github.com/bmachimbira/loyalty/api/internal/httputil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

// AnalyticsHandler handles analytics-related API endpoints
type AnalyticsHandler struct {
	service *analytics.Service
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(service *analytics.Service) *AnalyticsHandler {
	return &AnalyticsHandler{service: service}
}

// DashboardStatsResponse represents the dashboard statistics response
type DashboardStatsResponse struct {
	ActiveCustomers    int64   `json:"active_customers"`
	EventsToday        int64   `json:"events_today"`
	RewardsIssuedToday int64   `json:"rewards_issued_today"`
	RedemptionRate     float64 `json:"redemption_rate"`
}

// GetDashboardStats handles GET /v1/tenants/:tid/analytics/dashboard
func (h *AnalyticsHandler) GetDashboardStats(c *gin.Context) {
	tenantID := c.Param("tid")
	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	// Parse tenant UUID
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}

	stats, err := h.service.GetDashboardStats(c.Request.Context(), tenantUUID)
	if err != nil {
		httputil.InternalError(c, "Failed to fetch dashboard stats")
		return
	}

	response := DashboardStatsResponse{
		ActiveCustomers:    stats.ActiveCustomers,
		EventsToday:        stats.EventsToday,
		RewardsIssuedToday: stats.RewardsIssuedToday,
		RedemptionRate:     stats.RedemptionRate,
	}

	c.JSON(200, response)
}
