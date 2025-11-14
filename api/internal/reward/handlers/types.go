package handlers

import (
	"context"
	"time"

	"github.com/bmachimbira/loyalty/api/internal/db"
)

// RewardHandler defines the interface that all reward type handlers must implement
type RewardHandler interface {
	// Process handles the issuance of a specific reward type
	// Returns ProcessResult with codes/references or error if processing fails
	Process(ctx context.Context, issuance *db.Issuance, reward *db.RewardCatalog) (*ProcessResult, error)
}

// ProcessResult contains the output of reward processing
type ProcessResult struct {
	// Code is the redemption code (OTP, discount code, claim token, etc.)
	Code string

	// ExternalRef is a reference ID from an external provider
	ExternalRef string

	// ExpiresAt is when this reward expires (optional)
	ExpiresAt *time.Time

	// Metadata contains additional reward-specific data
	Metadata map[string]interface{}
}
