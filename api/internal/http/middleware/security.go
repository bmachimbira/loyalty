package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds security headers to all responses
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Enable XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// Referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")

		// Permissions policy
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Remove server header
		c.Header("Server", "")

		// HSTS (Strict-Transport-Security) - only in production with HTTPS
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		c.Next()
	}
}

// RequestSizeLimiter limits the size of request bodies
func RequestSizeLimiter(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Limit request body size
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)

		c.Next()

		// Check if request body was too large
		if err := c.Request.Body.Close(); err != nil {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Request body too large",
			})
			c.Abort()
			return
		}
	}
}

// TimeoutMiddleware enforces request timeout
// Note: Gin doesn't have built-in timeout middleware, so this is a placeholder
// In production, use context.WithTimeout or a reverse proxy timeout
func TimeoutMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// This is handled at the HTTP server level (ReadTimeout, WriteTimeout)
		// or at the reverse proxy level (Caddy, nginx)
		c.Next()
	}
}
