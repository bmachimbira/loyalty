package handlers

import (
	"github.com/bmachimbira/loyalty/api/internal/auth"
	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/bmachimbira/loyalty/api/internal/httputil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	pool      *pgxpool.Pool
	queries   *db.Queries
	jwtSecret string
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(pool *pgxpool.Pool, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		pool:      pool,
		queries:   db.New(pool),
		jwtSecret: jwtSecret,
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

	// Find user by email (across all tenants)
	user, err := h.queries.GetStaffUserByEmailOnly(c.Request.Context(), req.Email)
	if err != nil {
		httputil.Unauthorized(c, "Invalid email or password")
		return
	}

	// Verify password
	if err := auth.ComparePassword(user.PwdHash, req.Password); err != nil {
		httputil.Unauthorized(c, "Invalid email or password")
		return
	}

	// Generate tokens
	accessToken, err := auth.GenerateToken(
		user.ID.String(),
		user.TenantID.String(),
		user.Email,
		user.Role,
		h.jwtSecret,
	)
	if err != nil {
		httputil.InternalError(c, "Failed to generate access token")
		return
	}

	refreshToken, err := auth.CreateRefreshToken(
		user.ID.String(),
		user.TenantID.String(),
		user.Email,
		user.Role,
		h.jwtSecret,
	)
	if err != nil {
		httputil.InternalError(c, "Failed to generate refresh token")
		return
	}

	c.JSON(200, LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900, // 15 minutes
		User: UserInfo{
			ID:       user.ID.String(),
			Email:    user.Email,
			FullName: user.FullName,
			Role:     user.Role,
			TenantID: user.TenantID.String(),
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
	tokenPair, err := auth.RefreshAccessToken(req.RefreshToken, h.jwtSecret)
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
	email, _ := c.Get("email")
	role, _ := c.Get("role")

	c.JSON(200, UserInfo{
		ID:       userID.(string),
		Email:    email.(string),
		Role:     role.(string),
		TenantID: tenantID.(string),
	})
}
