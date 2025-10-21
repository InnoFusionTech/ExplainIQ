package adk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// TestNewClient tests client creation
func TestNewClient(t *testing.T) {
	client := NewClient("https://api.example.com")

	if client == nil {
		t.Fatal("Expected client to be created, got nil")
	}

	if client.baseURL != "https://api.example.com" {
		t.Errorf("Expected baseURL 'https://api.example.com', got '%s'", client.baseURL)
	}

	if client.userAgent != "ExplainIQ-ADK-Client/1.0" {
		t.Errorf("Expected userAgent 'ExplainIQ-ADK-Client/1.0', got '%s'", client.userAgent)
	}

	if client.correlationID == "" {
		t.Error("Expected correlation ID to be set")
	}
}

// TestClientOptions tests client configuration options
func TestClientOptions(t *testing.T) {
	timeout := 60 * time.Second
	apiKey := "test-api-key"
	correlationID := "test-correlation-id"

	config := TaskConfig{
		Timeout:     45 * time.Second,
		MaxRetries:  5,
		RetryDelay:  2 * time.Second,
		BackoffType: "linear",
	}

	logger := logrus.New()

	client := NewClient("https://api.example.com",
		WithTimeout(timeout),
		WithConfig(config),
		WithAPIKey(apiKey),
		WithLogger(logger),
		WithCorrelationID(correlationID),
	)

	if client.httpClient.Timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, client.httpClient.Timeout)
	}

	if client.config.Timeout != config.Timeout {
		t.Errorf("Expected config timeout %v, got %v", config.Timeout, client.config.Timeout)
	}

	if client.apiKey != apiKey {
		t.Errorf("Expected API key '%s', got '%s'", apiKey, client.apiKey)
	}

	if client.correlationID != correlationID {
		t.Errorf("Expected correlation ID '%s', got '%s'", correlationID, client.correlationID)
	}
}

