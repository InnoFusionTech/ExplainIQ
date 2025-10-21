package main

import (
	"fmt"
	"os"

	"github.com/explainiq/agent/internal/config"
)

func main() {
	fmt.Println("🔧 ExplainIQ Environment Setup")
	fmt.Println("==============================")

	// Load environment variables
	fmt.Println("📁 Loading environment variables...")
	if err := config.LoadEnvFiles(); err != nil {
		fmt.Printf("❌ Failed to load .env file: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Environment variables loaded successfully!")

	// Check required variables
	fmt.Println("\n🔍 Checking required environment variables...")

	required := []string{
		"GEMINI_API_KEY",
		"EXPLAINIQ_PROJECT_ID",
		"EXPLAINIQ_REGION",
		"JWT_SECRET",
	}

	allSet := true
	for _, key := range required {
		value := os.Getenv(key)
		if value == "" || value == "your-actual-api-key-here" || value == "your-project-id" {
			fmt.Printf("❌ %s is not set or using default value\n", key)
			allSet = false
		} else {
			// Mask the value for security
			masked := value
			if len(value) > 10 {
				masked = value[:10] + "..."
			}
			fmt.Printf("✅ %s = %s\n", key, masked)
		}
	}

	if !allSet {
		fmt.Println("\n❌ Some required environment variables are missing!")
		fmt.Println("   Please edit your .env file and set the required values.")
		os.Exit(1)
	}

	fmt.Println("\n🎉 All environment variables are properly configured!")
	fmt.Println("   You can now start the services with: .\\start-services.ps1")
}
