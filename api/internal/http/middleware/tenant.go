package middleware

import (
	"context"

	httputil "github.com/bmachimbira/loyalty/api/internal/http"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TenantContext extracts tenant_id from JWT claims and sets PostgreSQL session variable
// This enables Row-Level Security (RLS) to automatically filter data by tenant
func TenantContext(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, exists := c.Get(TenantIDKey)
		if !exists {
			httputil.BadRequest(c, "Tenant ID not found in token", nil)
			c.Abort()
			return
		}

		tid := tenantID.(string)

		// Get a connection from the pool
		conn, err := pool.Acquire(c.Request.Context())
		if err != nil {
			httputil.InternalError(c, "Failed to acquire database connection")
			c.Abort()
			return
		}
		defer conn.Release()

		// Set PostgreSQL session variable for RLS
		_, err = conn.Exec(context.Background(), "SET app.tenant_id = $1", tid)
		if err != nil {
			httputil.InternalError(c, "Failed to set tenant context")
			c.Abort()
			return
		}

		// Store the connection in context for handlers to use
		c.Set("db_conn", conn)

		c.Next()
	}
}
