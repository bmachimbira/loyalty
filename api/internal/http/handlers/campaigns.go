package handlers

import (
	"github.com/bmachimbira/loyalty/api/internal/httputil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CampaignsHandler handles campaign-related API endpoints
type CampaignsHandler struct {
	pool *pgxpool.Pool
}

// NewCampaignsHandler creates a new campaigns handler
func NewCampaignsHandler(pool *pgxpool.Pool) *CampaignsHandler {
	return &CampaignsHandler{pool: pool}
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

	// TODO: Once sqlc is generated, use queries.CreateCampaign()
	c.JSON(201, gin.H{
		"id":         uuid.New().String(),
		"tenant_id":  tenantID,
		"name":       req.Name,
		"start_at":   req.StartAt,
		"end_at":     req.EndAt,
		"budget_id":  req.BudgetID,
		"status":     req.Status,
		"created_at": "2025-11-14T00:00:00Z",
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

	// TODO: Once sqlc is generated, use queries.ListCampaigns()
	c.JSON(200, gin.H{
		"campaigns": []gin.H{},
		"total":     0,
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

	// TODO: Once sqlc is generated, use queries.GetCampaign()
	c.JSON(200, gin.H{
		"id":         campaignID,
		"tenant_id":  tenantID,
		"name":       "Sample Campaign",
		"start_at":   nil,
		"end_at":     nil,
		"budget_id":  nil,
		"status":     "active",
		"created_at": "2025-11-14T00:00:00Z",
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

	// TODO: Once sqlc is generated, use queries.UpdateCampaign()
	c.JSON(200, gin.H{
		"id":         campaignID,
		"tenant_id":  tenantID,
		"updated_at": "2025-11-14T00:00:00Z",
	})
}
