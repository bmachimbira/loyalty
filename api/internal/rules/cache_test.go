package rules

import (
	"testing"
	"time"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestRuleCache_GetSet(t *testing.T) {
	cache := NewRuleCache(1 * time.Second)

	// Test cache miss
	_, found := cache.Get("test-key")
	if found {
		t.Error("expected cache miss, got hit")
	}

	// Test cache set and hit
	rules := []db.Rule{
		{
			ID:   pgtype.UUID{Valid: true},
			Name: "test-rule",
		},
	}
	cache.Set("test-key", rules)

	retrieved, found := cache.Get("test-key")
	if !found {
		t.Error("expected cache hit, got miss")
	}
	if len(retrieved) != 1 || retrieved[0].Name != "test-rule" {
		t.Error("retrieved rules don't match")
	}
}

func TestRuleCache_Expiry(t *testing.T) {
	cache := NewRuleCache(100 * time.Millisecond)

	rules := []db.Rule{
		{Name: "test-rule"},
	}
	cache.Set("test-key", rules)

	// Should be available immediately
	_, found := cache.Get("test-key")
	if !found {
		t.Error("expected cache hit immediately after set")
	}

	// Wait for expiry
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, found = cache.Get("test-key")
	if found {
		t.Error("expected cache miss after expiry")
	}
}

func TestRuleCache_Delete(t *testing.T) {
	cache := NewRuleCache(1 * time.Second)

	rules := []db.Rule{{Name: "test-rule"}}
	cache.Set("test-key", rules)

	// Verify it's there
	_, found := cache.Get("test-key")
	if !found {
		t.Error("expected cache hit before delete")
	}

	// Delete it
	cache.Delete("test-key")

	// Verify it's gone
	_, found = cache.Get("test-key")
	if found {
		t.Error("expected cache miss after delete")
	}
}

func TestRuleCache_Clear(t *testing.T) {
	cache := NewRuleCache(1 * time.Second)

	cache.Set("key1", []db.Rule{{Name: "rule1"}})
	cache.Set("key2", []db.Rule{{Name: "rule2"}})

	if cache.Size() != 2 {
		t.Errorf("expected size 2, got %d", cache.Size())
	}

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", cache.Size())
	}
}
