package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// LoadEnvFile loads environment variables from a .env file
func LoadEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", filename, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}

		// Set environment variable
		os.Setenv(key, value)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading %s: %w", filename, err)
	}

	return nil
}

// LoadEnvFiles attempts to load environment variables from multiple files
// in order of preference: .env.local, .env
func LoadEnvFiles() error {
	// Try .env.local first (for local overrides)
	if err := LoadEnvFile(".env.local"); err != nil {
		// .env.local is optional, so we continue
	}

	// Try .env (main environment file)
	if err := LoadEnvFile(".env"); err != nil {
		// .env is also optional for development
		return nil
	}

	return nil
}

