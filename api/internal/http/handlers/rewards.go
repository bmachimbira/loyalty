package handlers

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"io"
	"strings"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/bmachimbira/loyalty/api/internal/httputil"
	"github.com/bmachimbira/loyalty/api/internal/rewardcatalog"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RewardsHandler handles reward catalog API endpoints
type RewardsHandler struct {
	pool    *pgxpool.Pool
	service *rewardcatalog.Service
	queries *db.Queries
}

// NewRewardsHandler creates a new rewards handler
func NewRewardsHandler(pool *pgxpool.Pool) *RewardsHandler {
	queries := db.New(pool)
	return &RewardsHandler{
		pool:    pool,
		service: rewardcatalog.NewService(queries),
		queries: queries,
	}
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

	// Parse tenant UUID
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}

	// Prepare parameters
	var faceValue pgtype.Numeric
	if req.FaceValue != nil {
		if err := faceValue.Scan(*req.FaceValue); err != nil {
			httputil.BadRequest(c, "Invalid face value", nil)
			return
		}
	}

	var currency pgtype.Text
	if req.Currency != "" {
		currency = pgtype.Text{String: req.Currency, Valid: true}
	}

	var supplierID pgtype.UUID
	if req.SupplierID != nil {
		if err := supplierID.Scan(*req.SupplierID); err != nil {
			httputil.BadRequest(c, "Invalid supplier ID format", nil)
			return
		}
	}

	// Serialize metadata
	var metadataJSON []byte
	if req.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(req.Metadata)
		if err != nil {
			httputil.BadRequest(c, "Invalid metadata format", nil)
			return
		}
	} else {
		metadataJSON = []byte("{}")
	}

	// Create reward using service
	reward, err := h.service.CreateReward(c.Request.Context(), db.CreateRewardParams{
		TenantID:   tenantUUID,
		Name:       req.Name,
		Type:       req.Type,
		FaceValue:  faceValue,
		Currency:   currency,
		Inventory:  req.Inventory,
		SupplierID: supplierID,
		Metadata:   metadataJSON,
		Active:     req.Active,
	})
	if err != nil {
		httputil.InternalError(c, "Failed to create reward")
		return
	}

	// Parse metadata back
	var metadata map[string]interface{}
	if len(reward.Metadata) > 0 {
		json.Unmarshal(reward.Metadata, &metadata)
	}

	c.JSON(201, gin.H{
		"id":          formatUUID(reward.ID),
		"tenant_id":   formatUUID(reward.TenantID),
		"name":        reward.Name,
		"type":        reward.Type,
		"face_value":  reward.FaceValue.Int.String(),
		"currency":    reward.Currency.String,
		"inventory":   reward.Inventory,
		"supplier_id": formatUUID(reward.SupplierID),
		"metadata":    metadata,
		"active":      reward.Active,
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

	// Parse tenant UUID
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}

	// List rewards using service
	rewards, err := h.service.ListRewards(c.Request.Context(), tenantUUID, activeOnly == "true")
	if err != nil {
		httputil.InternalError(c, "Failed to list rewards")
		return
	}

	// Format response
	rewardsList := make([]gin.H, len(rewards))
	for i, reward := range rewards {
		var metadata map[string]interface{}
		if len(reward.Metadata) > 0 {
			json.Unmarshal(reward.Metadata, &metadata)
		}

		rewardsList[i] = gin.H{
			"id":          formatUUID(reward.ID),
			"tenant_id":   formatUUID(reward.TenantID),
			"name":        reward.Name,
			"type":        reward.Type,
			"face_value":  reward.FaceValue.Int.String(),
			"currency":    reward.Currency.String,
			"inventory":   reward.Inventory,
			"supplier_id": formatUUID(reward.SupplierID),
			"metadata":    metadata,
			"active":      reward.Active,
		}
	}

	c.JSON(200, gin.H{
		"data":  rewardsList,
		"total": len(rewards),
		"limit": 0,
		"page":  0,
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

	// Parse UUIDs
	var tenantUUID, rewardUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}
	if err := rewardUUID.Scan(rewardID); err != nil {
		httputil.BadRequest(c, "Invalid reward ID format", nil)
		return
	}

	// Get reward using service
	reward, err := h.service.GetRewardByID(c.Request.Context(), rewardUUID, tenantUUID)
	if err != nil {
		httputil.NotFound(c, "Reward not found")
		return
	}

	// Parse metadata
	var metadata map[string]interface{}
	if len(reward.Metadata) > 0 {
		json.Unmarshal(reward.Metadata, &metadata)
	}

	c.JSON(200, gin.H{
		"id":          formatUUID(reward.ID),
		"tenant_id":   formatUUID(reward.TenantID),
		"name":        reward.Name,
		"type":        reward.Type,
		"face_value":  reward.FaceValue.Int.String(),
		"currency":    reward.Currency.String,
		"inventory":   reward.Inventory,
		"supplier_id": formatUUID(reward.SupplierID),
		"metadata":    metadata,
		"active":      reward.Active,
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

	// Parse UUIDs
	var tenantUUID, rewardUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}
	if err := rewardUUID.Scan(rewardID); err != nil {
		httputil.BadRequest(c, "Invalid reward ID format", nil)
		return
	}

	// For now, we only support updating the active status
	if req.Active != nil {
		err := h.service.UpdateRewardStatus(c.Request.Context(), rewardUUID, tenantUUID, *req.Active)
		if err != nil {
			httputil.InternalError(c, "Failed to update reward")
			return
		}
	}

	// Get updated reward
	reward, err := h.service.GetRewardByID(c.Request.Context(), rewardUUID, tenantUUID)
	if err != nil {
		httputil.InternalError(c, "Failed to get updated reward")
		return
	}

	// Parse metadata
	var metadata map[string]interface{}
	if len(reward.Metadata) > 0 {
		json.Unmarshal(reward.Metadata, &metadata)
	}

	c.JSON(200, gin.H{
		"id":         formatUUID(reward.ID),
		"tenant_id":  formatUUID(reward.TenantID),
		"name":       reward.Name,
		"type":       reward.Type,
		"face_value": reward.FaceValue.Int.String(),
		"currency":   reward.Currency.String,
		"inventory":  reward.Inventory,
		"metadata":   metadata,
		"active":     reward.Active,
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

	// Parse UUIDs
	var tenantUUID, rewardUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}
	if err := rewardUUID.Scan(rewardID); err != nil {
		httputil.BadRequest(c, "Invalid reward ID format", nil)
		return
	}

	// Verify reward exists and is of type voucher_code
	reward, err := h.service.GetRewardByID(c.Request.Context(), rewardUUID, tenantUUID)
	if err != nil {
		httputil.NotFound(c, "Reward not found")
		return
	}

	if reward.Type != "voucher_code" {
		httputil.BadRequest(c, "Reward must be of type voucher_code", nil)
		return
	}

	// Open the uploaded file
	fileHandle, err := file.Open()
	if err != nil {
		httputil.InternalError(c, "Failed to open file")
		return
	}
	defer fileHandle.Close()

	// Parse CSV file
	reader := csv.NewReader(bufio.NewReader(fileHandle))
	var codes []string
	var codesUploaded int

	// Read CSV line by line
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			httputil.BadRequest(c, "Failed to parse CSV file", err.Error())
			return
		}

		// Skip empty lines
		if len(record) == 0 || strings.TrimSpace(record[0]) == "" {
			continue
		}

		// Get code from first column and trim whitespace
		code := strings.TrimSpace(record[0])
		if code != "" {
			codes = append(codes, code)
		}
	}

	// Insert voucher codes
	for _, code := range codes {
		_, err := h.queries.InsertVoucherCode(c.Request.Context(), db.InsertVoucherCodeParams{
			TenantID: tenantUUID,
			RewardID: rewardUUID,
			Code:     code,
		})
		if err != nil {
			// Log error but continue with other codes
			// In production, you might want to collect errors and return them
			continue
		}
		codesUploaded++
	}

	c.JSON(200, gin.H{
		"reward_id":      formatUUID(rewardUUID),
		"codes_uploaded": codesUploaded,
		"file_name":      file.Filename,
		"total_codes":    len(codes),
	})
}
