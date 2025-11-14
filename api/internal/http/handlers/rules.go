package handlers

import (
	httputil "github.com/bmachimbira/loyalty/api/internal/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RulesHandler handles rule-related API endpoints
type RulesHandler struct {
	pool *pgxpool.Pool
}

// NewRulesHandler creates a new rules handler
func NewRulesHandler(pool *pgxpool.Pool) *RulesHandler {
	return &RulesHandler{pool: pool}
}

// CreateRuleRequest represents the request to create a rule
type CreateRuleRequest struct {
	Name         string                 `json:"name" binding:"required"`
	Description  string                 `json:"description"`
	Conditions   map[string]interface{} `json:"conditions" binding:"required"`
	RewardID     string                 `json:"reward_id" binding:"required"`
	Priority     int                    `json:"priority"`
	CapPerUser   *int                   `json:"cap_per_user"`
	CapGlobal    *int                   `json:"cap_global"`
	CooldownMins *int                   `json:"cooldown_mins"`
	Active       bool                   `json:"active"`
}

// UpdateRuleRequest represents the request to update a rule
type UpdateRuleRequest struct {
	Name         *string                 `json:"name"`
	Description  *string                 `json:"description"`
	Conditions   *map[string]interface{} `json:"conditions"`
	Priority     *int                    `json:"priority"`
	CapPerUser   *int                    `json:"cap_per_user"`
	CapGlobal    *int                    `json:"cap_global"`
	CooldownMins *int                    `json:"cooldown_mins"`
	Active       *bool                   `json:"active"`
}

// Create handles POST /v1/tenants/:tid/rules
func (h *RulesHandler) Create(c *gin.Context) {
	tenantID := c.Param("tid")
	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	var req CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	// Validate reward ID
	if err := httputil.ValidateUUID(req.RewardID); err != nil {
		httputil.BadRequest(c, "Invalid reward ID", nil)
		return
	}

	// TODO: Validate JsonLogic conditions format
	// TODO: Once sqlc is generated, use queries.CreateRule()

	c.JSON(201, gin.H{
		"id":            uuid.New().String(),
		"tenant_id":     tenantID,
		"name":          req.Name,
		"description":   req.Description,
		"conditions":    req.Conditions,
		"reward_id":     req.RewardID,
		"priority":      req.Priority,
		"cap_per_user":  req.CapPerUser,
		"cap_global":    req.CapGlobal,
		"cooldown_mins": req.CooldownMins,
		"active":        req.Active,
		"created_at":    "2025-11-14T00:00:00Z",
	})
}

// List handles GET /v1/tenants/:tid/rules
func (h *RulesHandler) List(c *gin.Context) {
	tenantID := c.Param("tid")
	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	// Get filter params
	activeOnly := c.DefaultQuery("active_only", "false")

	// TODO: Once sqlc is generated, use queries.ListRules()
	c.JSON(200, gin.H{
		"rules": []gin.H{},
		"total": 0,
		"filters": gin.H{
			"active_only": activeOnly,
		},
	})
}

// Get handles GET /v1/tenants/:tid/rules/:id
func (h *RulesHandler) Get(c *gin.Context) {
	tenantID := c.Param("tid")
	ruleID := c.Param("id")

	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	if err := httputil.ValidateUUID(ruleID); err != nil {
		httputil.BadRequest(c, "Invalid rule ID", nil)
		return
	}

	// TODO: Once sqlc is generated, use queries.GetRule()
	c.JSON(200, gin.H{
		"id":            ruleID,
		"tenant_id":     tenantID,
		"name":          "Sample Rule",
		"description":   "A sample rule",
		"conditions":    gin.H{},
		"reward_id":     uuid.New().String(),
		"priority":      0,
		"cap_per_user":  nil,
		"cap_global":    nil,
		"cooldown_mins": nil,
		"active":        true,
		"created_at":    "2025-11-14T00:00:00Z",
	})
}

// Update handles PATCH /v1/tenants/:tid/rules/:id
func (h *RulesHandler) Update(c *gin.Context) {
	tenantID := c.Param("tid")
	ruleID := c.Param("id")

	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	if err := httputil.ValidateUUID(ruleID); err != nil {
		httputil.BadRequest(c, "Invalid rule ID", nil)
		return
	}

	var req UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	// TODO: Once sqlc is generated, use queries.UpdateRule()
	c.JSON(200, gin.H{
		"id":         ruleID,
		"tenant_id":  tenantID,
		"updated_at": "2025-11-14T00:00:00Z",
	})
}

// Delete handles DELETE /v1/tenants/:tid/rules/:id
// This is a soft delete (sets active=false)
func (h *RulesHandler) Delete(c *gin.Context) {
	tenantID := c.Param("tid")
	ruleID := c.Param("id")

	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	if err := httputil.ValidateUUID(ruleID); err != nil {
		httputil.BadRequest(c, "Invalid rule ID", nil)
		return
	}

	// TODO: Once sqlc is generated, use queries.DeactivateRule()
	c.JSON(200, gin.H{
		"id":      ruleID,
		"message": "Rule deactivated successfully",
	})
}
