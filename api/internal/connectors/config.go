package connectors

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config contains configuration for all external connectors
type Config struct {
	Airtime AirtimeConfig
	Data    DataConfig
}

// AirtimeConfig contains configuration for airtime provider
type AirtimeConfig struct {
	Enabled bool
	BaseURL string
	APIKey  string
	Secret  string
	Timeout time.Duration
}

// DataConfig contains configuration for data provider (similar to airtime)
type DataConfig struct {
	Enabled bool
	BaseURL string
	APIKey  string
	Secret  string
	Timeout time.Duration
}

// LoadConfig loads connector configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{}

	// Load airtime provider config
	airtimeConfig, err := loadAirtimeConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load airtime config: %w", err)
	}
	config.Airtime = airtimeConfig

	// Load data provider config
	dataConfig, err := loadDataConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load data config: %w", err)
	}
	config.Data = dataConfig

	return config, nil
}

func loadAirtimeConfig() (AirtimeConfig, error) {
	config := AirtimeConfig{
		Timeout: 30 * time.Second, // Default timeout
	}

	// Check if enabled
	enabledStr := os.Getenv("AIRTIME_PROVIDER_ENABLED")
	if enabledStr == "" {
		enabledStr = "false"
	}
	enabled, err := strconv.ParseBool(enabledStr)
	if err != nil {
		return config, fmt.Errorf("invalid AIRTIME_PROVIDER_ENABLED value: %w", err)
	}
	config.Enabled = enabled

	if !config.Enabled {
		return config, nil
	}

	// Load required fields
	config.BaseURL = os.Getenv("AIRTIME_PROVIDER_URL")
	if config.BaseURL == "" {
		return config, fmt.Errorf("AIRTIME_PROVIDER_URL is required when airtime provider is enabled")
	}

	config.APIKey = os.Getenv("AIRTIME_PROVIDER_KEY")
	if config.APIKey == "" {
		return config, fmt.Errorf("AIRTIME_PROVIDER_KEY is required when airtime provider is enabled")
	}

	config.Secret = os.Getenv("AIRTIME_PROVIDER_SECRET")
	if config.Secret == "" {
		return config, fmt.Errorf("AIRTIME_PROVIDER_SECRET is required when airtime provider is enabled")
	}

	// Optional timeout override
	if timeoutStr := os.Getenv("AIRTIME_PROVIDER_TIMEOUT"); timeoutStr != "" {
		timeoutSec, err := strconv.Atoi(timeoutStr)
		if err != nil {
			return config, fmt.Errorf("invalid AIRTIME_PROVIDER_TIMEOUT value: %w", err)
		}
		config.Timeout = time.Duration(timeoutSec) * time.Second
	}

	return config, nil
}

func loadDataConfig() (DataConfig, error) {
	config := DataConfig{
		Timeout: 30 * time.Second, // Default timeout
	}

	// Check if enabled
	enabledStr := os.Getenv("DATA_PROVIDER_ENABLED")
	if enabledStr == "" {
		enabledStr = "false"
	}
	enabled, err := strconv.ParseBool(enabledStr)
	if err != nil {
		return config, fmt.Errorf("invalid DATA_PROVIDER_ENABLED value: %w", err)
	}
	config.Enabled = enabled

	if !config.Enabled {
		return config, nil
	}

	// Load required fields
	config.BaseURL = os.Getenv("DATA_PROVIDER_URL")
	if config.BaseURL == "" {
		return config, fmt.Errorf("DATA_PROVIDER_URL is required when data provider is enabled")
	}

	config.APIKey = os.Getenv("DATA_PROVIDER_KEY")
	if config.APIKey == "" {
		return config, fmt.Errorf("DATA_PROVIDER_KEY is required when data provider is enabled")
	}

	config.Secret = os.Getenv("DATA_PROVIDER_SECRET")
	if config.Secret == "" {
		return config, fmt.Errorf("DATA_PROVIDER_SECRET is required when data provider is enabled")
	}

	// Optional timeout override
	if timeoutStr := os.Getenv("DATA_PROVIDER_TIMEOUT"); timeoutStr != "" {
		timeoutSec, err := strconv.Atoi(timeoutStr)
		if err != nil {
			return config, fmt.Errorf("invalid DATA_PROVIDER_TIMEOUT value: %w", err)
		}
		config.Timeout = time.Duration(timeoutSec) * time.Second
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validation is already done in Load functions
	return nil
}
