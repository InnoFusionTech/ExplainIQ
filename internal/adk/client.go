package adk

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Client represents an ADK client for task execution
type Client struct {
	httpClient    *http.Client
	logger        *logrus.Logger
	config        TaskConfig
	baseURL       string
	userAgent     string
	apiKey        string
	correlationID string
}

// NewClient creates a new ADK client
func NewClient(baseURL string, options ...ClientOption) *Client {
	client := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:        logrus.New(),
		config:        DefaultTaskConfig(),
		baseURL:       baseURL,
		userAgent:     "ExplainIQ-ADK-Client/1.0",
		correlationID: uuid.New().String(),
	}

	// Apply options
	for _, option := range options {
		option(client)
	}

	return client
}

// ClientOption represents a client configuration option
type ClientOption func(*Client)

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithConfig sets the task configuration
func WithConfig(config TaskConfig) ClientOption {
	return func(c *Client) {
		c.config = config
	}
}

// WithAPIKey sets the API key for authentication
func WithAPIKey(apiKey string) ClientOption {
	return func(c *Client) {
		c.apiKey = apiKey
	}
}

// WithLogger sets a custom logger
func WithLogger(logger *logrus.Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithCorrelationID sets a custom correlation ID
func WithCorrelationID(correlationID string) ClientOption {
	return func(c *Client) {
		c.correlationID = correlationID
	}
}

// WithAuthToken sets the authentication token for requests
func WithAuthToken(token string) ClientOption {
	return func(c *Client) {
		c.apiKey = token // Reuse apiKey field for auth token
	}
}

// DoTask executes a task with idempotency, retry logic, and request tracing
func (c *Client) DoTask(ctx context.Context, url string, req TaskRequest) (TaskResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return TaskResponse{}, fmt.Errorf("invalid task request: %w", err)
	}

	// Generate idempotency key
	idempotencyKey := c.generateIdempotencyKey(req)

	// Create task metadata
	metadata := &TaskMetadata{
		TaskID:         uuid.New().String(),
		SessionID:      req.SessionID,
		Step:           req.Step,
		Topic:          req.Topic,
		Status:         TaskStatusPending,
		CreatedAt:      time.Now(),
		MaxRetries:     c.config.MaxRetries,
		Inputs:         req.Inputs,
		CorrelationID:  c.correlationID,
		IdempotencyKey: idempotencyKey,
	}

	c.logger.WithFields(logrus.Fields{
		"task_id":         metadata.TaskID,
		"session_id":      req.SessionID,
		"step":            req.Step,
		"topic":           req.Topic,
		"correlation_id":  c.correlationID,
		"idempotency_key": idempotencyKey,
	}).Info("Starting task execution")

	// Execute task with retry logic
	var response TaskResponse
	var err error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate backoff delay
			delay := c.calculateBackoffDelay(attempt)
			c.logger.WithFields(logrus.Fields{
				"task_id": metadata.TaskID,
				"attempt": attempt,
				"delay":   delay,
			}).Warn("Retrying task execution")

			// Wait for backoff delay
			select {
			case <-ctx.Done():
				return TaskResponse{}, ctx.Err()
			case <-time.After(delay):
			}
		}

		// Update metadata
		metadata.IncrementRetry()
		metadata.UpdateStatus(TaskStatusRunning)

		// Execute the task
		response, err = c.executeTask(ctx, url, req, idempotencyKey, metadata)
		if err == nil {
			// Success
			metadata.UpdateStatus(TaskStatusCompleted)
			metadata.Outputs = response.Artifacts
			metadata.Metrics = response.Metrics

			c.logger.WithFields(logrus.Fields{
				"task_id":  metadata.TaskID,
				"attempt":  attempt + 1,
				"duration": metadata.Duration,
			}).Info("Task completed successfully")

			return response, nil
		}

		// Check if error is retryable
		if taskErr, ok := err.(*TaskError); ok && taskErr.IsRetryable() {
			metadata.SetError(err)
			c.logger.WithFields(logrus.Fields{
				"task_id": metadata.TaskID,
				"attempt": attempt + 1,
				"error":   err.Error(),
			}).Warn("Task failed with retryable error")
			continue
		}

		// Check if it's a wrapped TaskError
		if wrappedErr, ok := err.(interface{ Unwrap() error }); ok {
			if taskErr, ok := wrappedErr.Unwrap().(*TaskError); ok && taskErr.IsRetryable() {
				metadata.SetError(err)
				c.logger.WithFields(logrus.Fields{
					"task_id": metadata.TaskID,
					"attempt": attempt + 1,
					"error":   err.Error(),
				}).Warn("Task failed with retryable error")
				continue
			}
		}

		// Non-retryable error
		metadata.SetError(err)
		c.logger.WithFields(logrus.Fields{
			"task_id": metadata.TaskID,
			"attempt": attempt + 1,
			"error":   err.Error(),
		}).Error("Task failed with non-retryable error")
		break
	}

	// All retries exhausted
	metadata.UpdateStatus(TaskStatusFailed)
	return TaskResponse{}, fmt.Errorf("task failed after %d attempts: %w", c.config.MaxRetries+1, err)
}

