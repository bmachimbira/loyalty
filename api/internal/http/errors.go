package http

import (
	"github.com/gin-gonic/gin"
)

// Error codes
const (
	ErrCodeInvalidRequest   = "invalid_request"
	ErrCodeUnauthorized     = "unauthorized"
	ErrCodeForbidden        = "forbidden"
	ErrCodeNotFound         = "not_found"
	ErrCodeConflict         = "conflict"
	ErrCodeBudgetExceeded   = "budget_exceeded"
	ErrCodeRateLimited      = "rate_limited"
	ErrCodeInternalError    = "internal_error"
	ErrCodeValidationFailed = "validation_failed"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details any    `json:"details,omitempty"`
}

// RespondError sends a standardized error response
func RespondError(c *gin.Context, status int, code, message string, details any) {
	c.JSON(status, ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	})
}

// BadRequest sends a 400 error
func BadRequest(c *gin.Context, message string, details any) {
	RespondError(c, 400, ErrCodeInvalidRequest, message, details)
}

// Unauthorized sends a 401 error
func Unauthorized(c *gin.Context, message string) {
	RespondError(c, 401, ErrCodeUnauthorized, message, nil)
}

// Forbidden sends a 403 error
func Forbidden(c *gin.Context, message string) {
	RespondError(c, 403, ErrCodeForbidden, message, nil)
}

// NotFound sends a 404 error
func NotFound(c *gin.Context, message string) {
	RespondError(c, 404, ErrCodeNotFound, message, nil)
}

// Conflict sends a 409 error
func Conflict(c *gin.Context, message string, details any) {
	RespondError(c, 409, ErrCodeConflict, message, details)
}

// BudgetExceeded sends a 422 error for budget exceeded
func BudgetExceeded(c *gin.Context, message string) {
	RespondError(c, 422, ErrCodeBudgetExceeded, message, nil)
}

// RateLimited sends a 429 error
func RateLimited(c *gin.Context, message string) {
	RespondError(c, 429, ErrCodeRateLimited, message, nil)
}

// InternalError sends a 500 error
func InternalError(c *gin.Context, message string) {
	RespondError(c, 500, ErrCodeInternalError, message, nil)
}

// ValidationError sends a 400 error with validation details
func ValidationError(c *gin.Context, details any) {
	RespondError(c, 400, ErrCodeValidationFailed, "Validation failed", details)
}