// TestDoTaskSuccess tests successful task execution
func TestDoTaskSuccess(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		if r.Header.Get("X-Idempotency-Key") == "" {
			t.Error("Expected X-Idempotency-Key header")
		}

		if r.Header.Get("X-Correlation-ID") == "" {
			t.Error("Expected X-Correlation-ID header")
		}

		// Parse request body
		var req TaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Verify request content
		if req.SessionID != "test-session" {
			t.Errorf("Expected SessionID 'test-session', got '%s'", req.SessionID)
		}

		// Create response
		response := TaskResponse{
			Delta: "Task completed successfully",
			Artifacts: map[string]string{
				"result.txt": "Task result content",
			},
			Next: "next-step",
			Metrics: map[string]interface{}{
				"duration_ms": 1500,
				"memory_mb":   128,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client := NewClient(server.URL)

	// Create task request
	req := TaskRequest{
		SessionID: "test-session",
		Step:      "process-data",
		Topic:     "data-processing",
		Inputs: map[string]string{
			"input_file": "data.csv",
			"format":     "csv",
		},
	}

	// Execute task
	ctx := context.Background()
	response, err := client.DoTask(ctx, server.URL+"/task", req)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify response
	if response.Delta != "Task completed successfully" {
		t.Errorf("Expected Delta 'Task completed successfully', got '%s'", response.Delta)
	}

	if len(response.Artifacts) != 1 {
		t.Errorf("Expected 1 artifact, got %d", len(response.Artifacts))
	}

	if response.Artifacts["result.txt"] != "Task result content" {
		t.Errorf("Expected artifact content 'Task result content', got '%s'", response.Artifacts["result.txt"])
	}

	if response.Next != "next-step" {
		t.Errorf("Expected Next 'next-step', got '%s'", response.Next)
	}

	if len(response.Metrics) != 2 {
		t.Errorf("Expected 2 metrics, got %d", len(response.Metrics))
	}
}

// TestDoTaskRetry tests retry logic with 5xx errors
func TestDoTaskRetry(t *testing.T) {
	attemptCount := 0

	// Create mock server that fails first two times, then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++

		if attemptCount <= 2 {
			// Return 5xx error for first two attempts
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{
					"code":    "INTERNAL_ERROR",
					"message": "Internal server error",
				},
			})
		} else {
			// Return success on third attempt
			response := TaskResponse{
				Delta: "Task completed after retry",
				Artifacts: map[string]string{
					"result.txt": "Retry success",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	// Create client with short retry delay for testing
	config := TaskConfig{
		Timeout:     5 * time.Second,
		MaxRetries:  3,
		RetryDelay:  100 * time.Millisecond,
		BackoffType: "fixed",
	}

	client := NewClient(server.URL, WithConfig(config))

	// Create task request
	req := TaskRequest{
		SessionID: "test-session",
		Step:      "retry-test",
		Topic:     "retry-testing",
		Inputs:    map[string]string{"test": "data"},
	}

	// Execute task
	ctx := context.Background()
	response, err := client.DoTask(ctx, server.URL+"/task", req)
	if err != nil {
		t.Fatalf("Expected no error after retries, got: %v", err)
	}

	// Verify response
	if response.Delta != "Task completed after retry" {
		t.Errorf("Expected Delta 'Task completed after retry', got '%s'", response.Delta)
	}

	// Verify that we made 3 attempts
	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

// TestDoTaskRateLimit tests retry logic with 429 errors
func TestDoTaskRateLimit(t *testing.T) {
	attemptCount := 0

	// Create mock server that returns 429 for first attempt, then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++

		if attemptCount == 1 {
			// Return 429 error for first attempt
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{
					"code":    "RATE_LIMITED",
					"message": "Rate limit exceeded",
				},
			})
		} else {
			// Return success on second attempt
			response := TaskResponse{
				Delta: "Task completed after rate limit retry",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	// Create client with short retry delay for testing
	config := TaskConfig{
		Timeout:     5 * time.Second,
		MaxRetries:  3,
		RetryDelay:  100 * time.Millisecond,
		BackoffType: "fixed",
	}

	client := NewClient(server.URL, WithConfig(config))

	// Create task request
	req := TaskRequest{
		SessionID: "test-session",
		Step:      "rate-limit-test",
		Topic:     "rate-limit-testing",
		Inputs:    map[string]string{"test": "data"},
	}

	// Execute task
	ctx := context.Background()
	response, err := client.DoTask(ctx, server.URL+"/task", req)
	if err != nil {
		t.Fatalf("Expected no error after rate limit retry, got: %v", err)
	}

	// Verify response
	if response.Delta != "Task completed after rate limit retry" {
		t.Errorf("Expected Delta 'Task completed after rate limit retry', got '%s'", response.Delta)
	}

	// Verify that we made 2 attempts
	if attemptCount != 2 {
		t.Errorf("Expected 2 attempts, got %d", attemptCount)
	}
}

// TestDoTaskNonRetryableError tests non-retryable error handling
func TestDoTaskNonRetryableError(t *testing.T) {
	// Create mock server that returns 400 error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"code":    "BAD_REQUEST",
				"message": "Invalid request",
				"details": "Missing required field",
			},
		})
	}))
	defer server.Close()

	// Create client
	client := NewClient(server.URL)

	// Create task request
	req := TaskRequest{
		SessionID: "test-session",
		Step:      "bad-request-test",
		Topic:     "error-testing",
		Inputs:    map[string]string{"test": "data"},
	}

	// Execute task
	ctx := context.Background()
	_, err := client.DoTask(ctx, server.URL+"/task", req)
	if err == nil {
		t.Fatal("Expected error for bad request, got nil")
	}

	// Verify error type - check for wrapped error
	var taskErr *TaskError
	if te, ok := err.(*TaskError); ok {
		taskErr = te
	} else if wrappedErr, ok := err.(interface{ Unwrap() error }); ok {
		if te, ok := wrappedErr.Unwrap().(*TaskError); ok {
			taskErr = te
		}
	}

	if taskErr == nil {
		t.Errorf("Expected TaskError, got %T: %v", err, err)
		return
	}

	if taskErr.Code != "BAD_REQUEST" {
		t.Errorf("Expected error code 'BAD_REQUEST', got '%s'", taskErr.Code)
	}
	if taskErr.IsRetryable() {
		t.Error("Expected error to be non-retryable")
	}
}

// TestDoTaskValidation tests request validation
func TestDoTaskValidation(t *testing.T) {
	client := NewClient("https://api.example.com")

	// Test empty SessionID
	req := TaskRequest{
		SessionID: "",
		Step:      "test-step",
		Topic:     "test-topic",
		Inputs:    map[string]string{"test": "data"},
	}

	ctx := context.Background()
	_, err := client.DoTask(ctx, "https://api.example.com/task", req)
	if err == nil {
		t.Fatal("Expected validation error for empty SessionID, got nil")
	}

	if !strings.Contains(err.Error(), "SessionID is required") {
		t.Errorf("Expected validation error about SessionID, got: %v", err)
	}

	// Test empty Step
	req.SessionID = "test-session"
	req.Step = ""

	_, err = client.DoTask(ctx, "https://api.example.com/task", req)
	if err == nil {
		t.Fatal("Expected validation error for empty Step, got nil")
	}

	if !strings.Contains(err.Error(), "Step is required") {
		t.Errorf("Expected validation error about Step, got: %v", err)
	}

	// Test empty Topic
	req.Step = "test-step"
	req.Topic = ""

	_, err = client.DoTask(ctx, "https://api.example.com/task", req)
	if err == nil {
		t.Fatal("Expected validation error for empty Topic, got nil")
	}

	if !strings.Contains(err.Error(), "Topic is required") {
		t.Errorf("Expected validation error about Topic, got: %v", err)
	}
}

