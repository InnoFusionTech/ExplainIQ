# ADK (Agent Development Kit) Package

The ADK package provides a comprehensive client for executing tasks with robust error handling, retry logic, idempotency, and request tracing. It's designed for building reliable agent-based systems that can handle distributed task execution.

## Features

- **Task Execution**: Execute tasks with structured request/response patterns
- **Idempotency**: Automatic idempotency key generation and handling
- **Retry Logic**: Configurable retry with exponential backoff for transient failures
- **Request Tracing**: Full request correlation and tracing support
- **Error Handling**: Comprehensive error classification and handling
- **Metrics**: Built-in performance and execution metrics
- **Health Checks**: Service health monitoring capabilities

## Quick Start

### Basic Usage

```go
import "github.com/creduntvitam/explainiq/internal/adk"

// Create client
client := adk.NewClient("https://api.example.com")

// Create task request
req := adk.TaskRequest{
    SessionID: "session-123",
    Step:      "process-data",
    Topic:     "data-processing",
    Inputs: map[string]string{
        "input_file": "data.csv",
        "format":     "csv",
    },
}

// Execute task
ctx := context.Background()
response, err := client.DoTask(ctx, "https://api.example.com/task", req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Task completed: %s\n", response.Delta)
fmt.Printf("Next step: %s\n", response.Next)
```

### Advanced Configuration

```go
// Create client with custom configuration
config := adk.TaskConfig{
    Timeout:     60 * time.Second,
    MaxRetries:  5,
    RetryDelay:  2 * time.Second,
    BackoffType: "exponential",
}

client := adk.NewClient("https://api.example.com",
    adk.WithConfig(config),
    adk.WithAPIKey("your-api-key"),
    adk.WithTimeout(30 * time.Second),
    adk.WithCorrelationID("custom-correlation-id"),
)
```

## API Reference

### TaskRequest

```go
type TaskRequest struct {
    SessionID string            `json:"session_id"` // Unique session identifier
    Step      string            `json:"step"`       // Current step in the workflow
    Topic     string            `json:"topic"`      // Topic or category of the task
    Inputs    map[string]string `json:"inputs"`     // Input parameters for the task
}
```

### TaskResponse

```go
type TaskResponse struct {
    Delta     string                 `json:"delta"`     // Incremental output or changes
    Artifacts map[string]string      `json:"artifacts"` // Generated artifacts (files, data, etc.)
    Next      string                 `json:"next"`      // Next step or action to take
    Metrics   map[string]interface{} `json:"metrics"`   // Performance and execution metrics
}
```

### Client Methods

#### NewClient

```go
func NewClient(baseURL string, options ...ClientOption) *Client
```

Creates a new ADK client with optional configuration.

#### DoTask

```go
func (c *Client) DoTask(ctx context.Context, url string, req TaskRequest) (TaskResponse, error)
```

Executes a task with full retry logic, idempotency, and tracing.

#### Health

```go
func (c *Client) Health(ctx context.Context) error
```

Performs a health check on the service.

## Configuration

### TaskConfig

```go
type TaskConfig struct {
    Timeout     time.Duration `json:"timeout"`      // Task timeout
    MaxRetries  int           `json:"max_retries"`  // Maximum retries
    RetryDelay  time.Duration `json:"retry_delay"`  // Initial retry delay
    BackoffType string        `json:"backoff_type"` // Backoff strategy
}
```

### Client Options

- `WithTimeout(timeout)`: Set HTTP client timeout
- `WithConfig(config)`: Set task configuration
- `WithAPIKey(apiKey)`: Set API key for authentication
- `WithLogger(logger)`: Set custom logger
- `WithCorrelationID(id)`: Set custom correlation ID

## Retry Logic

The client implements sophisticated retry logic with configurable backoff strategies:

### Retryable Errors

- **5xx HTTP Errors**: 500, 502, 503, 504
- **Rate Limiting**: 429 Too Many Requests
- **Network Errors**: Connection timeouts, DNS failures
- **Temporary Failures**: Service unavailable, gateway errors

### Non-Retryable Errors

- **4xx Client Errors**: 400, 401, 403, 404
- **Validation Errors**: Invalid request format
- **Authentication Errors**: Invalid credentials

### Backoff Strategies

- **Exponential**: `delay * 2^attempt` (default)
- **Linear**: `delay * attempt`
- **Fixed**: Always the same delay

## Idempotency

The client automatically generates idempotency keys based on:

- Session ID
- Step name
- Topic
- Correlation ID
- Random component for uniqueness

This ensures that duplicate requests are handled safely without side effects.

## Request Tracing

Every request includes comprehensive tracing headers:

- `X-Correlation-ID`: Request correlation ID
- `X-Idempotency-Key`: Idempotency key
- `X-Task-ID`: Unique task identifier
- `X-Session-ID`: Session identifier
- `X-Step`: Current step
- `X-Topic`: Task topic
- `X-Request-ID`: Request identifier
- `X-Started-At`: Request start time
- `X-Retry-Count`: Current retry count
- `X-Max-Retries`: Maximum retries allowed

## Error Handling

### TaskError

```go
type TaskError struct {
    Code    string `json:"code"`    // Error code
    Message string `json:"message"` // Error message
    Details string `json:"details"` // Additional error details
}
```

