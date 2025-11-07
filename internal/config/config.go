package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds application configuration
type Config struct {
	Port             string
	GinMode          string
	LogLevel         string
	ServiceURL       string
	GCPProjectID     string
	ElasticURL       string
	ElasticAPIKey    string
	GeminiAPIKey     string
	ShutdownTimeout  time.Duration
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	cfg := &Config{
		Port:            getEnv("PORT", "8080"),
		GinMode:         getEnv("GIN_MODE", "release"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		ServiceURL:       getEnv("SERVICE_URL", ""),
		GCPProjectID:    getEnv("GCP_PROJECT_ID", ""),
		ElasticURL:      getEnv("ELASTIC_URL", "http://elasticsearch:9200"),
		ElasticAPIKey:   getEnv("ELASTIC_API_KEY", ""),
		GeminiAPIKey:    getEnv("GEMINI_API_KEY", ""),
		ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", 30*time.Second),
		ReadTimeout:     getDurationEnv("READ_TIMEOUT", 15*time.Second),
		WriteTimeout:    getDurationEnv("WRITE_TIMEOUT", 5*time.Minute), // Increased for long-running tasks like critic
	}

	return cfg
}

// LoadForService loads configuration for a specific service
func LoadForService(defaultPort string, defaultServiceURL string) *Config {
	cfg := Load()
	
	if cfg.Port == "8080" && defaultPort != "" {
		cfg.Port = getEnv("PORT", defaultPort)
	}
	
	if cfg.ServiceURL == "" && defaultServiceURL != "" {
		cfg.ServiceURL = getEnv("SERVICE_URL", defaultServiceURL)
	}

	return cfg
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getDurationEnv gets a duration from an environment variable or returns a default
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// getIntEnv gets an integer from an environment variable or returns a default
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.GeminiAPIKey == "" {
		return fmt.Errorf("GEMINI_API_KEY is required")
	}
	return nil
}

// ValidateOptional validates optional configuration (doesn't fail on missing keys)
func (c *Config) ValidateOptional() []string {
	var warnings []string
	
	if c.GeminiAPIKey == "" {
		warnings = append(warnings, "GEMINI_API_KEY not set - AI features may not work")
	}
	
	if c.GCPProjectID == "" {
		warnings = append(warnings, "GCP_PROJECT_ID not set - cost tracking disabled")
	}
	
	return warnings
}



