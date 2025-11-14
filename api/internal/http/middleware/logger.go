package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger middleware logs HTTP requests
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		requestID, _ := c.Get(RequestIDKey)

		log.Printf("[%s] %s %s %s %d %s %s",
			requestID,
			c.Request.Method,
			path,
			query,
			c.Writer.Status(),
			latency,
			c.ClientIP(),
		)
	}
}
