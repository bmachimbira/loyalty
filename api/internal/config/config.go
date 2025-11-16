package config

import (
	"fmt"
	"os"

	"github.com/bmachimbira/loyalty/api/internal/auth"
	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	DatabaseURL string
	JWTSecret   string
	HMACKeys    auth.HMACKeys
	Port        string

	// WhatsApp Business API configuration
	WhatsAppVerifyToken   string
	WhatsAppAppSecret     string
	WhatsAppPhoneNumberID string
	WhatsAppAccessToken   string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if file doesn't exist)
	// Try current directory first, then parent directory (for project root)
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load("../.env")
	}

	cfg := &Config{
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		JWTSecret:             os.Getenv("JWT_SECRET"),
		Port:                  getEnvOrDefault("PORT", "8080"),
		WhatsAppVerifyToken:   os.Getenv("WHATSAPP_VERIFY_TOKEN"),
		WhatsAppAppSecret:     os.Getenv("WHATSAPP_APP_SECRET"),
		WhatsAppPhoneNumberID: os.Getenv("WHATSAPP_PHONE_NUMBER_ID"),
		WhatsAppAccessToken:   os.Getenv("WHATSAPP_ACCESS_TOKEN"),
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

	// WhatsApp configuration is optional (not all deployments will use it)
	// Log warning if not configured
	if cfg.WhatsAppPhoneNumberID == "" || cfg.WhatsAppAccessToken == "" {
		// WhatsApp not configured, but that's okay for development
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
