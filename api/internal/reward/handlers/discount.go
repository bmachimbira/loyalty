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

// DiscountHandler handles discount code rewards
type DiscountHandler struct{}

// Process generates a unique discount code and sets expiry
func (h *DiscountHandler) Process(ctx context.Context, issuance *db.Issuance, rewardCatalog *db.RewardCatalog) (*ProcessResult, error) {
	// Generate unique discount code
	code, err := generateDiscountCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate discount code: %w", err)
	}

	// Parse metadata
	var meta rewardtypes.DiscountMetadata
	if err := json.Unmarshal(rewardCatalog.Metadata, &meta); err != nil {
		return nil, fmt.Errorf("invalid discount metadata: %w", err)
	}

	// Validate discount metadata
	if meta.DiscountType != "amount" && meta.DiscountType != "percent" {
		return nil, fmt.Errorf("invalid discount type: %s", meta.DiscountType)
	}

	if meta.Amount <= 0 {
		return nil, fmt.Errorf("discount amount must be positive")
	}

	if meta.ValidDays <= 0 {
		meta.ValidDays = 30 // Default to 30 days
	}

	// Calculate expiry based on valid_days
	expiresAt := time.Now().AddDate(0, 0, meta.ValidDays)

	// Prepare result metadata
	resultMeta := map[string]interface{}{
		"discount_type": meta.DiscountType,
		"amount":        meta.Amount,
		"min_basket":    meta.MinBasket,
		"valid_days":    meta.ValidDays,
	}

	return &ProcessResult{
		Code:      code,
		ExpiresAt: &expiresAt,
		Metadata:  resultMeta,
	}, nil
}

// generateDiscountCode generates a cryptographically secure alphanumeric code
// Uses characters that are easy to read and type (excludes 0, O, 1, I, etc.)
func generateDiscountCode() (string, error) {
	const (
		chars  = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
		length = 8
	)

	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}

	return string(b), nil
}