// TestIdempotencyKeyGeneration tests idempotency key generation
func TestIdempotencyKeyGeneration(t *testing.T) {
	client := NewClient("https://api.example.com")

	req := TaskRequest{
		SessionID: "test-session",
		Step:      "test-step",
		Topic:     "test-topic",
		Inputs:    map[string]string{"test": "data"},
	}

	// Generate two keys for the same request
	key1 := client.generateIdempotencyKey(req)
	key2 := client.generateIdempotencyKey(req)

	// Keys should be different due to randomness
	if key1 == key2 {
		t.Error("Expected different idempotency keys, got same")
	}

	// Keys should contain request information
	if !strings.Contains(key1, "test-session") {
		t.Error("Expected idempotency key to contain session ID")
	}

	if !strings.Contains(key1, "test-step") {
		t.Error("Expected idempotency key to contain step")
	}

	if !strings.Contains(key1, "test-topic") {
		t.Error("Expected idempotency key to contain topic")
	}
}

// TestBackoffDelayCalculation tests backoff delay calculation
func TestBackoffDelayCalculation(t *testing.T) {
	client := NewClient("https://api.example.com")

	// Test exponential backoff
	config := TaskConfig{
		RetryDelay:  1 * time.Second,
		BackoffType: "exponential",
	}
	client.SetConfig(config)

	delay1 := client.calculateBackoffDelay(1)
	delay2 := client.calculateBackoffDelay(2)
	delay3 := client.calculateBackoffDelay(3)

	if delay1 != 1*time.Second {
		t.Errorf("Expected delay1 1s, got %v", delay1)
	}

	if delay2 != 2*time.Second {
		t.Errorf("Expected delay2 2s, got %v", delay2)
	}

	if delay3 != 4*time.Second {
		t.Errorf("Expected delay3 4s, got %v", delay3)
	}

	// Test linear backoff
	config.BackoffType = "linear"
	client.SetConfig(config)

	delay1 = client.calculateBackoffDelay(1)
	delay2 = client.calculateBackoffDelay(2)
	delay3 = client.calculateBackoffDelay(3)

	if delay1 != 1*time.Second {
		t.Errorf("Expected delay1 1s, got %v", delay1)
	}

	if delay2 != 2*time.Second {
		t.Errorf("Expected delay2 2s, got %v", delay2)
	}

	if delay3 != 3*time.Second {
		t.Errorf("Expected delay3 3s, got %v", delay3)
	}

	// Test fixed backoff
	config.BackoffType = "fixed"
	client.SetConfig(config)

	delay1 = client.calculateBackoffDelay(1)
	delay2 = client.calculateBackoffDelay(2)
	delay3 = client.calculateBackoffDelay(3)

	if delay1 != 1*time.Second {
		t.Errorf("Expected delay1 1s, got %v", delay1)
	}

	if delay2 != 1*time.Second {
		t.Errorf("Expected delay2 1s, got %v", delay2)
	}

	if delay3 != 1*time.Second {
		t.Errorf("Expected delay3 1s, got %v", delay3)
	}
}

// TestTaskRequestValidation tests TaskRequest validation
func TestTaskRequestValidation(t *testing.T) {
	// Test valid request
	req := TaskRequest{
		SessionID: "test-session",
		Step:      "test-step",
		Topic:     "test-topic",
		Inputs:    map[string]string{"test": "data"},
	}

	if err := req.Validate(); err != nil {
		t.Errorf("Expected valid request, got error: %v", err)
	}

	// Test invalid requests
	invalidRequests := []TaskRequest{
		{SessionID: "", Step: "test-step", Topic: "test-topic"},
		{SessionID: "test-session", Step: "", Topic: "test-topic"},
		{SessionID: "test-session", Step: "test-step", Topic: ""},
	}

	for i, req := range invalidRequests {
		if err := req.Validate(); err == nil {
			t.Errorf("Expected validation error for request %d, got nil", i)
		}
	}
}

