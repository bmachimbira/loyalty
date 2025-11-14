package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/bmachimbira/loyalty/api/internal/rewardtypes"
)

// WebhookHandler handles custom webhook notifications
type WebhookHandler struct {
	client *http.Client
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler() *WebhookHandler {
	return &WebhookHandler{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Process calls a custom webhook URL with reward issuance details
func (h *WebhookHandler) Process(ctx context.Context, issuance *db.Issuance, rewardCatalog *db.RewardCatalog) (*ProcessResult, error) {
	// Parse metadata
	var meta rewardtypes.WebhookMetadata
	if err := json.Unmarshal(rewardCatalog.Metadata, &meta); err != nil {
		return nil, fmt.Errorf("invalid webhook metadata: %w", err)
	}

	// Validate webhook URL
	if meta.WebhookURL == "" {
		return nil, fmt.Errorf("webhook URL is required")
	}

	// Prepare webhook payload
	payload := map[string]interface{}{
		"event":       "reward.issued",
		"issuance_id": issuance.ID,
		"customer_id": issuance.CustomerID,
		"reward": map[string]interface{}{
			"id":         rewardCatalog.ID,
			"name":       rewardCatalog.Name,
			"type":       rewardCatalog.Type,
			"face_value": rewardCatalog.FaceValue,
			"currency":   rewardCatalog.Currency,
		},
		"timestamp": time.Now().Unix(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Compute HMAC signature if secret is provided
	var signature string
	if meta.Secret != "" {
		signature = computeHMAC(meta.Secret, payloadBytes)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", meta.WebhookURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ZW-Loyalty-Platform/1.0")

	if signature != "" {
		req.Header.Set("X-Signature-SHA256", signature)
	}

	// Add custom headers from metadata
	if meta.Headers != nil {
		for key, value := range meta.Headers {
			req.Header.Set(key, value)
		}
	}

	// Send request with retry logic
	var resp *http.Response
	var lastErr error

	for attempt := 1; attempt <= 3; attempt++ {
		resp, lastErr = h.client.Do(req)
		if lastErr == nil {
			break
		}

		if attempt < 3 {
			// Exponential backoff: 1s, 2s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				continue
			}
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("webhook request failed after 3 attempts: %w", lastErr)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Prepare result metadata
	resultMeta := map[string]interface{}{
		"webhook_url":     meta.WebhookURL,
		"response_status": resp.StatusCode,
		"response_body":   string(responseBody),
		"timestamp":       time.Now().Unix(),
	}

	return &ProcessResult{
		Metadata: resultMeta,
	}, nil
}

// computeHMAC generates HMAC-SHA256 signature
func computeHMAC(secret string, data []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
