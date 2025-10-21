# LLM Package

The `llm` package provides comprehensive language model functionality for the ExplainIQ platform, including text embedding generation using Google Vertex AI's text-embedding-004 model.

## Features

- **Vertex AI Integration**: Uses Google's text-embedding-004 model via REST API
- **Batching Support**: Efficiently processes multiple texts in batches
- **Retry Logic**: Automatic retry with exponential backoff for transient failures
- **Error Handling**: Comprehensive error handling with detailed error messages
- **Configuration**: Flexible client configuration for different use cases
- **Authentication**: Support for both environment variables and Application Default Credentials

## Quick Start

### Basic Usage

```go
import "github.com/creduntvitam/explainiq/internal/llm"

// Set up environment variables
os.Setenv("EXPLAINIQ_PROJECT_ID", "your-project-id")
os.Setenv("EXPLAINIQ_REGION", "us-central1")
os.Setenv("GOOGLE_ACCESS_TOKEN", "your-access-token")

// Generate embeddings
ctx := context.Background()
texts := []string{
    "Machine learning is a subset of artificial intelligence.",
    "Deep learning uses neural networks with multiple layers.",
}

embeddings, err := llm.Embed(ctx, texts)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Generated %d embeddings\n", len(embeddings))
```

### Advanced Usage

```go
// Create client with custom configuration
client := llm.NewEmbeddingClient("your-project-id", "us-central1")
client.SetBatchSize(5)        // Maximum batch size
client.SetMaxRetries(3)       // Retry failed requests
client.SetRetryDelay(1 * time.Second) // Initial retry delay
client.SetMaxTokens(3072)     // Maximum tokens per text

// Generate embeddings
embeddings, err := client.Embed(ctx, texts)
```

## API Reference

### Embed Function

```go
func Embed(ctx context.Context, texts []string) ([][]float32, error)
```

Convenience function that creates a client and generates embeddings for the given texts.

**Parameters:**
- `ctx`: Context for request cancellation and timeout
- `texts`: Slice of text strings to embed

**Returns:**
- `[][]float32`: Slice of embedding vectors (768 dimensions each)
- `error`: Error if embedding generation fails

### EmbeddingClient

```go
type EmbeddingClient struct {
    // Private fields
}
```

Client for generating embeddings using Vertex AI.

#### NewEmbeddingClient

```go
func NewEmbeddingClient(projectID, location string) *EmbeddingClient
```

Creates a new embedding client.

**Parameters:**
- `projectID`: Google Cloud project ID
- `location`: Google Cloud region (e.g., "us-central1")

#### Embed

```go
func (c *EmbeddingClient) Embed(ctx context.Context, texts []string) ([][]float32, error)
```

Generates embeddings for the given texts.

#### Configuration Methods

```go
func (c *EmbeddingClient) SetBatchSize(size int)
func (c *EmbeddingClient) SetMaxRetries(retries int)
func (c *EmbeddingClient) SetRetryDelay(delay time.Duration)
func (c *EmbeddingClient) SetMaxTokens(tokens int)
func (c *EmbeddingClient) GetModelInfo() map[string]interface{}
```

## Request/Response Types

### EmbeddingRequest

```go
type EmbeddingRequest struct {
    Instances  []EmbeddingInstance  `json:"instances"`
    Parameters EmbeddingParameters  `json:"parameters"`
}
```

### EmbeddingInstance

```go
type EmbeddingInstance struct {
    Content  string `json:"content"`
    TaskType string `json:"task_type,omitempty"`
    Title    string `json:"title,omitempty"`
}
```

### EmbeddingResponse

```go
type EmbeddingResponse struct {
    Predictions []EmbeddingPrediction `json:"predictions"`
    Metadata    EmbeddingMetadata     `json:"metadata"`
}
```

### EmbeddingPrediction

```go
type EmbeddingPrediction struct {
    Embeddings EmbeddingValue `json:"embeddings"`
    Stats      EmbeddingStats `json:"stats,omitempty"`
}
```

### EmbeddingValue

```go
type EmbeddingValue struct {
    Values []float32 `json:"values"`
}
```

## Configuration

### Environment Variables

- `EXPLAINIQ_PROJECT_ID`: Google Cloud project ID
- `EXPLAINIQ_REGION`: Google Cloud region
- `GOOGLE_ACCESS_TOKEN`: Access token for authentication (development)
- `GOOGLE_APPLICATION_CREDENTIALS`: Path to service account key (production)

