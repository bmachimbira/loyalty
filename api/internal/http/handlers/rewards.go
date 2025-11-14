package handlers

import (
	httputil "github.com/bmachimbira/loyalty/api/internal/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RewardsHandler handles reward catalog API endpoints
type RewardsHandler struct {
	pool *pgxpool.Pool
}

// NewRewardsHandler creates a new rewards handler
func NewRewardsHandler(pool *pgxpool.Pool) *RewardsHandler {
	return &RewardsHandler{pool: pool}
}

// CreateRewardRequest represents the request to create a reward
type CreateRewardRequest struct {
	Name       string                 `json:"name" binding:"required"`
	Type       string                 `json:"type" binding:"required"`
	FaceValue  *float64               `json:"face_value"`
	Currency   string                 `json:"currency"`
	Inventory  string                 `json:"inventory" binding:"required"`
	SupplierID *string                `json:"supplier_id"`
	Metadata   map[string]interface{} `json:"metadata"`
	Active     bool                   `json:"active"`
}

// UpdateRewardRequest represents the request to update a reward
type UpdateRewardRequest struct {
	Name      *string                 `json:"name"`
	FaceValue *float64                `json:"face_value"`
	Metadata  *map[string]interface{} `json:"metadata"`
	Active    *bool                   `json:"active"`
}

// Create handles POST /v1/tenants/:tid/reward-catalog
func (h *RewardsHandler) Create(c *gin.Context) {
	tenantID := c.Param("tid")
	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	var req CreateRewardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	// Validate reward type
	if err := httputil.ValidateRewardType(req.Type); err != nil {
		httputil.BadRequest(c, err.Error(), nil)
		return
	}

	// Validate inventory type
	if err := httputil.ValidateInventoryType(req.Inventory); err != nil {
		httputil.BadRequest(c, err.Error(), nil)
		return
	}

	// Validate currency if face value is provided
	if req.FaceValue != nil && req.Currency != "" {
		if err := httputil.ValidateCurrency(req.Currency); err != nil {
			httputil.BadRequest(c, err.Error(), nil)
			return
		}
	}

	// Validate supplier ID if provided
	if req.SupplierID != nil {
		if err := httputil.ValidateUUID(*req.SupplierID); err != nil {
			httputil.BadRequest(c, "Invalid supplier ID", nil)
			return
		}
	}

	// TODO: Once sqlc is generated, use queries.CreateReward()
	c.JSON(201, gin.H{
		"id":          uuid.New().String(),
		"tenant_id":   tenantID,
		"name":        req.Name,
		"type":        req.Type,
		"face_value":  req.FaceValue,
		"currency":    req.Currency,
		"inventory":   req.Inventory,
		"supplier_id": req.SupplierID,
		"metadata":    req.Metadata,
		"active":      req.Active,
		"created_at":  "2025-11-14T00:00:00Z",
	})
}

// List handles GET /v1/tenants/:tid/reward-catalog
func (h *RewardsHandler) List(c *gin.Context) {
	tenantID := c.Param("tid")
	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	// Get filter params
	activeOnly := c.DefaultQuery("active_only", "true")

	// TODO: Once sqlc is generated, use queries.ListRewards()
	c.JSON(200, gin.H{
		"rewards": []gin.H{},
		"total":   0,
		"filters": gin.H{
			"active_only": activeOnly,
		},
	})
}

// Get handles GET /v1/tenants/:tid/reward-catalog/:id
func (h *RewardsHandler) Get(c *gin.Context) {
	tenantID := c.Param("tid")
	rewardID := c.Param("id")

	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	if err := httputil.ValidateUUID(rewardID); err != nil {
		httputil.BadRequest(c, "Invalid reward ID", nil)
		return
	}

	// TODO: Once sqlc is generated, use queries.GetReward()
	c.JSON(200, gin.H{
		"id":         rewardID,
		"tenant_id":  tenantID,
		"name":       "Sample Reward",
		"type":       "discount",
		"face_value": 10.0,
		"currency":   "USD",
		"inventory":  "none",
		"active":     true,
		"created_at": "2025-11-14T00:00:00Z",
	})
}

// Update handles PATCH /v1/tenants/:tid/reward-catalog/:id
func (h *RewardsHandler) Update(c *gin.Context) {
	tenantID := c.Param("tid")
	rewardID := c.Param("id")

	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	if err := httputil.ValidateUUID(rewardID); err != nil {
		httputil.BadRequest(c, "Invalid reward ID", nil)
		return
	}

	var req UpdateRewardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	// TODO: Once sqlc is generated, use queries.UpdateReward()
	c.JSON(200, gin.H{
		"id":         rewardID,
		"tenant_id":  tenantID,
		"updated_at": "2025-11-14T00:00:00Z",
	})
}

// UploadCodes handles POST /v1/tenants/:tid/reward-catalog/:id/upload-codes
// For voucher_code type rewards
func (h *RewardsHandler) UploadCodes(c *gin.Context) {
	tenantID := c.Param("tid")
	rewardID := c.Param("id")

	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	if err := httputil.ValidateUUID(rewardID); err != nil {
		httputil.BadRequest(c, "Invalid reward ID", nil)
		return
	}

	// Parse multipart form for CSV file
	file, err := c.FormFile("file")
	if err != nil {
		httputil.BadRequest(c, "CSV file is required", nil)
		return
	}

	// TODO: Once sqlc is generated:
	// 1. Verify reward exists and is of type voucher_code
	// 2. Parse CSV file
	// 3. Insert voucher codes into voucher_pool table
	// 4. Return count of codes uploaded

	c.JSON(200, gin.H{
		"reward_id":      rewardID,
		"codes_uploaded": 0,
		"file_name":      file.Filename,
	})
}
