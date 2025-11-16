package handlers

import (
	"errors"

	"github.com/bmachimbira/loyalty/api/internal/auth"
	"github.com/bmachimbira/loyalty/api/internal/httputil"
	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authService *auth.Service
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *auth.Service) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	User         UserInfo `json:"user"`
}

// UserInfo represents user information in the response
type UserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
	TenantID string `json:"tenant_id"`
}

// RefreshRequest represents the refresh token request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Login handles POST /v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	// Authenticate user
	result, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			httputil.Unauthorized(c, "Invalid email or password")
			return
		}
		httputil.InternalError(c, "Failed to authenticate")
		return
	}

	c.JSON(200, LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
		User: UserInfo{
			ID:       result.User.ID,
			Email:    result.User.Email,
			FullName: result.User.FullName,
			Role:     result.User.Role,
			TenantID: result.User.TenantID,
		},
	})
}

// Refresh handles POST /v1/auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	// Refresh the token
	tokenPair, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		httputil.Unauthorized(c, "Invalid or expired refresh token")
		return
	}

	c.JSON(200, tokenPair)
}

// Me handles GET /v1/auth/me (requires authentication)
func (h *AuthHandler) Me(c *gin.Context) {
	// Extract user info from context (set by RequireAuth middleware)
	userID, _ := c.Get("user_id")
	tenantID, _ := c.Get("tenant_id")

	// Get fresh user info from database
	userInfo, err := h.authService.GetUserInfo(c.Request.Context(), userID.(string), tenantID.(string))
	if err != nil {
		httputil.Unauthorized(c, "User not found")
		return
	}

	c.JSON(200, UserInfo{
		ID:       userInfo.ID,
		Email:    userInfo.Email,
		FullName: userInfo.FullName,
		Role:     userInfo.Role,
		TenantID: userInfo.TenantID,
	})
}
