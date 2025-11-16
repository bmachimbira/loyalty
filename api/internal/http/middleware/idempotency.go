package middleware

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const IdempotencyKeyHeader = "Idempotency-Key"

// Simple in-memory cache for idempotency keys
// In production, this should be backed by Redis or database
type idempotencyCache struct {
	mu    sync.RWMutex
	cache map[string]*cachedResponse
}

type cachedResponse struct {
	status  int
	body    []byte
	headers map[string]string
	expires time.Time
}

var cache = &idempotencyCache{
	cache: make(map[string]*cachedResponse),
}

// IdempotencyCheck middleware for POST requests with Idempotency-Key header
// Returns cached response if the key has been seen before
func IdempotencyCheck() gin.HandlerFunc {
	// Start cleanup goroutine
	go cleanupExpiredEntries()

	return func(c *gin.Context) {
		// Only check for POST requests
		if c.Request.Method != "POST" {
			c.Next()
			return
		}

		idempotencyKey := c.GetHeader(IdempotencyKeyHeader)
		if idempotencyKey == "" {
			// No idempotency key provided, continue normally
			c.Next()
			return
		}

		// Create a unique key combining tenant + idempotency key
		tenantID, _ := c.Get(TenantIDKey)
		cacheKey := tenantID.(string) + ":" + idempotencyKey

		// Check if we've seen this key before
		cache.mu.RLock()
		cached, exists := cache.cache[cacheKey]
		cache.mu.RUnlock()

		if exists && time.Now().Before(cached.expires) {
			// Return cached response
			for key, value := range cached.headers {
				c.Header(key, value)
			}
			c.Data(cached.status, "application/json", cached.body)
			c.Abort()
			return
		}

		// Capture response
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           []byte{},
		}
		c.Writer = writer

		c.Next()

		// Cache the response (expires in 24 hours)
		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			headers := make(map[string]string)
			for key, values := range c.Writer.Header() {
				if len(values) > 0 {
					headers[key] = values[0]
				}
			}

			cache.mu.Lock()
			cache.cache[cacheKey] = &cachedResponse{
				status:  c.Writer.Status(),
				body:    writer.body,
				headers: headers,
				expires: time.Now().Add(24 * time.Hour),
			}
			cache.mu.Unlock()
		}
	}
}

// responseWriter captures the response body
type responseWriter struct {
	gin.ResponseWriter
	body []byte
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteString(s string) (int, error) {
	w.body = append(w.body, []byte(s)...)
	return w.ResponseWriter.WriteString(s)
}

func (w *responseWriter) WriteJSON(obj interface{}) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	w.body = append(w.body, data...)
	w.ResponseWriter.Header().Set("Content-Type", "application/json")
	_, err = w.ResponseWriter.Write(data)
	return err
}

// cleanupExpiredEntries removes expired entries from cache every hour
func cleanupExpiredEntries() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		cache.mu.Lock()
		now := time.Now()
		for key, cached := range cache.cache {
			if now.After(cached.expires) {
				delete(cache.cache, key)
			}
		}
		cache.mu.Unlock()
	}
}

// ClearIdempotencyCache clears all cached responses (for testing)
func ClearIdempotencyCache() {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.cache = make(map[string]*cachedResponse)
}
