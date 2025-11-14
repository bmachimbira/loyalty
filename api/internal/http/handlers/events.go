package handlers

import (
	httputil "github.com/bmachimbira/loyalty/api/internal/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EventsHandler handles event-related API endpoints
type EventsHandler struct {
	pool *pgxpool.Pool
}

// NewEventsHandler creates a new events handler
func NewEventsHandler(pool *pgxpool.Pool) *EventsHandler {
	return &EventsHandler{pool: pool}
}

// CreateEventRequest represents the request to create an event
type CreateEventRequest struct {
	CustomerID string                 `json:"customer_id" binding:"required"`
	EventType  string                 `json:"event_type" binding:"required"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// Create handles POST /v1/tenants/:tid/events
// Requires Idempotency-Key header
func (h *EventsHandler) Create(c *gin.Context) {
	tenantID := c.Param("tid")
	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	// Validate customer ID
	if err := httputil.ValidateUUID(req.CustomerID); err != nil {
		httputil.BadRequest(c, "Invalid customer ID", nil)
		return
	}

	// Validate event type
	if err := httputil.ValidateEventType(req.EventType); err != nil {
		httputil.BadRequest(c, err.Error(), nil)
		return
	}

	// Check for idempotency key
	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		httputil.BadRequest(c, "Idempotency-Key header is required for event creation", nil)
		return
	}

	// TODO: Once sqlc is generated:
	// 1. Check if event with this idempotency key already exists
	// 2. Insert event record
	// 3. Trigger rules engine evaluation
	// 4. Return event + any triggered issuances

	c.JSON(201, gin.H{
		"id":              uuid.New().String(),
		"tenant_id":       tenantID,
		"customer_id":     req.CustomerID,
		"event_type":      req.EventType,
		"metadata":        req.Metadata,
		"idempotency_key": idempotencyKey,
		"created_at":      "2025-11-14T00:00:00Z",
		"issuances":       []gin.H{},
	})
}

// Get handles GET /v1/tenants/:tid/events/:id
func (h *EventsHandler) Get(c *gin.Context) {
	tenantID := c.Param("tid")
	eventID := c.Param("id")

	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	if err := httputil.ValidateUUID(eventID); err != nil {
		httputil.BadRequest(c, "Invalid event ID", nil)
		return
	}

	// TODO: Once sqlc is generated, use queries.GetEvent()
	c.JSON(200, gin.H{
		"id":          eventID,
		"tenant_id":   tenantID,
		"customer_id": uuid.New().String(),
		"event_type":  "purchase",
		"metadata":    gin.H{},
		"created_at":  "2025-11-14T00:00:00Z",
	})
}

// List handles GET /v1/tenants/:tid/events
func (h *EventsHandler) List(c *gin.Context) {
	tenantID := c.Param("tid")
	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	// Get pagination and filter params
	limit := c.DefaultQuery("limit", "50")
	offset := c.DefaultQuery("offset", "0")
	customerID := c.Query("customer_id")

	if customerID != "" {
		if err := httputil.ValidateUUID(customerID); err != nil {
			httputil.BadRequest(c, "Invalid customer ID", nil)
			return
		}
	}

	// TODO: Once sqlc is generated, use queries.ListEvents()
	c.JSON(200, gin.H{
		"events": []gin.H{},
		"total":  0,
		"limit":  limit,
		"offset": offset,
		"filters": gin.H{
			"customer_id": customerID,
		},
	})
}