// executeTask executes a single task request
func (c *Client) executeTask(ctx context.Context, url string, req TaskRequest, idempotencyKey string, metadata *TaskMetadata) (TaskResponse, error) {
	// Create HTTP request
	requestBody, err := json.Marshal(req)
	if err != nil {
		return TaskResponse{}, &TaskError{
			Code:    "SERIALIZATION_ERROR",
			Message: "Failed to marshal request",
			Details: err.Error(),
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return TaskResponse{}, &TaskError{
			Code:    "REQUEST_ERROR",
			Message: "Failed to create HTTP request",
			Details: err.Error(),
		}
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", c.userAgent)
	httpReq.Header.Set("X-Correlation-ID", c.correlationID)
	httpReq.Header.Set("X-Idempotency-Key", idempotencyKey)
	httpReq.Header.Set("X-Task-ID", metadata.TaskID)
	httpReq.Header.Set("X-Session-ID", req.SessionID)
	httpReq.Header.Set("X-Step", req.Step)
	httpReq.Header.Set("X-Topic", req.Topic)

	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	// Add tracing headers
	c.addTracingHeaders(httpReq, metadata)

	// Execute request
	startTime := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return TaskResponse{}, &TaskError{
			Code:    "NETWORK_ERROR",
			Message: "Failed to execute HTTP request",
			Details: err.Error(),
		}
	}
	defer resp.Body.Close()

	// Record request duration
	duration := time.Since(startTime)
	metadata.AddMetric("request_duration_ms", duration.Milliseconds())
	metadata.AddMetric("http_status_code", resp.StatusCode)

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return TaskResponse{}, &TaskError{
			Code:    "RESPONSE_ERROR",
			Message: "Failed to read response body",
			Details: err.Error(),
		}
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		return TaskResponse{}, c.handleHTTPError(resp.StatusCode, responseBody, metadata)
	}

	// Parse response
	var taskResponse TaskResponse
	if err := json.Unmarshal(responseBody, &taskResponse); err != nil {
		return TaskResponse{}, &TaskError{
			Code:    "DESERIALIZATION_ERROR",
			Message: "Failed to unmarshal response",
			Details: err.Error(),
		}
	}

	// Add response metrics
	metadata.AddMetric("response_size_bytes", len(responseBody))
	metadata.AddMetric("artifacts_count", len(taskResponse.Artifacts))
	metadata.AddMetric("metrics_count", len(taskResponse.Metrics))

	return taskResponse, nil
}

// generateIdempotencyKey generates a unique idempotency key
func (c *Client) generateIdempotencyKey(req TaskRequest) string {
	// Create a deterministic key based on request content
	keyData := fmt.Sprintf("%s:%s:%s:%s", req.SessionID, req.Step, req.Topic, c.correlationID)

	// Add some randomness to ensure uniqueness
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)

	return fmt.Sprintf("%x-%s", randomBytes, keyData)
}