### Default Settings

- **Model**: text-embedding-004
- **Output Dimensions**: 768
- **Batch Size**: 5 (Vertex AI limit)
- **Max Retries**: 3
- **Retry Delay**: 1 second
- **Max Tokens**: 3072 per text
- **Timeout**: 60 seconds

## Batching

The client automatically batches requests to optimize performance:

- **Batch Size**: Configurable (default: 5, max: 5 for Vertex AI)
- **Automatic Batching**: Large text arrays are split into batches
- **Parallel Processing**: Each batch is processed independently
- **Error Handling**: Batch failures don't affect other batches

Example:
```go
// 10 texts will be processed in 2 batches of 5 each
texts := make([]string, 10)
embeddings, err := client.Embed(ctx, texts)
```

## Retry Logic

The client implements robust retry logic:

- **Exponential Backoff**: Delay increases with each retry
- **Retryable Errors**: 5xx errors, timeouts, connection issues
- **Non-Retryable Errors**: 4xx errors (auth, bad request, quota exceeded)
- **Max Retries**: Configurable (default: 3)

Retryable errors:
- 500 Internal Server Error
- 502 Bad Gateway
- 503 Service Unavailable
- Connection timeouts
- Network errors

Non-retryable errors:
- 401 Unauthorized
- 403 Forbidden
- 400 Bad Request
- 429 Quota Exceeded

## Error Handling

The client provides detailed error messages:

```go
embeddings, err := client.Embed(ctx, texts)
if err != nil {
    // Handle different error types
    switch {
    case strings.Contains(err.Error(), "401"):
        // Authentication error
    case strings.Contains(err.Error(), "quota exceeded"):
        // Quota exceeded
    case strings.Contains(err.Error(), "timeout"):
        // Timeout error
    default:
        // Other errors
    }
}
```

## Performance Considerations

### Batch Size
- **Optimal**: 5 (Vertex AI maximum)
- **Memory**: Each embedding is 768 × 4 bytes = 3KB
- **Throughput**: Larger batches improve throughput

### Token Limits
- **Per Text**: 3072 tokens (configurable)
- **Estimation**: ~4 characters per token
- **Validation**: Automatic token count validation

### Network
- **Timeout**: 60 seconds per request
- **Retries**: Automatic retry on failures
- **Connection Pooling**: HTTP client connection reuse

## Authentication

### Development
```bash
export GOOGLE_ACCESS_TOKEN="your-access-token"
```

### Production
```bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account.json"
```

### Service Account Permissions
Required IAM roles:
- `aiplatform.endpoints.predict`

## Testing

Run tests with:
```bash
go test ./internal/llm
go test -v ./internal/llm  # Verbose output
```

Test coverage includes:
- Client creation and configuration
- Request/response structure validation
- Error handling scenarios
- Batching logic
- Retry behavior
- Text validation

## Examples

See `embeddings_example.go` for comprehensive examples:
- Basic embedding generation
- Advanced client configuration
- Batching demonstration
- Error handling patterns
- Request/response structures

## Security

- **Authentication**: OAuth 2.0 or service account
- **TLS**: All requests use HTTPS
- **Token Management**: Automatic token refresh
- **No Logging**: Sensitive data not logged

## Monitoring

The client includes structured logging:
- Request/response metrics
- Error tracking
- Performance monitoring
- Retry statistics

Use with your preferred logging framework for centralized monitoring.

## Limitations

- **Batch Size**: Maximum 5 texts per request (Vertex AI limit)
- **Token Limit**: 3072 tokens per text
- **Rate Limits**: Subject to Vertex AI quotas
- **Model**: Currently supports text-embedding-004 only

## Troubleshooting

### Common Issues

1. **Authentication Errors**
   - Verify project ID and region
   - Check access token or service account
   - Ensure proper IAM permissions

2. **Quota Exceeded**
   - Check Vertex AI quotas
   - Implement rate limiting
   - Use smaller batch sizes

3. **Timeout Errors**
   - Increase timeout settings
   - Check network connectivity
   - Verify text length limits

4. **Invalid Text**
   - Check for empty or whitespace-only texts
   - Verify token count limits
   - Ensure proper text encoding

