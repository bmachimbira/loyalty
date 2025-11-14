package handlers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/bmachimbira/loyalty/api/internal/rewardtypes"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestDiscountHandler_Process(t *testing.T) {
	handler := &DiscountHandler{}
	ctx := context.Background()

	// Create test data
	issuance := &db.Issuance{
		ID:         pgtype.UUID{Valid: true},
		TenantID:   pgtype.UUID{Valid: true},
		CustomerID: pgtype.UUID{Valid: true},
	}

	metadata := rewardtypes.DiscountMetadata{
		DiscountType: "percent",
		Amount:       10.0,
		MinBasket:    50.0,
		ValidDays:    30,
	}
	metaBytes, _ := json.Marshal(metadata)

	rewardCatalog := &db.RewardCatalog{
		ID:       pgtype.UUID{Valid: true},
		Type:     "discount",
		Metadata: metaBytes,
	}

	// Process
	result, err := handler.Process(ctx, issuance, rewardCatalog)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Verify result
	if result.Code == "" {
		t.Error("Expected code to be generated")
	}

	if len(result.Code) != 8 {
		t.Errorf("Expected code length 8, got %d", len(result.Code))
	}

	if result.ExpiresAt == nil {
		t.Error("Expected ExpiresAt to be set")
	}

	if result.ExpiresAt.Before(time.Now()) {
		t.Error("ExpiresAt should be in the future")
	}

	// Verify metadata
	if result.Metadata["discount_type"] != "percent" {
		t.Error("Metadata discount_type mismatch")
	}
}

func TestDiscountHandler_InvalidType(t *testing.T) {
	handler := &DiscountHandler{}
	ctx := context.Background()

	issuance := &db.Issuance{}
	metadata := rewardtypes.DiscountMetadata{
		DiscountType: "invalid",
		Amount:       10.0,
	}
	metaBytes, _ := json.Marshal(metadata)

	rewardCatalog := &db.RewardCatalog{
		Type:     "discount",
		Metadata: metaBytes,
	}

	_, err := handler.Process(ctx, issuance, rewardCatalog)
	if err == nil {
		t.Error("Expected error for invalid discount type")
	}
}

func TestPointsCreditHandler_Process(t *testing.T) {
	handler := NewPointsCreditHandler()
	ctx := context.Background()

	issuance := &db.Issuance{}
	metadata := rewardtypes.PointsMetadata{
		PointsAmount: 100,
		PointsType:   "loyalty",
	}
	metaBytes, _ := json.Marshal(metadata)

	rewardCatalog := &db.RewardCatalog{
		Type:     "points_credit",
		Metadata: metaBytes,
	}

	result, err := handler.Process(ctx, issuance, rewardCatalog)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Points don't need a code
	if result.Code != "" {
		t.Error("Expected no code for points")
	}

	// Verify metadata
	if result.Metadata["points_amount"] != 100 {
		t.Error("Metadata points_amount mismatch")
	}
}

func TestPhysicalItemHandler_Process(t *testing.T) {
	handler := NewPhysicalItemHandler()
	ctx := context.Background()

	issuance := &db.Issuance{}
	metadata := rewardtypes.PhysicalItemMetadata{
		ItemName:         "T-Shirt",
		CollectionPeriod: 14,
		PickupLocations:  []string{"Store A", "Store B"},
	}
	metaBytes, _ := json.Marshal(metadata)

	rewardCatalog := &db.RewardCatalog{
		Type:     "physical_item",
		Metadata: metaBytes,
	}

	result, err := handler.Process(ctx, issuance, rewardCatalog)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Should have a collection token
	if result.Code == "" {
		t.Error("Expected collection token to be generated")
	}

	if len(result.Code) != 6 {
		t.Errorf("Expected token length 6, got %d", len(result.Code))
	}

	// Should have expiry
	if result.ExpiresAt == nil {
		t.Error("Expected ExpiresAt to be set")
	}

	// Verify metadata
	if result.Metadata["item_name"] != "T-Shirt" {
		t.Error("Metadata item_name mismatch")
	}
}

func TestGenerateDiscountCode(t *testing.T) {
	// Generate multiple codes to check uniqueness and format
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code, err := generateDiscountCode()
		if err != nil {
			t.Fatalf("Failed to generate code: %v", err)
		}

		if len(code) != 8 {
			t.Errorf("Expected code length 8, got %d", len(code))
		}

		// Check for duplicates (very unlikely with crypto/rand)
		if codes[code] {
			t.Errorf("Duplicate code generated: %s", code)
		}
		codes[code] = true

		// Verify characters
		for _, c := range code {
			validChars := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
			if !contains(validChars, byte(c)) {
				t.Errorf("Invalid character in code: %c", c)
			}
		}
	}
}

func TestGenerateClaimToken(t *testing.T) {
	// Generate multiple tokens to check format
	for i := 0; i < 100; i++ {
		token, err := generateClaimToken()
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		if len(token) != 6 {
			t.Errorf("Expected token length 6, got %d", len(token))
		}

		// Verify all characters are digits
		for _, c := range token {
			if c < '0' || c > '9' {
				t.Errorf("Invalid character in token: %c", c)
			}
		}
	}
}

func contains(s string, b byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return true
		}
	}
	return false
}
