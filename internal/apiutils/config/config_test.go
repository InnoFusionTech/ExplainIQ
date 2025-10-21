package config

import (
	"os"
	"testing"
)

func TestFromEnv(t *testing.T) {
	// Set up test environment variables
	testEnv := map[string]string{
		"EXPLAINIQ_PROJECT_ID":      "test-project-123",
		"EXPLAINIQ_REGION":          "us-central1",
		"EXPLAINIQ_LOG_LEVEL":       "debug",
		"EXPLAINIQ_GEMINI_ENDPOINT": "https://generativelanguage.googleapis.com/v1beta",
		"EXPLAINIQ_ELASTIC_URL":     "https://test-elastic.com:9200",
		"EXPLAINIQ_ELASTIC_API_KEY": "test-api-key",
		"EXPLAINIQ_BUCKET":          "test-bucket",
		"EXPLAINIQ_AUTH_AUDIENCE":   "test-audience",
	}

	// Set environment variables
	for key, value := range testEnv {
		os.Setenv(key, value)
	}
	defer func() {
		// Clean up environment variables
		for key := range testEnv {
			os.Unsetenv(key)
		}
	}()

	// Test successful configuration loading
	config, err := FromEnv()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify all fields are set correctly
	if config.ProjectID != "test-project-123" {
		t.Errorf("Expected ProjectID 'test-project-123', got '%s'", config.ProjectID)
	}
	if config.Region != "us-central1" {
		t.Errorf("Expected Region 'us-central1', got '%s'", config.Region)
	}
	if config.LogLevel != "debug" {
		t.Errorf("Expected LogLevel 'debug', got '%s'", config.LogLevel)
	}
	if config.GeminiEndpoint != "https://generativelanguage.googleapis.com/v1beta" {
		t.Errorf("Expected GeminiEndpoint 'https://generativelanguage.googleapis.com/v1beta', got '%s'", config.GeminiEndpoint)
	}
	if config.ElasticURL != "https://test-elastic.com:9200" {
		t.Errorf("Expected ElasticURL 'https://test-elastic.com:9200', got '%s'", config.ElasticURL)
	}
	if config.ElasticAPIKey != "test-api-key" {
		t.Errorf("Expected ElasticAPIKey 'test-api-key', got '%s'", config.ElasticAPIKey)
	}
	if config.Bucket != "test-bucket" {
		t.Errorf("Expected Bucket 'test-bucket', got '%s'", config.Bucket)
	}
	if config.AuthAudience != "test-audience" {
		t.Errorf("Expected AuthAudience 'test-audience', got '%s'", config.AuthAudience)
	}
}

func TestFromEnvMissingRequired(t *testing.T) {
	// Clear all environment variables
	envVars := []string{
		"EXPLAINIQ_PROJECT_ID",
		"EXPLAINIQ_REGION",
		"EXPLAINIQ_LOG_LEVEL",
		"EXPLAINIQ_GEMINI_ENDPOINT",
		"EXPLAINIQ_ELASTIC_URL",
		"EXPLAINIQ_ELASTIC_API_KEY",
		"EXPLAINIQ_BUCKET",
		"EXPLAINIQ_AUTH_AUDIENCE",
	}

	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}

	// Test that missing required variables return an error
	_, err := FromEnv()
	if err == nil {
		t.Fatal("Expected error for missing required environment variables, got nil")
	}

	// Verify error message contains information about missing variables
	expectedErrorContains := "missing required environment variables"
	if !contains(err.Error(), expectedErrorContains) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedErrorContains, err.Error())
	}
}

func TestFromEnvDefaultLogLevel(t *testing.T) {
	// Set up test environment variables without LOG_LEVEL
	testEnv := map[string]string{
		"EXPLAINIQ_PROJECT_ID":      "test-project-123",
		"EXPLAINIQ_REGION":          "us-central1",
		"EXPLAINIQ_GEMINI_ENDPOINT": "https://generativelanguage.googleapis.com/v1beta",
		"EXPLAINIQ_ELASTIC_URL":     "https://test-elastic.com:9200",
		"EXPLAINIQ_ELASTIC_API_KEY": "test-api-key",
		"EXPLAINIQ_BUCKET":          "test-bucket",
		"EXPLAINIQ_AUTH_AUDIENCE":   "test-audience",
	}

	// Set environment variables
	for key, value := range testEnv {
		os.Setenv(key, value)
	}
	defer func() {
		// Clean up environment variables
		for key := range testEnv {
			os.Unsetenv(key)
		}
	}()

	// Test that default log level is used
	config, err := FromEnv()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config.LogLevel != "info" {
		t.Errorf("Expected default LogLevel 'info', got '%s'", config.LogLevel)
	}
}

func TestConfigValidation(t *testing.T) {
	// Test valid configuration
	config := Config{
		ProjectID:      "valid-project-123",
		Region:         "us-central1",
		LogLevel:       "info",
		GeminiEndpoint: "https://generativelanguage.googleapis.com/v1beta",
		ElasticURL:     "https://test-elastic.com:9200",
		ElasticAPIKey:  "test-api-key",
		Bucket:         "valid-bucket-name",
		AuthAudience:   "test-audience",
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("Expected no validation error, got: %v", err)
	}
}

func TestConfigIsProduction(t *testing.T) {
	// Test production detection
	config := Config{LogLevel: "info"}
	if !config.IsProduction() {
		t.Error("Expected IsProduction() to return true for 'info' log level")
	}

	config.LogLevel = "error"
	if !config.IsProduction() {
		t.Error("Expected IsProduction() to return true for 'error' log level")
	}

	config.LogLevel = "debug"
	if config.IsProduction() {
		t.Error("Expected IsProduction() to return false for 'debug' log level")
	}
}

func TestGetSecretManagerPath(t *testing.T) {
	config := Config{ProjectID: "test-project"}
	path := config.GetSecretManagerPath("test-secret")
	expected := "projects/test-project/secrets/test-secret/versions/latest"
	if path != expected {
		t.Errorf("Expected path '%s', got '%s'", expected, path)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			contains(s[1:], substr))))
}

