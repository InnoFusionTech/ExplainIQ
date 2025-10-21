package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// EmbeddingClient represents a client for generating embeddings using Vertex AI
type EmbeddingClient struct {
	projectID    string
	location     string
	model        string
	httpClient   *http.Client
	logger       *logrus.Logger
	maxRetries   int
	retryDelay   time.Duration
	batchSize    int
	maxTokens    int
}

// EmbeddingRequest represents a request to the Vertex AI embeddings API
type EmbeddingRequest struct {
	Instances []EmbeddingInstance `json:"instances"`
	Parameters EmbeddingParameters `json:"parameters"`
}

// EmbeddingInstance represents a single text instance for embedding
type EmbeddingInstance struct {
	Content string `json:"content"`
	TaskType string `json:"task_type,omitempty"`
	Title   string `json:"title,omitempty"`
}

// EmbeddingParameters represents parameters for the embedding request
type EmbeddingParameters struct {
	OutputDimensionality int `json:"outputDimensionality,omitempty"`
}

// EmbeddingResponse represents the response from the Vertex AI embeddings API
type EmbeddingResponse struct {
	Predictions []EmbeddingPrediction `json:"predictions"`
	Metadata    EmbeddingMetadata     `json:"metadata"`
}

// EmbeddingPrediction represents a single embedding prediction
type EmbeddingPrediction struct {
	Embeddings EmbeddingValue `json:"embeddings"`
	Stats      EmbeddingStats `json:"stats,omitempty"`
}

// EmbeddingValue represents the actual embedding values
type EmbeddingValue struct {
	Values []float32 `json:"values"`
}

// EmbeddingStats represents statistics about the embedding
type EmbeddingStats struct {
	TokenCount int `json:"token_count"`
}

// EmbeddingMetadata represents metadata about the response
type EmbeddingMetadata struct {
	Model        string `json:"model"`
	ModelVersion string `json:"model_version,omitempty"`
}

// EmbeddingError represents an error from the Vertex AI API
type EmbeddingError struct {
	Error EmbeddingErrorDetail `json:"error"`
}

// EmbeddingErrorDetail represents error details
type EmbeddingErrorDetail struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// NewEmbeddingClient creates a new embedding client
func NewEmbeddingClient(projectID, location string) *EmbeddingClient {
	return &EmbeddingClient{
		projectID:  projectID,
		location:   location,
		model:      "text-embedding-004",
		httpClient: &http.Client{Timeout: 60 * time.Second},
		logger:     logrus.New(),
		maxRetries: 3,
		retryDelay: 1 * time.Second,
		batchSize:  5, // Vertex AI text-embedding-004 supports up to 5 texts per request
		maxTokens:  3072, // Maximum tokens per text
	}
}

// Embed generates embeddings for the given texts using Vertex AI text-embedding-004
func (c *EmbeddingClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided")
	}

	c.logger.WithFields(logrus.Fields{
		"text_count": len(texts),
		"model":      c.model,
	}).Info("Generating embeddings")

	// Validate texts
	if err := c.validateTexts(texts); err != nil {
		return nil, fmt.Errorf("text validation failed: %w", err)
	}

	// Process in batches
	allEmbeddings := make([][]float32, 0, len(texts))
	
	for i := 0; i < len(texts); i += c.batchSize {
		end := i + c.batchSize
		if end > len(texts) {
			end = len(texts)
		}
		
		batch := texts[i:end]
		c.logger.WithFields(logrus.Fields{
			"batch_start": i,
			"batch_end":   end,
			"batch_size":  len(batch),
		}).Debug("Processing embedding batch")

		batchEmbeddings, err := c.embedBatch(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("batch embedding failed (batch %d-%d): %w", i, end-1, err)
		}

		allEmbeddings = append(allEmbeddings, batchEmbeddings...)
	}

	c.logger.WithField("total_embeddings", len(allEmbeddings)).Info("Embeddings generated successfully")
	return allEmbeddings, nil
}

// embedBatch processes a single batch of texts
func (c *EmbeddingClient) embedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	// Create request
	instances := make([]EmbeddingInstance, len(texts))
	for i, text := range texts {
		instances[i] = EmbeddingInstance{
			Content:  text,
			TaskType: "RETRIEVAL_DOCUMENT", // Default task type for document embedding
		}
	}

	request := EmbeddingRequest{
		Instances: instances,
		Parameters: EmbeddingParameters{
			OutputDimensionality: 768, // text-embedding-004 default output dimension
		},
	}

	// Execute request with retry logic
	var response EmbeddingResponse
	var err error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			delay := c.retryDelay * time.Duration(math.Pow(2, float64(attempt-1)))
			c.logger.WithFields(logrus.Fields{
				"attempt": attempt,
				"delay":   delay,
			}).Warn("Retrying embedding request")
			
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		response, err = c.executeRequest(ctx, request)
		if err == nil {
			break
		}

		c.logger.WithFields(logrus.Fields{
			"attempt": attempt + 1,
			"error":   err.Error(),
		}).Warn("Embedding request failed")

		// Don't retry on certain errors
		if c.isNonRetryableError(err) {
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("embedding request failed after %d attempts: %w", c.maxRetries+1, err)
	}

	// Extract embeddings from response
	embeddings := make([][]float32, len(response.Predictions))
	for i, prediction := range response.Predictions {
		embeddings[i] = prediction.Embeddings.Values
	}

	return embeddings, nil
}

