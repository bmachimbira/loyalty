package handlers

import (
	"encoding/json"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/bmachimbira/loyalty/api/internal/httputil"
	"github.com/bmachimbira/loyalty/api/internal/rule"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RulesHandler handles rule-related API endpoints
type RulesHandler struct {
	pool    *pgxpool.Pool
	service *rule.Service
}

// NewRulesHandler creates a new rules handler
func NewRulesHandler(pool *pgxpool.Pool) *RulesHandler {
	queries := db.New(pool)
	return &RulesHandler{
		pool:    pool,
		service: rule.NewService(queries),
	}
}

// CreateRuleRequest represents the request to create a rule
type CreateRuleRequest struct {
	Name         string                 `json:"name" binding:"required"`
	Description  string                 `json:"description"`
	EventType    string                 `json:"event_type" binding:"required"`
	Conditions   map[string]interface{} `json:"conditions" binding:"required"`
	RewardID     string                 `json:"reward_id" binding:"required"`
	CampaignID   *string                `json:"campaign_id"`
	Priority     int                    `json:"priority"`
	CapPerUser   int                    `json:"cap_per_user"`
	CapGlobal    *int                   `json:"cap_global"`
	CooldownSecs int                    `json:"cooldown_secs"`
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

	// Validate campaign ID if provided
	if req.CampaignID != nil {
		if err := httputil.ValidateUUID(*req.CampaignID); err != nil {
			httputil.BadRequest(c, "Invalid campaign ID", nil)
			return
		}
	}

	// Parse UUIDs
	var tenantUUID, rewardUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}
	if err := rewardUUID.Scan(req.RewardID); err != nil {
		httputil.BadRequest(c, "Invalid reward ID format", nil)
		return
	}

	var campaignUUID pgtype.UUID
	if req.CampaignID != nil {
		if err := campaignUUID.Scan(*req.CampaignID); err != nil {
			httputil.BadRequest(c, "Invalid campaign ID format", nil)
			return
		}
	}

	// Serialize conditions to JSON
	conditionsJSON, err := json.Marshal(req.Conditions)
	if err != nil {
		httputil.BadRequest(c, "Invalid conditions format", nil)
		return
	}

	// Prepare global cap
	var globalCap pgtype.Int4
	if req.CapGlobal != nil {
		globalCap = pgtype.Int4{Int32: int32(*req.CapGlobal), Valid: true}
	}

	// Create rule using service
	createdRule, err := h.service.CreateRule(c.Request.Context(), db.CreateRuleParams{
		TenantID:    tenantUUID,
		CampaignID:  campaignUUID,
		Name:        req.Name,
		EventType:   req.EventType,
		Conditions:  conditionsJSON,
		RewardID:    rewardUUID,
		PerUserCap:  int32(req.CapPerUser),
		GlobalCap:   globalCap,
		CoolDownSec: int32(req.CooldownSecs),
		Active:      req.Active,
	})
	if err != nil {
		httputil.InternalError(c, "Failed to create rule")
		return
	}

	// Parse conditions back for response
	var conditions map[string]interface{}
	json.Unmarshal(createdRule.Conditions, &conditions)

	c.JSON(201, gin.H{
		"id":           formatUUID(createdRule.ID),
		"tenant_id":    formatUUID(createdRule.TenantID),
		"campaign_id":  formatUUID(createdRule.CampaignID),
		"name":         createdRule.Name,
		"event_type":   createdRule.EventType,
		"conditions":   conditions,
		"reward_id":    formatUUID(createdRule.RewardID),
		"per_user_cap": createdRule.PerUserCap,
		"global_cap":   createdRule.GlobalCap.Int32,
		"cool_down_sec": createdRule.CoolDownSec,
		"active":       createdRule.Active,
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

	// Parse tenant UUID
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}

	// List rules using service
	rules, err := h.service.ListRules(c.Request.Context(), tenantUUID, activeOnly == "true")
	if err != nil {
		httputil.InternalError(c, "Failed to list rules")
		return
	}

	// Format response
	rulesList := make([]gin.H, len(rules))
	for i, rule := range rules {
		var conditions map[string]interface{}
		if len(rule.Conditions) > 0 {
			json.Unmarshal(rule.Conditions, &conditions)
		}

		rulesList[i] = gin.H{
			"id":            formatUUID(rule.ID),
			"tenant_id":     formatUUID(rule.TenantID),
			"campaign_id":   formatUUID(rule.CampaignID),
			"name":          rule.Name,
			"event_type":    rule.EventType,
			"conditions":    conditions,
			"reward_id":     formatUUID(rule.RewardID),
			"per_user_cap":  rule.PerUserCap,
			"global_cap":    rule.GlobalCap.Int32,
			"cool_down_sec": rule.CoolDownSec,
			"active":        rule.Active,
		}
	}

	c.JSON(200, gin.H{
		"rules": rulesList,
		"total": len(rules),
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

	// Parse UUIDs
	var tenantUUID, ruleUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}
	if err := ruleUUID.Scan(ruleID); err != nil {
		httputil.BadRequest(c, "Invalid rule ID format", nil)
		return
	}

	// Get rule using service
	rule, err := h.service.GetRuleByID(c.Request.Context(), ruleUUID, tenantUUID)
	if err != nil {
		httputil.NotFound(c, "Rule not found")
		return
	}

	// Parse conditions
	var conditions map[string]interface{}
	if len(rule.Conditions) > 0 {
		json.Unmarshal(rule.Conditions, &conditions)
	}

	c.JSON(200, gin.H{
		"id":            formatUUID(rule.ID),
		"tenant_id":     formatUUID(rule.TenantID),
		"campaign_id":   formatUUID(rule.CampaignID),
		"name":          rule.Name,
		"event_type":    rule.EventType,
		"conditions":    conditions,
		"reward_id":     formatUUID(rule.RewardID),
		"per_user_cap":  rule.PerUserCap,
		"global_cap":    rule.GlobalCap.Int32,
		"cool_down_sec": rule.CoolDownSec,
		"active":        rule.Active,
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

	// Parse UUIDs
	var tenantUUID, ruleUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}
	if err := ruleUUID.Scan(ruleID); err != nil {
		httputil.BadRequest(c, "Invalid rule ID format", nil)
		return
	}

	// For now, we only support updating the active status
	// Full update would require a new UpdateRule query
	if req.Active != nil {
		err := h.service.UpdateRuleStatus(c.Request.Context(), ruleUUID, tenantUUID, *req.Active)
		if err != nil {
			httputil.InternalError(c, "Failed to update rule")
			return
		}
	}

	// Get updated rule
	rule, err := h.service.GetRuleByID(c.Request.Context(), ruleUUID, tenantUUID)
	if err != nil {
		httputil.InternalError(c, "Failed to get updated rule")
		return
	}

	// Parse conditions
	var conditions map[string]interface{}
	if len(rule.Conditions) > 0 {
		json.Unmarshal(rule.Conditions, &conditions)
	}

	c.JSON(200, gin.H{
		"id":            formatUUID(rule.ID),
		"tenant_id":     formatUUID(rule.TenantID),
		"campaign_id":   formatUUID(rule.CampaignID),
		"name":          rule.Name,
		"event_type":    rule.EventType,
		"conditions":    conditions,
		"reward_id":     formatUUID(rule.RewardID),
		"per_user_cap":  rule.PerUserCap,
		"global_cap":    rule.GlobalCap.Int32,
		"cool_down_sec": rule.CoolDownSec,
		"active":        rule.Active,
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

	// Parse UUIDs
	var tenantUUID, ruleUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}
	if err := ruleUUID.Scan(ruleID); err != nil {
		httputil.BadRequest(c, "Invalid rule ID format", nil)
		return
	}

	// Deactivate rule using service
	err := h.service.DeactivateRule(c.Request.Context(), ruleUUID, tenantUUID)
	if err != nil {
		httputil.InternalError(c, "Failed to deactivate rule")
		return
	}

	c.JSON(200, gin.H{
		"id":      formatUUID(ruleUUID),
		"message": "Rule deactivated successfully",
	})
}
