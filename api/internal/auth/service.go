package auth

import (
	"context"
	"errors"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
)

// Service handles authentication business logic
type Service struct {
	queries   *db.Queries
	jwtSecret string
}

// NewService creates a new auth service
func NewService(queries *db.Queries, jwtSecret string) *Service {
	return &Service{
		queries:   queries,
		jwtSecret: jwtSecret,
	}
}

// LoginResult contains the result of a successful login
type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	User         UserInfo
}

// UserInfo represents user information
type UserInfo struct {
	ID       string
	Email    string
	FullName string
	Role     string
	TenantID string
}

// Login authenticates a user and returns tokens
func (s *Service) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	// Find user by email
	user, err := s.queries.GetStaffUserByEmailOnly(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Verify password
	if err := ComparePassword(user.PwdHash, password); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generate tokens
	accessToken, err := GenerateToken(
		user.ID.String(),
		user.TenantID.String(),
		user.Email,
		user.Role,
		s.jwtSecret,
	)
	if err != nil {
		return nil, err
	}

	refreshToken, err := CreateRefreshToken(
		user.ID.String(),
		user.TenantID.String(),
		user.Email,
		user.Role,
		s.jwtSecret,
	)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
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
	}, nil
}

// RefreshToken exchanges a refresh token for new tokens
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	return RefreshAccessToken(refreshToken, s.jwtSecret)
}

// GetUserInfo retrieves user information from token claims
func (s *Service) GetUserInfo(ctx context.Context, userID, tenantID string) (*UserInfo, error) {
	// Parse UUIDs
	var userUUID, tenantUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return nil, ErrUserNotFound
	}
	if err := tenantUUID.Scan(tenantID); err != nil {
		return nil, ErrUserNotFound
	}

	// Get user from database
	user, err := s.queries.GetStaffUserByID(ctx, db.GetStaffUserByIDParams{
		ID:       userUUID,
		TenantID: tenantUUID,
	})
	if err != nil {
		return nil, ErrUserNotFound
	}

	return &UserInfo{
		ID:       user.ID.String(),
		Email:    user.Email,
		FullName: user.FullName,
		Role:     user.Role,
		TenantID: user.TenantID.String(),
	}, nil
}
