package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"
)

// ContextKey is the type used for context keys
type ContextKey string

const (
	// RequestIDKey is the context key for request ID
	RequestIDKey ContextKey = "request_id"
	// TenantIDKey is the context key for tenant ID
	TenantIDKey ContextKey = "tenant_id"
	// UserIDKey is the context key for user ID
	UserIDKey ContextKey = "user_id"
)

// Logger wraps slog.Logger with additional context handling
type Logger struct {
	*slog.Logger
}

// New creates a new logger instance based on environment configuration
func New() *Logger {
	var handler slog.Handler

	// Get log level from environment
	level := getLogLevel()

	// Get log format from environment (json or text)
	format := os.Getenv("LOG_FORMAT")
	if format == "" {
		format = "json" // Default to JSON in production
	}

	// Create handler based on format
	handlerOpts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug, // Add source file info in debug mode
	}

	if format == "text" {
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// NewWithWriter creates a logger with a custom writer (useful for testing)
func NewWithWriter(w io.Writer, level slog.Level) *Logger {
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
	})

	return &Logger{
		Logger: slog.New(handler),
	}
}

// getLogLevel parses the LOG_LEVEL environment variable
func getLogLevel() slog.Level {
	levelStr := os.Getenv("LOG_LEVEL")

	switch levelStr {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithContext creates a logger with values from context
func (l *Logger) WithContext(ctx context.Context) *slog.Logger {
	attrs := make([]slog.Attr, 0, 3)

	// Add request ID if present
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		if id, ok := requestID.(string); ok && id != "" {
			attrs = append(attrs, slog.String("request_id", id))
		}
	}

	// Add tenant ID if present
	if tenantID := ctx.Value(TenantIDKey); tenantID != nil {
		if id, ok := tenantID.(string); ok && id != "" {
			attrs = append(attrs, slog.String("tenant_id", id))
		}
	}

	// Add user ID if present
	if userID := ctx.Value(UserIDKey); userID != nil {
		if id, ok := userID.(string); ok && id != "" {
			attrs = append(attrs, slog.String("user_id", id))
		}
	}

	if len(attrs) > 0 {
		return l.Logger.With(attrs...)
	}

	return l.Logger
}

// WithFields creates a logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *slog.Logger {
	attrs := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}
	return l.Logger.With(attrs...)
}

// LogRequest logs an HTTP request
func (l *Logger) LogRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration, fields map[string]interface{}) {
	logger := l.WithContext(ctx)

	// Base attributes
	attrs := []interface{}{
		"method", method,
		"path", path,
		"status", statusCode,
		"duration_ms", duration.Milliseconds(),
	}

	// Add additional fields
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}

	// Log at appropriate level based on status code
	if statusCode >= 500 {
		logger.Error("HTTP request", attrs...)
	} else if statusCode >= 400 {
		logger.Warn("HTTP request", attrs...)
	} else {
		logger.Info("HTTP request", attrs...)
	}
}

// LogError logs an error with context
func (l *Logger) LogError(ctx context.Context, msg string, err error, fields map[string]interface{}) {
	logger := l.WithContext(ctx)

	attrs := []interface{}{"error", err.Error()}

	// Add additional fields
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}

	logger.Error(msg, attrs...)
}

// LogDebug logs a debug message with context
func (l *Logger) LogDebug(ctx context.Context, msg string, fields map[string]interface{}) {
	logger := l.WithContext(ctx)

	attrs := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}

	logger.Debug(msg, attrs...)
}

// LogInfo logs an info message with context
func (l *Logger) LogInfo(ctx context.Context, msg string, fields map[string]interface{}) {
	logger := l.WithContext(ctx)

	attrs := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}

	logger.Info(msg, attrs...)
}

// LogWarn logs a warning message with context
func (l *Logger) LogWarn(ctx context.Context, msg string, fields map[string]interface{}) {
	logger := l.WithContext(ctx)

	attrs := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}

	logger.Warn(msg, attrs...)
}