// calculateBackoffDelay calculates the backoff delay for retries
func (c *Client) calculateBackoffDelay(attempt int) time.Duration {
	switch c.config.BackoffType {
	case "exponential":
		// Exponential backoff: delay * 2^attempt
		delay := float64(c.config.RetryDelay) * math.Pow(2, float64(attempt-1))
		return time.Duration(delay)
	case "linear":
		// Linear backoff: delay * attempt
		return c.config.RetryDelay * time.Duration(attempt)
	case "fixed":
		// Fixed backoff: always the same delay
		return c.config.RetryDelay
	default:
		// Default to exponential
		delay := float64(c.config.RetryDelay) * math.Pow(2, float64(attempt-1))
		return time.Duration(delay)
	}
}

// handleHTTPError handles HTTP error responses
func (c *Client) handleHTTPError(statusCode int, responseBody []byte, metadata *TaskMetadata) *TaskError {
	// Try to parse error response
	var errorResponse struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Details string `json:"details"`
		} `json:"error"`
	}

	if err := json.Unmarshal(responseBody, &errorResponse); err == nil {
		return &TaskError{
			Code:    errorResponse.Error.Code,
			Message: errorResponse.Error.Message,
			Details: errorResponse.Error.Details,
		}
	}

	// Fallback to generic error based on status code
	errorCode := "HTTP_ERROR"
	message := fmt.Sprintf("HTTP %d", statusCode)

	switch statusCode {
	case 400:
		errorCode = "BAD_REQUEST"
		message = "Bad request"
	case 401:
		errorCode = "UNAUTHORIZED"
		message = "Unauthorized"
	case 403:
		errorCode = "FORBIDDEN"
		message = "Forbidden"
	case 404:
		errorCode = "NOT_FOUND"
		message = "Not found"
	case 429:
		errorCode = "RATE_LIMITED"
		message = "Rate limited"
	case 500:
		errorCode = "INTERNAL_ERROR"
		message = "Internal server error"
	case 502:
		errorCode = "BAD_GATEWAY"
		message = "Bad gateway"
	case 503:
		errorCode = "SERVICE_UNAVAILABLE"
		message = "Service unavailable"
	case 504:
		errorCode = "GATEWAY_TIMEOUT"
		message = "Gateway timeout"
	}

	// Add response body as details if available
	details := string(responseBody)
	if len(details) > 500 {
		details = details[:500] + "..."
	}

	return &TaskError{
		Code:    errorCode,
		Message: message,
		Details: details,
	}
}

// addTracingHeaders adds tracing headers to the request
func (c *Client) addTracingHeaders(req *http.Request, metadata *TaskMetadata) {
	// Standard tracing headers
	req.Header.Set("X-Request-ID", metadata.TaskID)
	req.Header.Set("X-Session-ID", metadata.SessionID)
	req.Header.Set("X-Step", metadata.Step)
	req.Header.Set("X-Topic", metadata.Topic)
	req.Header.Set("X-Created-At", metadata.CreatedAt.Format(time.RFC3339))

	// Add timing headers
	if metadata.StartedAt != nil {
		req.Header.Set("X-Started-At", metadata.StartedAt.Format(time.RFC3339))
	}

	// Add retry information
	req.Header.Set("X-Retry-Count", strconv.Itoa(metadata.RetryCount))
	req.Header.Set("X-Max-Retries", strconv.Itoa(metadata.MaxRetries))

	// Add custom tracing headers
	req.Header.Set("X-Client-Version", "1.0")
	req.Header.Set("X-Client-Name", "ExplainIQ-ADK")
}

// GetBaseURL returns the client's base URL
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// GetConfig returns the current client configuration
func (c *Client) GetConfig() TaskConfig {
	return c.config
}

// SetConfig updates the client configuration
func (c *Client) SetConfig(config TaskConfig) {
	c.config = config
}

// SetTimeout updates the HTTP client timeout
func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// SetAPIKey updates the API key
func (c *Client) SetAPIKey(apiKey string) {
	c.apiKey = apiKey
}

// GetCorrelationID returns the current correlation ID
func (c *Client) GetCorrelationID() string {
	return c.correlationID
}

// SetCorrelationID updates the correlation ID
func (c *Client) SetCorrelationID(correlationID string) {
	c.correlationID = correlationID
}

// Health checks the health of the ADK client
func (c *Client) Health(ctx context.Context) error {
	// Create a simple health check request
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("X-Correlation-ID", c.correlationID)

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	return nil
}
