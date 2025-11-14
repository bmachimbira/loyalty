package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/bmachimbira/loyalty/api/internal/db"
	httputil "github.com/bmachimbira/loyalty/api/internal/http"
	"github.com/bmachimbira/loyalty/api/internal/rules"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EventsHandler handles event-related API endpoints
type EventsHandler struct {
	pool        *pgxpool.Pool
	queries     *db.Queries
	rulesEngine *rules.Engine
	logger      *slog.Logger
}

// NewEventsHandler creates a new events handler
func NewEventsHandler(pool *pgxpool.Pool, rulesEngine *rules.Engine, logger *slog.Logger) *EventsHandler {
	return &EventsHandler{
		pool:        pool,
		queries:     db.New(pool),
		rulesEngine: rulesEngine,
		logger:      logger,
	}
}

// CreateEventRequest represents the request to create an event
type CreateEventRequest struct {
	CustomerID string                 `json:"customer_id" binding:"required"`
	EventType  string                 `json:"event_type" binding:"required"`
	Properties map[string]interface{} `json:"properties"`
	OccurredAt *time.Time             `json:"occurred_at"`
	Source     string                 `json:"source"`
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

	// Parse tenant and customer UUIDs
	var tenantUUID, customerUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}
	if err := customerUUID.Scan(req.CustomerID); err != nil {
		httputil.BadRequest(c, "Invalid customer ID format", nil)
		return
	}

	// Check if event with this idempotency key already exists
	existingEvent, err := h.queries.GetEventByIdemKey(c.Request.Context(), db.GetEventByIdemKeyParams{
		TenantID:       tenantUUID,
		IdempotencyKey: idempotencyKey,
	})
	if err == nil {
		// Event already exists, return it
		h.logger.Info("returning existing event for idempotency key",
			"idempotency_key", idempotencyKey,
			"event_id", existingEvent.ID,
		)
		c.JSON(200, formatEventResponse(existingEvent, nil))
		return
	} else if err != pgx.ErrNoRows {
		// Unexpected error
		h.logger.Error("failed to check idempotency", "error", err)
		httputil.InternalServerError(c, "Failed to check idempotency", nil)
		return
	}

	// Serialize properties
	var propertiesJSON []byte
	if req.Properties != nil {
		propertiesJSON, err = json.Marshal(req.Properties)
		if err != nil {
			httputil.BadRequest(c, "Invalid properties format", nil)
			return
		}
	} else {
		propertiesJSON = []byte("{}")
	}

	// Set occurred_at (default to now if not provided)
	var occurredAt pgtype.Timestamptz
	if req.OccurredAt != nil {
		if err := occurredAt.Scan(*req.OccurredAt); err != nil {
			httputil.BadRequest(c, "Invalid occurred_at format", nil)
			return
		}
	} else {
		if err := occurredAt.Scan(time.Now()); err != nil {
			httputil.InternalServerError(c, "Failed to set timestamp", nil)
			return
		}
	}

	// Set source (default to "api" if not provided)
	source := req.Source
	if source == "" {
		source = "api"
	}

	// Create event
	event, err := h.queries.InsertEvent(c.Request.Context(), db.InsertEventParams{
		TenantID:       tenantUUID,
		CustomerID:     customerUUID,
		EventType:      req.EventType,
		Properties:     propertiesJSON,
		OccurredAt:     occurredAt,
		Source:         source,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		h.logger.Error("failed to create event", "error", err)
		httputil.InternalServerError(c, "Failed to create event", nil)
		return
	}

	h.logger.Info("event created",
		"event_id", event.ID,
		"event_type", event.EventType,
		"customer_id", event.CustomerID,
	)

	// Process event through rules engine
	issuances, err := h.rulesEngine.ProcessEvent(c.Request.Context(), event)
	if err != nil {
		// Log error but don't fail the request - event was created successfully
		h.logger.Error("rules engine processing failed",
			"event_id", event.ID,
			"error", err,
		)
		// Return event without issuances
		c.JSON(201, formatEventResponse(event, nil))
		return
	}

	// Return event with issuances
	c.JSON(201, formatEventResponse(event, issuances))
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

// Helper functions

// formatEventResponse formats an event and its issuances for the API response
func formatEventResponse(event db.Event, issuances []db.Issuance) gin.H {
	// Parse properties
	var properties map[string]interface{}
	if len(event.Properties) > 0 {
		json.Unmarshal(event.Properties, &properties)
	}

	response := gin.H{
		"id":              formatUUID(event.ID),
		"tenant_id":       formatUUID(event.TenantID),
		"customer_id":     formatUUID(event.CustomerID),
		"event_type":      event.EventType,
		"properties":      properties,
		"occurred_at":     formatTimestamp(event.OccurredAt),
		"source":          event.Source,
		"idempotency_key": event.IdempotencyKey,
		"created_at":      formatTimestamp(event.CreatedAt),
	}

	// Add issuances if any
	if len(issuances) > 0 {
		issuancesList := make([]gin.H, len(issuances))
		for i, issuance := range issuances {
			issuancesList[i] = gin.H{
				"id":          formatUUID(issuance.ID),
				"reward_id":   formatUUID(issuance.RewardID),
				"campaign_id": formatUUID(issuance.CampaignID),
				"status":      issuance.Status,
				"currency":    issuance.Currency.String,
				"face_amount": issuance.FaceAmount.Int.String(),
				"issued_at":   formatTimestamp(issuance.IssuedAt),
			}
		}
		response["issuances"] = issuancesList
	} else {
		response["issuances"] = []gin.H{}
	}

	return response
}

// formatUUID converts pgtype.UUID to string
func formatUUID(uuid pgtype.UUID) string {
	if !uuid.Valid {
		return ""
	}
	b := uuid.Bytes
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// formatTimestamp converts pgtype.Timestamptz to ISO8601 string
func formatTimestamp(ts pgtype.Timestamptz) string {
	if !ts.Valid {
		return ""
	}
	return ts.Time.Format(time.RFC3339)
}
