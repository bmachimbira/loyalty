package handlers

import (
	"log/slog"
	"strings"

	"github.com/bmachimbira/loyalty/api/internal/auth"
	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/bmachimbira/loyalty/api/internal/httputil"
	"github.com/bmachimbira/loyalty/api/internal/issuance"
	"github.com/bmachimbira/loyalty/api/internal/reward"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IssuancesHandler handles issuance-related API endpoints
type IssuancesHandler struct {
	pool          *pgxpool.Pool
	queries       *db.Queries
	service       *issuance.Service
	rewardService *reward.Service
}

// NewIssuancesHandler creates a new issuances handler
func NewIssuancesHandler(pool *pgxpool.Pool, logger *slog.Logger) *IssuancesHandler {
	queries := db.New(pool)
	return &IssuancesHandler{
		pool:          pool,
		queries:       queries,
		service:       issuance.NewService(queries),
		rewardService: reward.NewService(pool, queries),
	}
}

// RedeemIssuanceRequest represents the request to redeem an issuance
type RedeemIssuanceRequest struct {
	OTP      string `json:"otp"`
	StaffPIN string `json:"staff_pin"`
}

// List handles GET /v1/tenants/:tid/issuances
func (h *IssuancesHandler) List(c *gin.Context) {
	tenantID := c.Param("tid")
	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	// Get pagination and filter params
	limit := c.DefaultQuery("limit", "50")
	offset := c.DefaultQuery("offset", "0")
	customerID := c.Query("customer_id")
	status := c.Query("status")

	// Customer ID is required for listing issuances
	if customerID == "" {
		httputil.BadRequest(c, "customer_id query parameter is required", nil)
		return
	}

	if err := httputil.ValidateUUID(customerID); err != nil {
		httputil.BadRequest(c, "Invalid customer ID", nil)
		return
	}

	// Validate status if provided
	if status != "" {
		validStatuses := map[string]bool{
			"reserved":  true,
			"issued":    true,
			"redeemed":  true,
			"expired":   true,
			"cancelled": true,
			"failed":    true,
		}
		if !validStatuses[status] {
			httputil.BadRequest(c, "Invalid status", nil)
			return
		}
	}

	// Parse UUIDs
	var tenantUUID, customerUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}
	if err := customerUUID.Scan(customerID); err != nil {
		httputil.BadRequest(c, "Invalid customer ID format", nil)
		return
	}

	// List issuances using service
	issuances, total, err := h.service.ListIssuancesByCustomer(c.Request.Context(), tenantUUID, customerUUID, limit, offset)
	if err != nil {
		httputil.InternalError(c, "Failed to list issuances")
		return
	}

	// Format response
	issuancesList := make([]gin.H, len(issuances))
	for i, issuance := range issuances {
		issuancesList[i] = gin.H{
			"id":           formatUUID(issuance.ID),
			"tenant_id":    formatUUID(issuance.TenantID),
			"customer_id":  formatUUID(issuance.CustomerID),
			"campaign_id":  formatUUID(issuance.CampaignID),
			"reward_id":    formatUUID(issuance.RewardID),
			"status":       issuance.Status,
			"code":         issuance.Code.String,
			"external_ref": issuance.ExternalRef.String,
			"currency":     issuance.Currency.String,
			"cost_amount":  issuance.CostAmount.Int.String(),
			"face_amount":  issuance.FaceAmount.Int.String(),
			"issued_at":    formatTimestamp(issuance.IssuedAt),
			"expires_at":   formatTimestamp(issuance.ExpiresAt),
			"redeemed_at":  formatTimestamp(issuance.RedeemedAt),
		}
	}

	c.JSON(200, gin.H{
		"issuances": issuancesList,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
		"filters": gin.H{
			"customer_id": customerID,
			"status":      status,
		},
	})
}

// Get handles GET /v1/tenants/:tid/issuances/:id
func (h *IssuancesHandler) Get(c *gin.Context) {
	tenantID := c.Param("tid")
	issuanceID := c.Param("id")

	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	if err := httputil.ValidateUUID(issuanceID); err != nil {
		httputil.BadRequest(c, "Invalid issuance ID", nil)
		return
	}

	// Parse UUIDs
	var tenantUUID, issuanceUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}
	if err := issuanceUUID.Scan(issuanceID); err != nil {
		httputil.BadRequest(c, "Invalid issuance ID format", nil)
		return
	}

	// Get issuance using service
	issuance, err := h.service.GetIssuanceByID(c.Request.Context(), issuanceUUID, tenantUUID)
	if err != nil {
		httputil.NotFound(c, "Issuance not found")
		return
	}

	c.JSON(200, gin.H{
		"id":           formatUUID(issuance.ID),
		"tenant_id":    formatUUID(issuance.TenantID),
		"customer_id":  formatUUID(issuance.CustomerID),
		"campaign_id":  formatUUID(issuance.CampaignID),
		"reward_id":    formatUUID(issuance.RewardID),
		"status":       issuance.Status,
		"code":         issuance.Code.String,
		"external_ref": issuance.ExternalRef.String,
		"currency":     issuance.Currency.String,
		"cost_amount":  issuance.CostAmount.Int.String(),
		"face_amount":  issuance.FaceAmount.Int.String(),
		"issued_at":    formatTimestamp(issuance.IssuedAt),
		"expires_at":   formatTimestamp(issuance.ExpiresAt),
		"redeemed_at":  formatTimestamp(issuance.RedeemedAt),
	})
}

