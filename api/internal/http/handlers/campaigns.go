package handlers

import (
	"time"

	"github.com/bmachimbira/loyalty/api/internal/campaign"
	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/bmachimbira/loyalty/api/internal/httputil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CampaignsHandler handles campaign-related API endpoints
type CampaignsHandler struct {
	pool    *pgxpool.Pool
	service *campaign.Service
}

// NewCampaignsHandler creates a new campaigns handler
func NewCampaignsHandler(pool *pgxpool.Pool) *CampaignsHandler {
	queries := db.New(pool)
	return &CampaignsHandler{
		pool:    pool,
		service: campaign.NewService(queries),
	}
}

// CreateCampaignRequest represents the request to create a campaign
type CreateCampaignRequest struct {
	Name     string  `json:"name" binding:"required"`
	StartAt  *string `json:"start_at"`
	EndAt    *string `json:"end_at"`
	BudgetID *string `json:"budget_id"`
	Status   string  `json:"status"`
}

// UpdateCampaignRequest represents the request to update a campaign
type UpdateCampaignRequest struct {
	Name     *string `json:"name"`
	StartAt  *string `json:"start_at"`
	EndAt    *string `json:"end_at"`
	BudgetID *string `json:"budget_id"`
	Status   *string `json:"status"`
}

// Create handles POST /v1/tenants/:tid/campaigns
func (h *CampaignsHandler) Create(c *gin.Context) {
	tenantID := c.Param("tid")
	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	var req CreateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	// Validate budget ID if provided
	if req.BudgetID != nil {
		if err := httputil.ValidateUUID(*req.BudgetID); err != nil {
			httputil.BadRequest(c, "Invalid budget ID", nil)
			return
		}
	}

	// Validate status
	if req.Status == "" {
		req.Status = "active"
	}
	validStatuses := map[string]bool{
		"active":    true,
		"paused":    true,
		"completed": true,
	}
	if !validStatuses[req.Status] {
		httputil.BadRequest(c, "Invalid status. Must be active, paused, or completed", nil)
		return
	}

	// Parse tenant UUID
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}

	// Prepare parameters
	var startAt, endAt pgtype.Timestamptz
	if req.StartAt != nil {
		startTime, err := time.Parse(time.RFC3339, *req.StartAt)
		if err != nil {
			httputil.BadRequest(c, "Invalid start_at format", nil)
			return
		}
		startAt = pgtype.Timestamptz{Time: startTime, Valid: true}
	}
	if req.EndAt != nil {
		endTime, err := time.Parse(time.RFC3339, *req.EndAt)
		if err != nil {
			httputil.BadRequest(c, "Invalid end_at format", nil)
			return
		}
		endAt = pgtype.Timestamptz{Time: endTime, Valid: true}
	}

	var budgetID pgtype.UUID
	if req.BudgetID != nil {
		if err := budgetID.Scan(*req.BudgetID); err != nil {
			httputil.BadRequest(c, "Invalid budget ID format", nil)
			return
		}
	}

	// Create campaign using service
	campaign, err := h.service.CreateCampaign(c.Request.Context(), db.CreateCampaignParams{
		TenantID: tenantUUID,
		Name:     req.Name,
		StartAt:  startAt,
		EndAt:    endAt,
		BudgetID: budgetID,
		Status:   req.Status,
	})
	if err != nil {
		httputil.InternalError(c, "Failed to create campaign")
		return
	}

	c.JSON(201, gin.H{
		"id":         formatUUID(campaign.ID),
		"tenant_id":  formatUUID(campaign.TenantID),
		"name":       campaign.Name,
		"start_at":   formatTimestamp(campaign.StartAt),
		"end_at":     formatTimestamp(campaign.EndAt),
		"budget_id":  formatUUID(campaign.BudgetID),
		"status":     campaign.Status,
	})
}

// List handles GET /v1/tenants/:tid/campaigns
func (h *CampaignsHandler) List(c *gin.Context) {
	tenantID := c.Param("tid")
	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	// Get filter params
	status := c.Query("status")

	if status != "" {
		validStatuses := map[string]bool{
			"active":    true,
			"paused":    true,
			"completed": true,
		}
		if !validStatuses[status] {
			httputil.BadRequest(c, "Invalid status", nil)
			return
		}
	}

	// Parse tenant UUID
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}

	// List campaigns using service
	campaigns, total, err := h.service.ListCampaigns(c.Request.Context(), tenantUUID, "50", "0", status)
	if err != nil {
		httputil.InternalError(c, "Failed to list campaigns")
		return
	}

	// Format response
	campaignsList := make([]gin.H, len(campaigns))
	for i, campaign := range campaigns {
		campaignsList[i] = gin.H{
			"id":         formatUUID(campaign.ID),
			"tenant_id":  formatUUID(campaign.TenantID),
			"name":       campaign.Name,
			"start_at":   formatTimestamp(campaign.StartAt),
			"end_at":     formatTimestamp(campaign.EndAt),
			"budget_id":  formatUUID(campaign.BudgetID),
			"status":     campaign.Status,
		}
	}

	c.JSON(200, gin.H{
		"campaigns": campaignsList,
		"total":     total,
		"filters": gin.H{
			"status": status,
		},
	})
}

