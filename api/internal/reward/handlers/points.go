package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/bmachimbira/loyalty/api/internal/rewardtypes"
)

// PointsCreditHandler handles points credit rewards
type PointsCreditHandler struct{}

// NewPointsCreditHandler creates a new points credit handler
func NewPointsCreditHandler() *PointsCreditHandler {
	return &PointsCreditHandler{}
}

// Process credits points to the customer's balance
// Points are stored in the issuance record and can be redeemed later
// No external action is needed at issuance time
func (h *PointsCreditHandler) Process(ctx context.Context, issuance *db.Issuance, rewardCatalog *db.RewardCatalog) (*ProcessResult, error) {
	// Parse metadata
	var meta rewardtypes.PointsMetadata
	if err := json.Unmarshal(rewardCatalog.Metadata, &meta); err != nil {
		return nil, fmt.Errorf("invalid points metadata: %w", err)
	}

	// Validate points amount
	if meta.PointsAmount <= 0 {
		return nil, fmt.Errorf("points amount must be positive")
	}

	// Default points type if not specified
	if meta.PointsType == "" {
		meta.PointsType = "loyalty"
	}

	// Prepare result metadata
	resultMeta := map[string]interface{}{
		"points_amount": meta.PointsAmount,
		"points_type":   meta.PointsType,
	}

	// Points are credited immediately upon issuance
	// The customer can redeem them later for rewards
	// No code is needed for points - they're automatically credited
	return &ProcessResult{
		Metadata: resultMeta,
	}, nil
}
