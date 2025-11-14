package rules

import (
	"sync"
	"time"

	"github.com/bmachimbira/loyalty/api/internal/db"
)

// RuleCache is a thread-safe in-memory cache for active rules
type RuleCache struct {
	mu    sync.RWMutex
	items map[string]*cacheItem
	ttl   time.Duration
}

// cacheItem represents a cached entry
type cacheItem struct {
	rules     []db.Rule
	expiresAt time.Time
}

// NewRuleCache creates a new rule cache with the specified TTL
func NewRuleCache(ttl time.Duration) *RuleCache {
	cache := &RuleCache{
		items: make(map[string]*cacheItem),
		ttl:   ttl,
	}

	// Start background cleanup goroutine
	go cache.cleanupExpired()

	return cache
}

// Get retrieves rules from cache if they exist and haven't expired
func (c *RuleCache) Get(key string) ([]db.Rule, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[key]
	if !ok {
		return nil, false
	}

	// Check if expired
	if time.Now().After(item.expiresAt) {
		return nil, false
	}

	return item.rules, true
}

// Set stores rules in the cache with the configured TTL
func (c *RuleCache) Set(key string, rules []db.Rule) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cacheItem{
		rules:     rules,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Delete removes a specific key from the cache
func (c *RuleCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear removes all entries from the cache
func (c *RuleCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*cacheItem)
}

// Size returns the number of entries in the cache
func (c *RuleCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// cleanupExpired periodically removes expired entries from the cache
func (c *RuleCache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.expiresAt) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}
