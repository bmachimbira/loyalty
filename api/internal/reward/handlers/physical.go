package handlers

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/bmachimbira/loyalty/api/internal/rewardtypes"
)

// PhysicalItemHandler handles physical item rewards
type PhysicalItemHandler struct{}

// NewPhysicalItemHandler creates a new physical item handler
func NewPhysicalItemHandler() *PhysicalItemHandler {
	return &PhysicalItemHandler{}
}

// Process generates a collection/claim token for physical items
// The customer presents this token at a pickup location to collect the item
func (h *PhysicalItemHandler) Process(ctx context.Context, issuance *db.Issuance, rewardCatalog *db.RewardCatalog) (*ProcessResult, error) {
	// Parse metadata
	var meta rewardtypes.PhysicalItemMetadata
	if err := json.Unmarshal(rewardCatalog.Metadata, &meta); err != nil {
		return nil, fmt.Errorf("invalid physical item metadata: %w", err)
	}

	// Validate metadata
	if meta.ItemName == "" {
		return nil, fmt.Errorf("item name is required")
	}

	// Default collection period if not specified (30 days)
	if meta.CollectionPeriod <= 0 {
		meta.CollectionPeriod = 30
	}

	// Generate unique collection token
	token, err := generateClaimToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate claim token: %w", err)
	}

	// Calculate collection expiry
	expiresAt := time.Now().AddDate(0, 0, meta.CollectionPeriod)

	// Prepare result metadata
	resultMeta := map[string]interface{}{
		"item_name":          meta.ItemName,
		"collection_period":  meta.CollectionPeriod,
		"collection_token":   token,
	}

	if len(meta.PickupLocations) > 0 {
		resultMeta["pickup_locations"] = meta.PickupLocations
	}

	// Add fulfillment instructions
	resultMeta["instructions"] = fmt.Sprintf(
		"Present code %s at any pickup location within %d days to collect your %s",
		token, meta.CollectionPeriod, meta.ItemName,
	)

	return &ProcessResult{
		Code:      token,
		ExpiresAt: &expiresAt,
		Metadata:  resultMeta,
	}, nil
}

// generateClaimToken generates a cryptographically secure numeric token
// Format: 6 digits for easy verification by store staff
func generateClaimToken() (string, error) {
	const (
		digits = "0123456789"
		length = 6
	)

	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	for i := range b {
		b[i] = digits[int(b[i])%len(digits)]
	}

	return string(b), nil
}
