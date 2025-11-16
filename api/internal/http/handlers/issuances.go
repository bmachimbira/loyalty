package handlers

import (
	"github.com/bmachimbira/loyalty/api/internal/httputil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IssuancesHandler handles issuance-related API endpoints
type IssuancesHandler struct {
	pool *pgxpool.Pool
}

// NewIssuancesHandler creates a new issuances handler
func NewIssuancesHandler(pool *pgxpool.Pool) *IssuancesHandler {
	return &IssuancesHandler{pool: pool}
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

	if customerID != "" {
		if err := httputil.ValidateUUID(customerID); err != nil {
			httputil.BadRequest(c, "Invalid customer ID", nil)
			return
		}
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

	// TODO: Once sqlc is generated, use queries.ListIssuances()
	c.JSON(200, gin.H{
		"issuances": []gin.H{},
		"total":     0,
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

	// TODO: Once sqlc is generated, use queries.GetIssuance()
	c.JSON(200, gin.H{
		"id":          issuanceID,
		"tenant_id":   tenantID,
		"customer_id": uuid.New().String(),
		"reward_id":   uuid.New().String(),
		"state":       "issued",
		"created_at":  "2025-11-14T00:00:00Z",
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

	// TODO: Once sqlc is generated:
	// 1. Verify issuance exists and is in 'issued' state
	// 2. Validate OTP or staff PIN
	// 3. Check expiry
	// 4. Transition state to 'redeemed'
	// 5. Charge budget
	// 6. Create audit log entry

	c.JSON(200, gin.H{
		"id":          issuanceID,
		"state":       "redeemed",
		"redeemed_at": "2025-11-14T00:00:00Z",
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

	// TODO: Once sqlc is generated:
	// 1. Verify issuance exists and is in cancellable state
	// 2. Transition state to 'cancelled'
	// 3. Release reserved budget
	// 4. Create audit log entry

	c.JSON(200, gin.H{
		"id":          issuanceID,
		"state":       "cancelled",
		"cancelled_at": "2025-11-14T00:00:00Z",
	})
}
