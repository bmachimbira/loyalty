package config

import (
	"fmt"
	"os"

	"github.com/bmachimbira/loyalty/api/internal/auth"
)

// Config holds all application configuration
type Config struct {
	DatabaseURL string
	JWTSecret   string
	HMACKeys    auth.HMACKeys
	Port        string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		Port:        getEnvOrDefault("PORT", "8080"),
	}

	// Validate required fields
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	// Load HMAC keys (optional)
	hmacKeys, err := auth.LoadHMACKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to load HMAC keys: %w", err)
	}
	cfg.HMACKeys = hmacKeys

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
