package whatsapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler handles WhatsApp webhook requests
type Handler struct {
	pool        *pgxpool.Pool
	queries     *db.Queries
	verifyToken string
	appSecret   string
	processor   *MessageProcessor
}

// NewHandler creates a new WhatsApp webhook handler
func NewHandler(pool *pgxpool.Pool, verifyToken, appSecret, phoneNumberID, accessToken string) *Handler {
	queries := db.New(pool)
	sender := NewMessageSender(phoneNumberID, accessToken)
	processor := NewMessageProcessor(pool, queries, sender)

	return &Handler{
		pool:        pool,
		queries:     queries,
		verifyToken: verifyToken,
		appSecret:   appSecret,
		processor:   processor,
	}
}

// Verify handles GET request for webhook verification
// This is called by WhatsApp to verify the webhook endpoint
func (h *Handler) Verify(c *gin.Context) {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	slog.Info("WhatsApp webhook verification request",
		"mode", mode,
		"token_matches", token == h.verifyToken,
	)

	if mode == "subscribe" && token == h.verifyToken {
		slog.Info("WhatsApp webhook verified successfully")
		c.String(http.StatusOK, challenge)
		return
	}

	slog.Warn("WhatsApp webhook verification failed",
		"mode", mode,
		"expected_token", h.verifyToken != "",
	)
	c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
}

// Webhook handles POST request with incoming messages
func (h *Handler) Webhook(c *gin.Context) {
	ctx := c.Request.Context()

	// Get signature from header
	signature := c.GetHeader("X-Hub-Signature-256")
	if signature == "" {
		slog.Warn("WhatsApp webhook missing signature header")
		c.JSON(http.StatusForbidden, gin.H{"error": "Missing signature"})
		return
	}

	// Read body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		slog.Error("Failed to read webhook body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Verify signature
	if !h.verifySignature(body, signature) {
		slog.Warn("WhatsApp webhook signature verification failed",
			"signature", signature,
		)
		c.JSON(http.StatusForbidden, gin.H{"error": "Invalid signature"})
		return
	}

	// Parse webhook payload
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.Error("Failed to parse webhook payload", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	slog.Info("WhatsApp webhook payload received",
		"object", payload.Object,
		"entries", len(payload.Entry),
	)

	// Process messages
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field == "messages" {
				// Process incoming messages
				for _, msg := range change.Value.Messages {
					slog.Info("Processing WhatsApp message",
						"from", msg.From,
						"type", msg.Type,
						"id", msg.ID,
					)

					if err := h.processor.ProcessMessage(ctx, msg); err != nil {
						slog.Error("Failed to process WhatsApp message",
							"error", err,
							"msg_id", msg.ID,
							"from", msg.From,
						)
						// Continue processing other messages even if one fails
					}
				}

				// Log status updates
				for _, status := range change.Value.Statuses {
					slog.Info("WhatsApp message status update",
						"msg_id", status.ID,
						"status", status.Status,
						"recipient", status.RecipientID,
					)
					// TODO: Update message status in database if needed
				}
			}
		}
	}

	// Always return 200 to acknowledge receipt
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// verifySignature verifies the HMAC-SHA256 signature from WhatsApp
func (h *Handler) verifySignature(body []byte, signature string) bool {
	if h.appSecret == "" {
		slog.Warn("WhatsApp app secret not configured, skipping signature verification")
		return true // Allow in development
	}

	// Create HMAC-SHA256 hash
	mac := hmac.New(sha256.New, []byte(h.appSecret))
	mac.Write(body)
	expectedMAC := mac.Sum(nil)
	expected := "sha256=" + hex.EncodeToString(expectedMAC)

	// Compare signatures
	return hmac.Equal([]byte(expected), []byte(signature))
}
