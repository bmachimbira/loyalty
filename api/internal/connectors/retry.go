package connectors

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// RetryConfig contains configuration for retry logic
type RetryConfig struct {
	MaxAttempts     int           // Maximum number of retry attempts
	InitialDelay    time.Duration // Initial delay before first retry
	MaxDelay        time.Duration // Maximum delay between retries
	BackoffMultiplier float64       // Multiplier for exponential backoff
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:     3,
		InitialDelay:    1 * time.Second,
		MaxDelay:        10 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// RetryWithBackoff executes a function with exponential backoff retry logic
func RetryWithBackoff(ctx context.Context, config RetryConfig, fn func() error) error {
	var lastErr error
	delay := config.InitialDelay

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
		}

		// Execute function
		err := fn()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if error is retriable
		if !IsRetriable(err) {
			return fmt.Errorf("non-retriable error: %w", err)
		}

		// Don't sleep after last attempt
		if attempt == config.MaxAttempts {
			break
		}

		// Wait with exponential backoff
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled during backoff: %w", ctx.Err())
		case <-time.After(delay):
		}

		// Calculate next delay with exponential backoff
		delay = time.Duration(float64(delay) * config.BackoffMultiplier)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
	}

	return fmt.Errorf("max retry attempts (%d) exceeded: %w", config.MaxAttempts, lastErr)
}

// IsRetriable determines if an error should be retried
func IsRetriable(err error) bool {
	if err == nil {
		return false
	}

	// Network errors are retriable
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// DNS errors are retriable
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	// HTTP errors - check status code
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return IsRetriableStatusCode(httpErr.StatusCode)
	}

	// Context errors are not retriable
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Circuit breaker errors are not retriable
	if errors.Is(err, ErrCircuitOpen) {
		return false
	}

	// Default to retriable for unknown errors
	return true
}

// IsRetriableStatusCode determines if an HTTP status code should be retried
func IsRetriableStatusCode(statusCode int) bool {
	switch {
	case statusCode >= 500 && statusCode < 600:
		// Server errors are retriable
		return true
	case statusCode == http.StatusTooManyRequests:
		// Rate limit errors are retriable (429)
		return true
	case statusCode == http.StatusRequestTimeout:
		// Request timeout is retriable (408)
		return true
	case statusCode >= 400 && statusCode < 500:
		// Other client errors are not retriable
		return false
	default:
		// Unknown status codes
		return false
	}
}

// HTTPError represents an HTTP error with status code
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// NewHTTPError creates a new HTTP error
func NewHTTPError(statusCode int, message string) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Message:    message,
	}
}