// Get handles GET /v1/tenants/:tid/campaigns/:id
func (h *CampaignsHandler) Get(c *gin.Context) {
	tenantID := c.Param("tid")
	campaignID := c.Param("id")

	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	if err := httputil.ValidateUUID(campaignID); err != nil {
		httputil.BadRequest(c, "Invalid campaign ID", nil)
		return
	}

	// Parse UUIDs
	var tenantUUID, campaignUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}
	if err := campaignUUID.Scan(campaignID); err != nil {
		httputil.BadRequest(c, "Invalid campaign ID format", nil)
		return
	}

	// Get campaign using service
	campaign, err := h.service.GetCampaignByID(c.Request.Context(), campaignUUID, tenantUUID)
	if err != nil {
		httputil.NotFound(c, "Campaign not found")
		return
	}

	c.JSON(200, gin.H{
		"id":         formatUUID(campaign.ID),
		"tenant_id":  formatUUID(campaign.TenantID),
		"name":       campaign.Name,
		"start_at":   formatTimestamp(campaign.StartAt),
		"end_at":     formatTimestamp(campaign.EndAt),
		"budget_id":  formatUUID(campaign.BudgetID),
		"status":     campaign.Status,
	})
}

// Update handles PATCH /v1/tenants/:tid/campaigns/:id
func (h *CampaignsHandler) Update(c *gin.Context) {
	tenantID := c.Param("tid")
	campaignID := c.Param("id")

	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	if err := httputil.ValidateUUID(campaignID); err != nil {
		httputil.BadRequest(c, "Invalid campaign ID", nil)
		return
	}

	var req UpdateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	// Validate budget ID if provided
	if req.BudgetID != nil {
		if err := httputil.ValidateUUID(*req.BudgetID); err != nil {
			httputil.BadRequest(c, "Invalid budget ID", nil)
			return
		}
	}

	// Validate status if provided
	if req.Status != nil {
		validStatuses := map[string]bool{
			"active":    true,
			"paused":    true,
			"completed": true,
		}
		if !validStatuses[*req.Status] {
			httputil.BadRequest(c, "Invalid status", nil)
			return
		}
	}

	// Parse UUIDs
	var tenantUUID, campaignUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}
	if err := campaignUUID.Scan(campaignID); err != nil {
		httputil.BadRequest(c, "Invalid campaign ID format", nil)
		return
	}

	// Get current campaign to merge updates
	currentCampaign, err := h.service.GetCampaignByID(c.Request.Context(), campaignUUID, tenantUUID)
	if err != nil {
		httputil.NotFound(c, "Campaign not found")
		return
	}

	// Prepare update parameters
	name := currentCampaign.Name
	if req.Name != nil {
		name = *req.Name
	}

	startAt := currentCampaign.StartAt
	if req.StartAt != nil {
		startTime, err := time.Parse(time.RFC3339, *req.StartAt)
		if err != nil {
			httputil.BadRequest(c, "Invalid start_at format", nil)
			return
		}
		startAt = pgtype.Timestamptz{Time: startTime, Valid: true}
	}

	endAt := currentCampaign.EndAt
	if req.EndAt != nil {
		endTime, err := time.Parse(time.RFC3339, *req.EndAt)
		if err != nil {
			httputil.BadRequest(c, "Invalid end_at format", nil)
			return
		}
		endAt = pgtype.Timestamptz{Time: endTime, Valid: true}
	}

	budgetID := currentCampaign.BudgetID
	if req.BudgetID != nil {
		if err := budgetID.Scan(*req.BudgetID); err != nil {
			httputil.BadRequest(c, "Invalid budget ID format", nil)
			return
		}
	}

	status := currentCampaign.Status
	if req.Status != nil {
		status = *req.Status
	}

	// Update campaign using service
	err = h.service.UpdateCampaign(c.Request.Context(), db.UpdateCampaignParams{
		ID:       campaignUUID,
		TenantID: tenantUUID,
		Name:     name,
		StartAt:  startAt,
		EndAt:    endAt,
		BudgetID: budgetID,
		Status:   status,
	})
	if err != nil {
		httputil.InternalError(c, "Failed to update campaign")
		return
	}

	// Get updated campaign
	campaign, err := h.service.GetCampaignByID(c.Request.Context(), campaignUUID, tenantUUID)
	if err != nil {
		httputil.InternalError(c, "Failed to get updated campaign")
		return
	}

	c.JSON(200, gin.H{
		"id":         formatUUID(campaign.ID),
		"tenant_id":  formatUUID(campaign.TenantID),
		"name":       campaign.Name,
		"start_at":   formatTimestamp(campaign.StartAt),
		"end_at":     formatTimestamp(campaign.EndAt),
		"budget_id":  formatUUID(campaign.BudgetID),
		"status":     campaign.Status,
	})
}
