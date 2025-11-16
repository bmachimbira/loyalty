package handlers

import (
	"github.com/bmachimbira/loyalty/api/internal/httputil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CustomersHandler handles customer-related API endpoints
type CustomersHandler struct {
	pool *pgxpool.Pool
}

// NewCustomersHandler creates a new customers handler
func NewCustomersHandler(pool *pgxpool.Pool) *CustomersHandler {
	return &CustomersHandler{pool: pool}
}

// CreateCustomerRequest represents the request to create a customer
type CreateCustomerRequest struct {
	PhoneE164   string            `json:"phone_e164"`
	ExternalRef string            `json:"external_ref"`
	Metadata    map[string]string `json:"metadata"`
}

// UpdateCustomerStatusRequest represents the request to update customer status
type UpdateCustomerStatusRequest struct {
	Status string `json:"status"`
}

// Create handles POST /v1/tenants/:tid/customers
func (h *CustomersHandler) Create(c *gin.Context) {
	var req CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	// Validate at least one identifier is provided
	if req.PhoneE164 == "" && req.ExternalRef == "" {
		httputil.BadRequest(c, "Either phone_e164 or external_ref is required", nil)
		return
	}

	// Validate E.164 phone format if provided
	if req.PhoneE164 != "" {
		if err := httputil.ValidateE164Phone(req.PhoneE164); err != nil {
			httputil.BadRequest(c, err.Error(), nil)
			return
		}
		req.PhoneE164 = httputil.NormalizeE164Phone(req.PhoneE164)
	}

	tenantID := c.Param("tid")
	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	// TODO: Once sqlc is generated, use queries.CreateCustomer()
	// For now, return a placeholder response
	c.JSON(201, gin.H{
		"id":           uuid.New().String(),
		"tenant_id":    tenantID,
		"phone_e164":   req.PhoneE164,
		"external_ref": req.ExternalRef,
		"status":       "active",
		"created_at":   "2025-11-14T00:00:00Z",
	})
}

// Get handles GET /v1/tenants/:tid/customers/:id
func (h *CustomersHandler) Get(c *gin.Context) {
	tenantID := c.Param("tid")
	customerID := c.Param("id")

	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	if err := httputil.ValidateUUID(customerID); err != nil {
		httputil.BadRequest(c, "Invalid customer ID", nil)
		return
	}

	// TODO: Once sqlc is generated, use queries.GetCustomer()
	c.JSON(200, gin.H{
		"id":           customerID,
		"tenant_id":    tenantID,
		"phone_e164":   "+263771234567",
		"external_ref": "CUST123",
		"status":       "active",
		"created_at":   "2025-11-14T00:00:00Z",
	})
}

// List handles GET /v1/tenants/:tid/customers
func (h *CustomersHandler) List(c *gin.Context) {
	tenantID := c.Param("tid")
	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	// Get pagination params
	limit := c.DefaultQuery("limit", "50")
	offset := c.DefaultQuery("offset", "0")

	// Get search params
	phone := c.Query("phone")
	externalRef := c.Query("external_ref")

	// TODO: Once sqlc is generated, use queries.ListCustomers()
	c.JSON(200, gin.H{
		"customers": []gin.H{},
		"total":     0,
		"limit":     limit,
		"offset":    offset,
		"filters": gin.H{
			"phone":        phone,
			"external_ref": externalRef,
		},
	})
}

// UpdateStatus handles PATCH /v1/tenants/:tid/customers/:id/status
func (h *CustomersHandler) UpdateStatus(c *gin.Context) {
	tenantID := c.Param("tid")
	customerID := c.Param("id")

	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	if err := httputil.ValidateUUID(customerID); err != nil {
		httputil.BadRequest(c, "Invalid customer ID", nil)
		return
	}

	var req UpdateCustomerStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	// Validate status
	validStatuses := map[string]bool{
		"active":    true,
		"suspended": true,
		"deleted":   true,
	}

	if !validStatuses[req.Status] {
		httputil.BadRequest(c, "Invalid status. Must be active, suspended, or deleted", nil)
		return
	}

	// TODO: Once sqlc is generated, use queries.UpdateCustomerStatus()
	c.JSON(200, gin.H{
		"id":         customerID,
		"tenant_id":  tenantID,
		"status":     req.Status,
		"updated_at": "2025-11-14T00:00:00Z",
	})
}