### Error Classification

Errors are automatically classified as retryable or non-retryable:

```go
if taskErr.IsRetryable() {
    // Will be retried automatically
} else {
    // Will fail immediately
}
```

## Metrics and Monitoring

The client automatically collects metrics:

- Request duration
- HTTP status codes
- Response size
- Retry counts
- Error rates
- Artifact counts

## Examples

### Basic Task Execution

```go
client := adk.NewClient("https://api.example.com")

req := adk.TaskRequest{
    SessionID: "user-session-123",
    Step:      "analyze-document",
    Topic:     "document-analysis",
    Inputs: map[string]string{
        "document_id": "doc-456",
        "analysis_type": "sentiment",
    },
}

ctx := context.Background()
response, err := client.DoTask(ctx, "https://api.example.com/analyze", req)
if err != nil {
    log.Printf("Task failed: %v", err)
    return
}

fmt.Printf("Analysis complete: %s\n", response.Delta)
for key, artifact := range response.Artifacts {
    fmt.Printf("Generated artifact %s: %s\n", key, artifact)
}
```

### Error Handling

```go
response, err := client.DoTask(ctx, url, req)
if err != nil {
    // Check if it's a TaskError
    if taskErr, ok := err.(*adk.TaskError); ok {
        switch taskErr.Code {
        case "RATE_LIMITED":
            log.Printf("Rate limited, will retry automatically")
        case "BAD_REQUEST":
            log.Printf("Invalid request: %s", taskErr.Message)
        case "UNAUTHORIZED":
            log.Printf("Authentication failed")
        default:
            log.Printf("Unknown error: %s", taskErr.Message)
        }
    } else {
        log.Printf("Non-task error: %v", err)
    }
    return
}
```

### Custom Configuration

```go
// Create custom configuration
config := adk.TaskConfig{
    Timeout:     120 * time.Second,
    MaxRetries:  10,
    RetryDelay:  5 * time.Second,
    BackoffType: "exponential",
}

// Create client with custom logger
logger := logrus.New()
logger.SetLevel(logrus.DebugLevel)

client := adk.NewClient("https://api.example.com",
    adk.WithConfig(config),
    adk.WithLogger(logger),
    adk.WithAPIKey("your-api-key"),
)

// Execute task with custom timeout
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

response, err := client.DoTask(ctx, url, req)
```

### Health Monitoring

```go
// Check service health
err := client.Health(ctx)
if err != nil {
    log.Printf("Service unhealthy: %v", err)
    // Implement fallback logic
} else {
    log.Println("Service is healthy")
}
```

## Best Practices

### 1. Use Appropriate Timeouts

```go
// Set reasonable timeouts based on task complexity
config := adk.TaskConfig{
    Timeout: 30 * time.Second, // For quick tasks
    // Timeout: 300 * time.Second, // For long-running tasks
}
```

### 2. Handle Errors Gracefully

```go
response, err := client.DoTask(ctx, url, req)
if err != nil {
    // Log error with context
    log.WithFields(logrus.Fields{
        "session_id": req.SessionID,
        "step":       req.Step,
        "error":      err.Error(),
    }).Error("Task execution failed")
    
    // Implement appropriate fallback
    return handleTaskFailure(err)
}
```

### 3. Use Correlation IDs

```go
// Use consistent correlation IDs across related requests
correlationID := uuid.New().String()
client := adk.NewClient(baseURL, adk.WithCorrelationID(correlationID))
```

### 4. Monitor Metrics

```go
// Access metrics from response
for key, value := range response.Metrics {
    log.Printf("Metric %s: %v", key, value)
}
```

### 5. Implement Circuit Breakers

```go
// For high-volume scenarios, implement circuit breakers
// to prevent cascading failures
```

## Testing

The package includes comprehensive tests:

```bash
go test ./internal/adk
go test -v ./internal/adk  # Verbose output
go test -race ./internal/adk  # Race condition detection
```

## Performance Considerations

- **Connection Pooling**: HTTP client reuses connections
- **Request Batching**: Group related requests when possible
- **Timeout Management**: Set appropriate timeouts to prevent hanging
- **Retry Limits**: Balance between reliability and performance
- **Metrics Collection**: Monitor performance and adjust configuration

## Security

- **API Key Authentication**: Secure API key handling
- **Request Validation**: Input validation and sanitization
- **Error Information**: Avoid exposing sensitive information in errors
- **TLS**: All requests use HTTPS by default

## Troubleshooting

### Common Issues

1. **Timeout Errors**
   - Increase timeout configuration
   - Check network connectivity
   - Verify service performance

2. **Rate Limiting**
   - Implement exponential backoff
   - Use appropriate retry delays
   - Consider request throttling

3. **Authentication Failures**
   - Verify API key validity
   - Check authentication headers
   - Ensure proper permissions

4. **Retry Loops**
   - Check error classification
   - Verify retry configuration
   - Monitor retry patterns

### Debugging

Enable debug logging to see detailed request/response information:

```go
logger := logrus.New()
logger.SetLevel(logrus.DebugLevel)
client := adk.NewClient(baseURL, adk.WithLogger(logger))
```




