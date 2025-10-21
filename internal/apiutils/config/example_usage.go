package config

import (
	"log"
	"os"
)

// ExampleUsage demonstrates how to use the config package
func ExampleUsage() {
	// Load configuration from environment variables
	config, err := FromEnv()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// Use the configuration
	log.Printf("Project ID: %s", config.ProjectID)
	log.Printf("Region: %s", config.Region)
	log.Printf("Log Level: %s", config.LogLevel)
	log.Printf("Is Production: %t", config.IsProduction())

	// Get Secret Manager path for a secret
	secretPath := config.GetSecretManagerPath("elastic-api-key")
	log.Printf("Secret Manager path: %s", secretPath)
}

// ExampleServiceSetup shows how to set up a service with the config
func ExampleServiceSetup() {
	// Set up test environment variables for demonstration
	os.Setenv("EXPLAINIQ_PROJECT_ID", "my-project-123")
	os.Setenv("EXPLAINIQ_REGION", "us-central1")
	os.Setenv("EXPLAINIQ_LOG_LEVEL", "info")
	os.Setenv("EXPLAINIQ_GEMINI_ENDPOINT", "https://generativelanguage.googleapis.com/v1beta")
	os.Setenv("EXPLAINIQ_ELASTIC_URL", "https://my-elastic.com:9200")
	os.Setenv("EXPLAINIQ_ELASTIC_API_KEY", "my-api-key")
	os.Setenv("EXPLAINIQ_BUCKET", "my-bucket")
	os.Setenv("EXPLAINIQ_AUTH_AUDIENCE", "my-audience")

	// Load and validate configuration
	config, err := FromEnv()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Now you can use the config throughout your service
	_ = config // Use config in your service initialization
}

