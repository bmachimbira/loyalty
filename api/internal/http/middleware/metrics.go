package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// MetricsMiddleware logs request metrics for monitoring
func MetricsMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Get status code
		statusCode := c.Writer.Status()

		// Get client IP
		clientIP := c.ClientIP()

		// Get request size
		requestSize := c.Request.ContentLength

		// Get response size
		responseSize := c.Writer.Size()

		// Get user agent
		userAgent := c.Request.UserAgent()

		// Get path and method
		path := c.Request.URL.Path
		method := c.Request.Method

		// Get tenant ID and request ID from context if available
		tenantID, _ := c.Get("tenant_id")
		requestID, _ := c.Get("request_id")

		// Log metrics based on status code severity
		fields := []interface{}{
			"method", method,
			"path", path,
			"status", statusCode,
			"duration_ms", duration.Milliseconds(),
			"duration_us", duration.Microseconds(),
			"client_ip", clientIP,
			"user_agent", userAgent,
		}

		// Add optional fields
		if requestID != nil {
			fields = append(fields, "request_id", requestID)
		}

		if tenantID != nil {
			fields = append(fields, "tenant_id", tenantID)
		}

		if requestSize > 0 {
			fields = append(fields, "request_size_bytes", requestSize)
		}

		if responseSize > 0 {
			fields = append(fields, "response_size_bytes", responseSize)
		}

		// Add error message if present
		if len(c.Errors) > 0 {
			fields = append(fields, "errors", c.Errors.String())
		}

		// Log at appropriate level based on status code
		if statusCode >= 500 {
			logger.Error("HTTP request completed", fields...)
		} else if statusCode >= 400 {
			logger.Warn("HTTP request completed", fields...)
		} else {
			logger.Info("HTTP request completed", fields...)
		}

		// In a production environment with Prometheus, you would also:
		// - Increment request counter by method, path, and status
		// - Record duration histogram by method and path
		// - Update active requests gauge
		// Example (if using prometheus/client_golang):
		// httpRequestsTotal.WithLabelValues(method, path, fmt.Sprint(statusCode)).Inc()
		// httpRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
	}
}

// PerformanceMetrics tracks detailed performance metrics
type PerformanceMetrics struct {
	RequestsTotal      int64
	RequestsInFlight   int64
	RequestDurations   []time.Duration
	ErrorsTotal        int64
	LastRequestTime    time.Time
}

// Note: For production Prometheus integration, create a separate file:
// api/internal/metrics/prometheus.go with:
//
// - Request counter by method, path, status
// - Request duration histogram by method, path
// - Active requests gauge
// - Business metrics counters (events, rewards, etc.)
