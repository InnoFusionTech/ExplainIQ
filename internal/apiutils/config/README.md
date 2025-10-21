# Config Package

The `config` package provides configuration management for the ExplainIQ platform, with validation and environment variable loading.

## Features

- **Environment Variable Loading**: Loads configuration from `EXPLAINIQ_*` environment variables
- **Validation**: Validates required fields and formats
- **Secret Manager Integration**: Helper methods for Google Cloud Secret Manager paths
- **Production Detection**: Determines if running in production mode
- **Default Values**: Provides sensible defaults for optional configuration

## Usage

### Basic Usage

```go
import "github.com/creduntvitam/explainiq/internal/apiutils/config"

// Load configuration from environment
cfg, err := config.FromEnv()
if err != nil {
    log.Fatal(err)
}

// Validate configuration
if err := cfg.Validate(); err != nil {
    log.Fatal(err)
}

// Use configuration
fmt.Printf("Project: %s, Region: %s\n", cfg.ProjectID, cfg.Region)
```

### Environment Variables

The following environment variables are required:

- `EXPLAINIQ_PROJECT_ID`: Google Cloud Project ID
- `EXPLAINIQ_REGION`: Google Cloud Region (e.g., us-central1)
- `EXPLAINIQ_GEMINI_ENDPOINT`: Gemini API endpoint URL
- `EXPLAINIQ_ELASTIC_URL`: Elasticsearch cluster URL
- `EXPLAINIQ_ELASTIC_API_KEY`: Elasticsearch API key
- `EXPLAINIQ_BUCKET`: Google Cloud Storage bucket name
- `EXPLAINIQ_AUTH_AUDIENCE`: JWT authentication audience

Optional variables:

- `EXPLAINIQ_LOG_LEVEL`: Log level (debug, info, warn, error) - defaults to "info"

### Secret Manager Integration

```go
// Get Secret Manager path for a secret
secretPath := cfg.GetSecretManagerPath("elastic-api-key")
// Returns: "projects/my-project/secrets/elastic-api-key/versions/latest"
```

### Production Detection

```go
if cfg.IsProduction() {
    // Production-specific configuration
    log.SetLevel(log.InfoLevel)
} else {
    // Development configuration
    log.SetLevel(log.DebugLevel)
}
```

## Configuration Structure

```go
type Config struct {
    ProjectID      string // Google Cloud Project ID
    Region         string // Google Cloud Region
    LogLevel       string // Log level (debug, info, warn, error)
    GeminiEndpoint string // Gemini API endpoint
    ElasticURL     string // Elasticsearch URL
    ElasticAPIKey  string // Elasticsearch API key
    Bucket         string // GCS bucket name
    AuthAudience   string // JWT auth audience
}
```

## Validation

The package validates:

- **Required Fields**: All required environment variables must be present
- **Log Level**: Must be one of: debug, info, warn, error
- **Project ID**: Must be valid GCP project ID format
- **Region**: Must be valid GCP region format
- **Bucket Name**: Must be valid GCS bucket name format

## Error Handling

The `FromEnv()` function returns descriptive errors for:

- Missing required environment variables
- Invalid log level values
- Configuration validation failures

Example error messages:

```
missing required environment variables: EXPLAINIQ_PROJECT_ID, EXPLAINIQ_REGION
invalid log level 'invalid', must be one of: debug, info, warn, error
invalid project ID format: invalid-project-id
```

## Testing

Run tests with:

```bash
go test ./internal/apiutils/config
go test -v ./internal/apiutils/config  # Verbose output
```

## Security

- Never commit actual API keys or secrets to version control
- Use Google Cloud Secret Manager for production secrets
- The `EXPLAINIQ_ELASTIC_API_KEY` should be stored in Secret Manager
- Grant appropriate IAM permissions for Secret Manager access