// Redeem handles POST /v1/tenants/:tid/issuances/:id/redeem
func (h *IssuancesHandler) Redeem(c *gin.Context) {
	tenantID := c.Param("tid")
	issuanceID := c.Param("id")

	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	if err := httputil.ValidateUUID(issuanceID); err != nil {
		httputil.BadRequest(c, "Invalid issuance ID", nil)
		return
	}

	var req RedeemIssuanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	// Validate that either OTP or staff PIN is provided
	if req.OTP == "" && req.StaffPIN == "" {
		httputil.BadRequest(c, "Either OTP or staff PIN is required", nil)
		return
	}

	// Parse UUIDs
	var tenantUUID, issuanceUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}
	if err := issuanceUUID.Scan(issuanceID); err != nil {
		httputil.BadRequest(c, "Invalid issuance ID format", nil)
		return
	}

	// If staff PIN is provided, validate it against the authenticated user's password
	if req.StaffPIN != "" {
		// Get authenticated user from context (set by auth middleware)
		userID, exists := c.Get("user_id")
		if !exists {
			httputil.Unauthorized(c, "User not authenticated")
			return
		}

		// Parse user UUID
		var userUUID pgtype.UUID
		if err := userUUID.Scan(userID.(string)); err != nil {
			httputil.InternalError(c, "Invalid user ID")
			return
		}

		// Get staff user from database
		staffUser, err := h.queries.GetStaffUserByID(c.Request.Context(), db.GetStaffUserByIDParams{
			ID:       userUUID,
			TenantID: tenantUUID,
		})
		if err != nil {
			httputil.Unauthorized(c, "Staff user not found")
			return
		}

		// Validate the staff PIN against the password hash
		if err := auth.ComparePassword(staffUser.PwdHash, req.StaffPIN); err != nil {
			httputil.Unauthorized(c, "Invalid staff PIN")
			return
		}
	}

	// Use the reward service to redeem the issuance
	// This handles:
	// - State validation (must be "issued")
	// - OTP/code validation (if OTP is provided)
	// - Expiry checking
	// - Budget charging
	code := req.OTP
	err := h.rewardService.RedeemIssuance(c.Request.Context(), issuanceUUID, tenantUUID, code)
	if err != nil {
		// Check for specific error types to provide better error messages
		errMsg := err.Error()
		if strings.Contains(errMsg, "cannot redeem") || strings.Contains(errMsg, "must be issued") {
			httputil.BadRequest(c, errMsg, nil)
			return
		}
		if strings.Contains(errMsg, "invalid redemption code") {
			httputil.BadRequest(c, "Invalid OTP code", nil)
			return
		}
		if strings.Contains(errMsg, "expired") {
			httputil.BadRequest(c, "Reward has expired", nil)
			return
		}
		httputil.InternalError(c, "Failed to redeem issuance")
		return
	}

	// Get updated issuance
	updatedIss, err := h.service.GetIssuanceByID(c.Request.Context(), issuanceUUID, tenantUUID)
	if err != nil {
		httputil.InternalError(c, "Failed to get updated issuance")
		return
	}

	c.JSON(200, gin.H{
		"id":          formatUUID(updatedIss.ID),
		"status":      updatedIss.Status,
		"redeemed_at": formatTimestamp(updatedIss.RedeemedAt),
	})
}

// Cancel handles POST /v1/tenants/:tid/issuances/:id/cancel
func (h *IssuancesHandler) Cancel(c *gin.Context) {
	tenantID := c.Param("tid")
	issuanceID := c.Param("id")

	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	if err := httputil.ValidateUUID(issuanceID); err != nil {
		httputil.BadRequest(c, "Invalid issuance ID", nil)
		return
	}

	// Parse UUIDs
	var tenantUUID, issuanceUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}
	if err := issuanceUUID.Scan(issuanceID); err != nil {
		httputil.BadRequest(c, "Invalid issuance ID format", nil)
		return
	}

	// Use reward service to cancel issuance (it handles budget release)
	err := h.rewardService.CancelIssuance(c.Request.Context(), issuanceUUID, tenantUUID)
	if err != nil {
		httputil.InternalError(c, "Failed to cancel issuance")
		return
	}

	// Get updated issuance
	updatedIss, err := h.service.GetIssuanceByID(c.Request.Context(), issuanceUUID, tenantUUID)
	if err != nil {
		httputil.InternalError(c, "Failed to get updated issuance")
		return
	}

	c.JSON(200, gin.H{
		"id":     formatUUID(updatedIss.ID),
		"status": updatedIss.Status,
	})
}