// executeRequest executes the HTTP request to Vertex AI
func (c *EmbeddingClient) executeRequest(ctx context.Context, request EmbeddingRequest) (EmbeddingResponse, error) {
	// Get access token
	accessToken, err := c.getAccessToken(ctx)
	if err != nil {
		return EmbeddingResponse{}, fmt.Errorf("failed to get access token: %w", err)
	}

	// Marshal request
	requestBody, err := json.Marshal(request)
	if err != nil {
		return EmbeddingResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:predict",
		c.location, c.projectID, c.location, c.model)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return EmbeddingResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return EmbeddingResponse{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return EmbeddingResponse{}, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var apiError EmbeddingError
		if err := json.Unmarshal(responseBody, &apiError); err == nil {
			return EmbeddingResponse{}, fmt.Errorf("API error %d: %s", apiError.Error.Code, apiError.Error.Message)
		}
		return EmbeddingResponse{}, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Parse response
	var response EmbeddingResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return EmbeddingResponse{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Validate response
	if len(response.Predictions) != len(request.Instances) {
		return EmbeddingResponse{}, fmt.Errorf("response predictions count (%d) doesn't match request instances count (%d)",
			len(response.Predictions), len(request.Instances))
	}

	return response, nil
}

// getAccessToken retrieves an access token for authentication
func (c *EmbeddingClient) getAccessToken(ctx context.Context) (string, error) {
	// Try to get token from environment first (for local development)
	if token := os.Getenv("GOOGLE_ACCESS_TOKEN"); token != "" {
		return token, nil
	}

	// For production, use Application Default Credentials
	// This requires the Google Cloud SDK or service account key
	// In a real implementation, you would use the Google Cloud Go client libraries
	// to get the access token. For this example, we'll return an error if no token is found.
	return "", fmt.Errorf("no access token found. Set GOOGLE_ACCESS_TOKEN environment variable or configure Application Default Credentials")
}

// validateTexts validates the input texts
func (c *EmbeddingClient) validateTexts(texts []string) error {
	for i, text := range texts {
		if strings.TrimSpace(text) == "" {
			return fmt.Errorf("text at index %d is empty", i)
		}
		
		// Estimate token count (rough approximation: 1 token ≈ 4 characters)
		estimatedTokens := len(text) / 4
		if estimatedTokens > c.maxTokens {
			return fmt.Errorf("text at index %d exceeds maximum token limit (%d tokens estimated, max %d)",
				i, estimatedTokens, c.maxTokens)
		}
	}
	return nil
}

// isNonRetryableError checks if an error should not be retried
func (c *EmbeddingClient) isNonRetryableError(err error) bool {
	errStr := err.Error()
	
	// Don't retry on authentication errors
	if strings.Contains(errStr, "401") || strings.Contains(errStr, "unauthorized") {
		return true
	}
	
	// Don't retry on permission errors
	if strings.Contains(errStr, "403") || strings.Contains(errStr, "forbidden") {
		return true
	}
	
	// Don't retry on bad request errors
	if strings.Contains(errStr, "400") || strings.Contains(errStr, "bad request") {
		return true
	}
	
	// Don't retry on quota exceeded (429)
	if strings.Contains(errStr, "429") || strings.Contains(errStr, "quota exceeded") {
		return true
	}
	
	return false
}

// SetBatchSize sets the batch size for embedding requests
func (c *EmbeddingClient) SetBatchSize(size int) {
	if size > 0 && size <= 5 { // Vertex AI text-embedding-004 limit
		c.batchSize = size
	}
}

// SetMaxRetries sets the maximum number of retries
func (c *EmbeddingClient) SetMaxRetries(retries int) {
	if retries >= 0 {
		c.maxRetries = retries
	}
}

// SetRetryDelay sets the initial retry delay
func (c *EmbeddingClient) SetRetryDelay(delay time.Duration) {
	if delay > 0 {
		c.retryDelay = delay
	}
}

// SetMaxTokens sets the maximum tokens per text
func (c *EmbeddingClient) SetMaxTokens(tokens int) {
	if tokens > 0 {
		c.maxTokens = tokens
	}
}

// GetModelInfo returns information about the embedding model
func (c *EmbeddingClient) GetModelInfo() map[string]interface{} {
	return map[string]interface{}{
		"model":               c.model,
		"project_id":          c.projectID,
		"location":            c.location,
		"max_batch_size":      c.batchSize,
		"max_tokens_per_text": c.maxTokens,
		"output_dimensions":   768,
		"max_retries":         c.maxRetries,
		"retry_delay":         c.retryDelay,
	}
}

// Embed is a convenience function that creates a client and generates embeddings
func Embed(ctx context.Context, texts []string) ([][]float32, error) {
	// Get configuration from environment
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		projectID = os.Getenv("EXPLAINIQ_PROJECT_ID")
	}
	if projectID == "" {
		return nil, fmt.Errorf("GOOGLE_CLOUD_PROJECT or EXPLAINIQ_PROJECT_ID environment variable must be set")
	}

	location := os.Getenv("GOOGLE_CLOUD_LOCATION")
	if location == "" {
		location = os.Getenv("EXPLAINIQ_REGION")
	}
	if location == "" {
		location = "us-central1" // Default location
	}

	client := NewEmbeddingClient(projectID, location)
	return client.Embed(ctx, texts)
}