// TestTaskResponseMethods tests TaskResponse methods
func TestTaskResponseMethods(t *testing.T) {
	// Test empty response
	response := TaskResponse{}
	if !response.IsEmpty() {
		t.Error("Expected empty response to be empty")
	}

	// Test non-empty response
	response.Delta = "test delta"
	if response.IsEmpty() {
		t.Error("Expected non-empty response to not be empty")
	}

	// Test AddArtifact
	response.AddArtifact("test.txt", "content")
	if response.Artifacts["test.txt"] != "content" {
		t.Error("Expected artifact to be added")
	}

	// Test AddMetric
	response.AddMetric("duration", 1000)
	if response.Metrics["duration"] != 1000 {
		t.Error("Expected metric to be added")
	}
}

// TestTaskErrorMethods tests TaskError methods
func TestTaskErrorMethods(t *testing.T) {
	// Test retryable error
	retryableError := &TaskError{
		Code:    "RATE_LIMITED",
		Message: "Rate limit exceeded",
		Details: "Try again later",
	}

	if !retryableError.IsRetryable() {
		t.Error("Expected rate limit error to be retryable")
	}

	// Test non-retryable error
	nonRetryableError := &TaskError{
		Code:    "BAD_REQUEST",
		Message: "Invalid request",
		Details: "Missing required field",
	}

	if nonRetryableError.IsRetryable() {
		t.Error("Expected bad request error to be non-retryable")
	}

	// Test error string formatting
	expectedError := "BAD_REQUEST: Invalid request (Missing required field)"
	if nonRetryableError.Error() != expectedError {
		t.Errorf("Expected error string '%s', got '%s'", expectedError, nonRetryableError.Error())
	}
}

// TestTaskMetadataMethods tests TaskMetadata methods
func TestTaskMetadataMethods(t *testing.T) {
	metadata := &TaskMetadata{
		MaxRetries: 3,
		RetryCount: 0,
		Status:     TaskStatusPending,
	}

	// Test status update
	metadata.UpdateStatus(TaskStatusRunning)
	if metadata.Status != TaskStatusRunning {
		t.Error("Expected status to be updated to running")
	}

	if metadata.StartedAt == nil {
		t.Error("Expected StartedAt to be set")
	}

	// Test retry increment
	metadata.IncrementRetry()
	if metadata.RetryCount != 1 {
		t.Error("Expected retry count to be incremented")
	}

	if metadata.Status != TaskStatusRetrying {
		t.Error("Expected status to be retrying")
	}

	// Test can retry
	if !metadata.CanRetry() {
		t.Error("Expected to be able to retry")
	}

	// Test max retries reached
	metadata.RetryCount = 3
	if metadata.CanRetry() {
		t.Error("Expected not to be able to retry when max retries reached")
	}

	// Test error setting
	err := &TaskError{Code: "TEST_ERROR", Message: "Test error"}
	metadata.SetError(err)
	if metadata.Error != err {
		t.Error("Expected error to be set")
	}

	if metadata.Status != TaskStatusFailed {
		t.Error("Expected status to be failed")
	}
}

// TestHealthCheck tests health check functionality
func TestHealthCheck(t *testing.T) {
	// Create mock server for health check
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("Expected path /health, got %s", r.URL.Path)
		}

		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}))
	defer server.Close()

	// Create client
	client := NewClient(server.URL)

	// Test health check
	ctx := context.Background()
	err := client.Health(ctx)
	if err != nil {
		t.Errorf("Expected no error from health check, got: %v", err)
	}
}

// Benchmark tests
func BenchmarkDoTask(b *testing.B) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := TaskResponse{
			Delta: "Benchmark response",
			Artifacts: map[string]string{
				"result.txt": "Benchmark result",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client := NewClient(server.URL)

	// Create task request
	req := TaskRequest{
		SessionID: "benchmark-session",
		Step:      "benchmark-step",
		Topic:     "benchmark-topic",
		Inputs:    map[string]string{"test": "data"},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.DoTask(ctx, server.URL+"/task", req)
		if err != nil {
			b.Fatalf("DoTask failed: %v", err)
		}
	}
}

func BenchmarkIdempotencyKeyGeneration(b *testing.B) {
	client := NewClient("https://api.example.com")

	req := TaskRequest{
		SessionID: "benchmark-session",
		Step:      "benchmark-step",
		Topic:     "benchmark-topic",
		Inputs:    map[string]string{"test": "data"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.generateIdempotencyKey(req)
	}
}
