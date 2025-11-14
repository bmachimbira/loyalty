package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	mu       sync.RWMutex
	buckets  map[string]*bucket
	rate     int           // requests per window
	window   time.Duration // time window
	cleanupInterval time.Duration
}

// bucket represents a token bucket for rate limiting
type bucket struct {
	tokens     int
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter
// rate: maximum requests per window
// window: time window duration (e.g., 1 minute)
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*bucket),
		rate:    rate,
		window:  window,
		cleanupInterval: 5 * time.Minute,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// cleanup removes old buckets periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, b := range rl.buckets {
			if now.Sub(b.lastRefill) > rl.window*2 {
				delete(rl.buckets, key)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request is allowed for the given key
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.buckets[key]

	if !exists {
		// Create new bucket
		rl.buckets[key] = &bucket{
			tokens:     rl.rate - 1,
			lastRefill: now,
		}
		return true
	}

	// Refill tokens if window has passed
	if now.Sub(b.lastRefill) >= rl.window {
		b.tokens = rl.rate
		b.lastRefill = now
	}

	// Check if tokens available
	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// RateLimit creates a rate limiting middleware
func RateLimit(rate int, window time.Duration) gin.HandlerFunc {
	limiter := NewRateLimiter(rate, window)

	return func(c *gin.Context) {
		// Use client IP as key
		key := c.ClientIP()

		// Check if request is allowed
		if !limiter.Allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests. Please try again later.",
				"retry_after": window.Seconds(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitByTenant creates a rate limiting middleware by tenant ID
func RateLimitByTenant(rate int, window time.Duration) gin.HandlerFunc {
	limiter := NewRateLimiter(rate, window)

	return func(c *gin.Context) {
		// Get tenant ID from context
		tenantID, exists := c.Get("tenant_id")
		if !exists {
			// If no tenant ID, use IP as fallback
			tenantID = c.ClientIP()
		}

		key := tenantID.(string)

		// Check if request is allowed
		if !limiter.Allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests for this tenant. Please try again later.",
				"retry_after": window.Seconds(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitByEndpoint creates different rate limits for different endpoints
func RateLimitByEndpoint(limits map[string]*RateLimiter) gin.HandlerFunc {
	// Default limiter if endpoint not found
	defaultLimiter := NewRateLimiter(100, time.Minute)

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		key := c.ClientIP()

		// Find matching limiter
		var limiter *RateLimiter
		for pattern, l := range limits {
			if matchPattern(path, pattern) {
				limiter = l
				break
			}
		}

		// Use default if no match
		if limiter == nil {
			limiter = defaultLimiter
		}

		// Check if request is allowed
		if !limiter.Allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// matchPattern checks if a path matches a pattern
func matchPattern(path, pattern string) bool {
	// Simple prefix matching for now
	// In production, use a proper path matcher
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(path) >= len(prefix) && path[:len(prefix)] == prefix
	}
	return path == pattern
}
