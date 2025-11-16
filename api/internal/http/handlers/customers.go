package handlers

import (
	"github.com/bmachimbira/loyalty/api/internal/customer"
	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/bmachimbira/loyalty/api/internal/httputil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CustomersHandler handles customer-related API endpoints
type CustomersHandler struct {
	pool    *pgxpool.Pool
	service *customer.Service
}

// NewCustomersHandler creates a new customers handler
func NewCustomersHandler(pool *pgxpool.Pool) *CustomersHandler {
	queries := db.New(pool)
	return &CustomersHandler{
		pool:    pool,
		service: customer.NewService(queries),
	}
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

	// Parse tenant UUID
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}

	// Create customer using service
	customer, err := h.service.CreateCustomer(c.Request.Context(), db.CreateCustomerParams{
		TenantID:    tenantUUID,
		PhoneE164:   pgtype.Text{String: req.PhoneE164, Valid: req.PhoneE164 != ""},
		ExternalRef: pgtype.Text{String: req.ExternalRef, Valid: req.ExternalRef != ""},
	})
	if err != nil {
		httputil.InternalError(c, "Failed to create customer")
		return
	}

	c.JSON(201, gin.H{
		"id":           formatUUID(customer.ID),
		"tenant_id":    formatUUID(customer.TenantID),
		"phone_e164":   customer.PhoneE164.String,
		"external_ref": customer.ExternalRef.String,
		"status":       customer.Status,
		"created_at":   formatTimestamp(customer.CreatedAt),
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

	// Get customer using service
	customer, err := h.service.GetCustomerByID(c.Request.Context(), customerUUID, tenantUUID)
	if err != nil {
		httputil.NotFound(c, "Customer not found")
		return
	}

	c.JSON(200, gin.H{
		"id":           formatUUID(customer.ID),
		"tenant_id":    formatUUID(customer.TenantID),
		"phone_e164":   customer.PhoneE164.String,
		"external_ref": customer.ExternalRef.String,
		"status":       customer.Status,
		"created_at":   formatTimestamp(customer.CreatedAt),
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

	// Parse tenant UUID
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}

	// List customers using service
	customers, total, err := h.service.ListCustomers(c.Request.Context(), tenantUUID, limit, offset)
	if err != nil {
		httputil.InternalError(c, "Failed to list customers")
		return
	}

	// Format response
	customersList := make([]gin.H, len(customers))
	for i, customer := range customers {
		customersList[i] = gin.H{
			"id":           formatUUID(customer.ID),
			"tenant_id":    formatUUID(customer.TenantID),
			"phone_e164":   customer.PhoneE164.String,
			"external_ref": customer.ExternalRef.String,
			"status":       customer.Status,
			"created_at":   formatTimestamp(customer.CreatedAt),
		}
	}

	c.JSON(200, gin.H{
		"customers": customersList,
		"total":     total,
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

	// Update customer status using service
	err := h.service.UpdateCustomerStatus(c.Request.Context(), customerUUID, tenantUUID, req.Status)
	if err != nil {
		httputil.InternalError(c, "Failed to update customer status")
		return
	}

	// Get updated customer
	customer, err := h.service.GetCustomerByID(c.Request.Context(), customerUUID, tenantUUID)
	if err != nil {
		httputil.InternalError(c, "Failed to get updated customer")
		return
	}

	c.JSON(200, gin.H{
		"id":         formatUUID(customer.ID),
		"tenant_id":  formatUUID(customer.TenantID),
		"status":     customer.Status,
		"updated_at": formatTimestamp(customer.CreatedAt),
	})
}
