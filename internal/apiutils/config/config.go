package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds the application configuration
type Config struct {
	ProjectID      string
	Region         string
	LogLevel       string
	GeminiEndpoint string
	ElasticURL     string
	ElasticAPIKey  string
	Bucket         string
	AuthAudience   string
}

// FromEnv loads configuration from environment variables
// Validates that all required variables are present
func FromEnv() (Config, error) {
	config := Config{
		ProjectID:      getEnv("EXPLAINIQ_PROJECT_ID"),
		Region:         getEnv("EXPLAINIQ_REGION"),
		LogLevel:       getEnvWithDefault("EXPLAINIQ_LOG_LEVEL", "info"),
		GeminiEndpoint: getEnv("EXPLAINIQ_GEMINI_ENDPOINT"),
		ElasticURL:     getEnv("EXPLAINIQ_ELASTIC_URL"),
		ElasticAPIKey:  getEnv("EXPLAINIQ_ELASTIC_API_KEY"),
		Bucket:         getEnv("EXPLAINIQ_BUCKET"),
		AuthAudience:   getEnv("EXPLAINIQ_AUTH_AUDIENCE"),
	}

	// Validate required fields
	required := map[string]string{
		"EXPLAINIQ_PROJECT_ID":      config.ProjectID,
		"EXPLAINIQ_REGION":          config.Region,
		"EXPLAINIQ_GEMINI_ENDPOINT": config.GeminiEndpoint,
		"EXPLAINIQ_ELASTIC_URL":     config.ElasticURL,
		"EXPLAINIQ_ELASTIC_API_KEY": config.ElasticAPIKey,
		"EXPLAINIQ_BUCKET":          config.Bucket,
		"EXPLAINIQ_AUTH_AUDIENCE":   config.AuthAudience,
	}

	var missing []string
	for envVar, value := range required {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, envVar)
		}
	}

	if len(missing) > 0 {
		return config, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[strings.ToLower(config.LogLevel)] {
		return config, fmt.Errorf("invalid log level '%s', must be one of: debug, info, warn, error", config.LogLevel)
	}

	return config, nil
}

// getEnv gets an environment variable or returns empty string
func getEnv(key string) string {
	return os.Getenv(key)
}

// getEnvWithDefault gets an environment variable or returns the default value
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// IsProduction returns true if the environment is production
func (c *Config) IsProduction() bool {
	return strings.ToLower(c.LogLevel) == "info" || strings.ToLower(c.LogLevel) == "error"
}

// GetSecretManagerPath returns the Secret Manager path for a given secret name
func (c *Config) GetSecretManagerPath(secretName string) string {
	return fmt.Sprintf("projects/%s/secrets/%s/versions/latest", c.ProjectID, secretName)
}

// Validate performs additional validation on the configuration
func (c *Config) Validate() error {
	// Validate project ID format (should be valid GCP project ID)
	if !isValidProjectID(c.ProjectID) {
		return fmt.Errorf("invalid project ID format: %s", c.ProjectID)
	}

	// Validate region format
	if !isValidRegion(c.Region) {
		return fmt.Errorf("invalid region format: %s", c.Region)
	}

	// Validate bucket name format
	if !isValidBucketName(c.Bucket) {
		return fmt.Errorf("invalid bucket name format: %s", c.Bucket)
	}

	return nil
}

// isValidProjectID validates GCP project ID format
func isValidProjectID(projectID string) bool {
	// GCP project IDs must be 6-30 characters, lowercase letters, numbers, and hyphens
	// Cannot start or end with hyphen
	if len(projectID) < 6 || len(projectID) > 30 {
		return false
	}

	// Basic validation - more comprehensive validation would check for valid characters
	return strings.ToLower(projectID) == projectID &&
		!strings.HasPrefix(projectID, "-") &&
		!strings.HasSuffix(projectID, "-")
}

// isValidRegion validates GCP region format
func isValidRegion(region string) bool {
	// Basic validation for GCP region format (e.g., us-central1, europe-west1)
	parts := strings.Split(region, "-")
	if len(parts) < 2 {
		return false
	}

	// Should have format: continent-location[-number] or continent-location-zone
	// Examples: us-central1, europe-west1, asia-southeast1, us-east1
	return len(parts[0]) >= 2 && len(parts[1]) >= 2
}

// isValidBucketName validates GCS bucket name format
func isValidBucketName(bucket string) bool {
	// GCS bucket names must be 3-63 characters, lowercase letters, numbers, dots, and hyphens
	// Cannot start or end with hyphen, cannot have consecutive dots
	if len(bucket) < 3 || len(bucket) > 63 {
		return false
	}

	// Basic validation
	return strings.ToLower(bucket) == bucket &&
		!strings.HasPrefix(bucket, "-") &&
		!strings.HasSuffix(bucket, "-") &&
		!strings.Contains(bucket, "..")
}
