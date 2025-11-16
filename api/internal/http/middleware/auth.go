package middleware

import (
	"io"
	"strings"

	"github.com/bmachimbira/loyalty/api/internal/auth"
	"github.com/bmachimbira/loyalty/api/internal/httputil"
	"github.com/gin-gonic/gin"
)

const (
	UserIDKey   = "user_id"
	TenantIDKey = "tenant_id"
	EmailKey    = "email"
	RoleKey     = "role"
)

// RequireAuth validates JWT token and extracts claims
func RequireAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			httputil.Unauthorized(c, "Missing authorization header")
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			httputil.Unauthorized(c, "Invalid authorization header format")
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token
		claims, err := auth.ValidateToken(token, jwtSecret)
		if err != nil {
			httputil.Unauthorized(c, "Invalid or expired token")
			c.Abort()
			return
		}

		// Set claims in context
		c.Set(UserIDKey, claims.UserID)
		c.Set(TenantIDKey, claims.TenantID)
		c.Set(EmailKey, claims.Email)
		c.Set(RoleKey, claims.Role)

		c.Next()
	}
}

// RequireRole checks if user has required role
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get(RoleKey)
		if !exists {
			httputil.Forbidden(c, "Role not found in context")
			c.Abort()
			return
		}

		role := userRole.(string)
		for _, allowedRole := range roles {
			if role == allowedRole {
				c.Next()
				return
			}
		}

		httputil.Forbidden(c, "Insufficient permissions")
		c.Abort()
	}
}

// RequireHMAC validates HMAC signature for server-to-server requests
func RequireHMAC(keys auth.HMACKeys) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-Key")
		timestamp := c.GetHeader("X-Timestamp")
		signature := c.GetHeader("X-Signature")

		// Read body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			httputil.BadRequest(c, "Failed to read request body", nil)
			c.Abort()
			return
		}

		// Validate HMAC
		if err := auth.ValidateHMAC(apiKey, timestamp, signature, string(body), keys); err != nil {
			httputil.Unauthorized(c, "Invalid HMAC signature: "+err.Error())
			c.Abort()
			return
		}

		c.Next()
	}
}
