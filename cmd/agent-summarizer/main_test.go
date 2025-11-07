package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/InnoFusionTech/ExplainIQ/internal/adk"
	"github.com/InnoFusionTech/ExplainIQ/internal/llm"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockGeminiClient is a mock implementation of the Gemini client
type MockGeminiClient struct {
	mock.Mock
}

func (m *MockGeminiClient) Summarize(ctx context.Context, topic, context string) (*llm.SummarizeResponse, error) {
	args := m.Called(ctx, topic, context)
	return args.Get(0).(*llm.SummarizeResponse), args.Error(1)
}

func (m *MockGeminiClient) ExplainWithOG(ctx context.Context, topic, outline, misconceptions, context string) (*llm.OGLesson, error) {
	args := m.Called(ctx, topic, outline, misconceptions, context)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llm.OGLesson), args.Error(1)
}

func (m *MockGeminiClient) CritiqueLesson(ctx context.Context, lessonJSON string) (*llm.CritiqueResponse, error) {
	args := m.Called(ctx, lessonJSON)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llm.CritiqueResponse), args.Error(1)
}

func (m *MockGeminiClient) VisualizeCore(ctx context.Context, lessonJSON, sessionID string) (*llm.VisualizeResponse, error) {
	args := m.Called(ctx, lessonJSON, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llm.VisualizeResponse), args.Error(1)
}

func (m *MockGeminiClient) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockGeminiClient) SetAPIKey(apiKey string) {
	m.Called(apiKey)
}

func (m *MockGeminiClient) SetModel(model string) {
	m.Called(model)
}

func (m *MockGeminiClient) SetBaseURL(baseURL string) {
	m.Called(baseURL)
}

func (m *MockGeminiClient) GetModelInfo() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

// TestNewSummarizerService tests service creation
func TestNewSummarizerService(t *testing.T) {
	service := NewSummarizerService()

	assert.NotNil(t, service)
	assert.NotNil(t, service.geminiClient)
	assert.NotNil(t, service.logger)
}

// TestProcessTaskSuccess tests successful task processing
func TestProcessTaskSuccess(t *testing.T) {
	// Create mock client
	mockClient := &MockGeminiClient{}

	// Create service with mock client
	service := &SummarizerService{
		geminiClient: mockClient,
		logger:       logrus.New(),
	}

	// Mock successful summarization
	expectedResult := &llm.SummarizeResponse{
		Outline:        []string{"Introduction to topic", "Key concepts", "Applications"},
		Prerequisites:  []string{"Basic knowledge", "Mathematical background"},
		Misconceptions: []string{"Common mistake 1", "Common mistake 2"},
		Citations:      []string{"doc1", "doc2"},
	}

	mockClient.On("Summarize", mock.Anything, "machine learning", "context about ML").Return(expectedResult, nil)

	// Create task request
	req := adk.TaskRequest{
		SessionID: "test-session",
		Step:      "summarizer",
		Topic:     "machine learning",
		Inputs: map[string]string{
			"topic":   "machine learning",
			"context": "context about ML",
		},
	}

	// Process task
	ctx := context.Background()
	response, err := service.ProcessTask(ctx, req)

	// Verify no error occurred
	assert.NoError(t, err)
	assert.NotNil(t, response)

	// Verify artifacts
	assert.Contains(t, response.Artifacts, "outline")
	assert.Contains(t, response.Artifacts, "prerequisites")
	assert.Contains(t, response.Artifacts, "misconceptions")
	assert.Contains(t, response.Artifacts, "citations")

	// Verify metrics
	assert.Equal(t, 3, response.Metrics["outline_count"])
	assert.Equal(t, 2, response.Metrics["prerequisites_count"])
	assert.Equal(t, 2, response.Metrics["misconceptions_count"])
	assert.Equal(t, 2, response.Metrics["citations_count"])

	// Verify JSON artifacts can be parsed
	var outline []string
	err = json.Unmarshal([]byte(response.Artifacts["outline"]), &outline)
	assert.NoError(t, err)
	assert.Equal(t, []string{"Introduction to topic", "Key concepts", "Applications"}, outline)

	mockClient.AssertExpectations(t)
}

// TestProcessTaskMissingTopic tests task processing with missing topic
func TestProcessTaskMissingTopic(t *testing.T) {
	// Create service
	service := &SummarizerService{
		geminiClient: &MockGeminiClient{},
		logger:       logrus.New(),
	}

	// Create task request without topic
	req := adk.TaskRequest{
		SessionID: "test-session",
		Step:      "summarizer",
		Topic:     "machine learning",
		Inputs: map[string]string{
			"context": "context about ML",
		},
	}

	// Process task
	ctx := context.Background()
	response, err := service.ProcessTask(ctx, req)

	// Verify error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "topic is required")
	assert.Empty(t, response.Artifacts)
}

// TestProcessTaskGeminiError tests task processing with Gemini API error
func TestProcessTaskGeminiError(t *testing.T) {
	// Create mock client
	mockClient := &MockGeminiClient{}

	// Create service with mock client
	service := &SummarizerService{
		geminiClient: mockClient,
		logger:       logrus.New(),
	}

	// Mock Gemini API error
	mockClient.On("Summarize", mock.Anything, "machine learning", "context about ML").Return((*llm.SummarizeResponse)(nil), fmt.Errorf("API error"))

	// Create task request
	req := adk.TaskRequest{
		SessionID: "test-session",
		Step:      "summarizer",
		Topic:     "machine learning",
		Inputs: map[string]string{
			"topic":   "machine learning",
			"context": "context about ML",
		},
	}

	// Process task
	ctx := context.Background()
	response, err := service.ProcessTask(ctx, req)

	// Verify error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "summarization failed")
	assert.Empty(t, response.Artifacts)

	mockClient.AssertExpectations(t)
}

// TestProcessTaskWithoutContext tests task processing without context
func TestProcessTaskWithoutContext(t *testing.T) {
	// Create mock client
	mockClient := &MockGeminiClient{}

	// Create service with mock client
	service := &SummarizerService{
		geminiClient: mockClient,
		logger:       logrus.New(),
	}

	// Mock successful summarization without context
	expectedResult := &llm.SummarizeResponse{
		Outline:        []string{"Introduction to topic"},
		Prerequisites:  []string{"Basic knowledge"},
		Misconceptions: []string{"Common mistake"},
		Citations:      []string{},
	}

	mockClient.On("Summarize", mock.Anything, "machine learning", "").Return(expectedResult, nil)

	// Create task request without context
	req := adk.TaskRequest{
		SessionID: "test-session",
		Step:      "summarizer",
		Topic:     "machine learning",
		Inputs: map[string]string{
			"topic": "machine learning",
		},
	}

	// Process task
	ctx := context.Background()
	response, err := service.ProcessTask(ctx, req)

	// Verify no error occurred
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Contains(t, response.Artifacts, "outline")

	mockClient.AssertExpectations(t)
}

// TestTaskEndpointSuccess tests the HTTP endpoint for successful requests
func TestTaskEndpointSuccess(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock client
	mockClient := &MockGeminiClient{}

	// Create service with mock client
	service := &SummarizerService{
		geminiClient: mockClient,
		logger:       logrus.New(),
	}

	// Mock successful summarization
	expectedResult := &llm.SummarizeResponse{
		Outline:        []string{"Introduction to topic"},
		Prerequisites:  []string{"Basic knowledge"},
		Misconceptions: []string{"Common mistake"},
		Citations:      []string{"doc1"},
	}

	mockClient.On("Summarize", mock.Anything, "machine learning", "context about ML").Return(expectedResult, nil)

	// Create router
	router := gin.New()
	router.POST("/task", func(c *gin.Context) {
		var req adk.TaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}

		if err := req.Validate(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		response, err := service.ProcessTask(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Task processing failed"})
			return
		}

		c.JSON(http.StatusOK, response)
	})

	// Create request
	reqBody := adk.TaskRequest{
		SessionID: "test-session",
		Step:      "summarizer",
		Topic:     "machine learning",
		Inputs: map[string]string{
			"topic":   "machine learning",
			"context": "context about ML",
		},
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/task", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response adk.TaskResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Artifacts, "outline")
	assert.Contains(t, response.Artifacts, "prerequisites")
	assert.Contains(t, response.Artifacts, "misconceptions")
	assert.Contains(t, response.Artifacts, "citations")

	mockClient.AssertExpectations(t)
}

// TestTaskEndpointInvalidRequest tests the HTTP endpoint with invalid request
func TestTaskEndpointInvalidRequest(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create service
	service := &SummarizerService{
		geminiClient: &MockGeminiClient{},
		logger:       logrus.New(),
	}

	// Create router
	router := gin.New()
	router.POST("/task", func(c *gin.Context) {
		var req adk.TaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}

		if err := req.Validate(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		response, err := service.ProcessTask(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Task processing failed"})
			return
		}

		c.JSON(http.StatusOK, response)
	})

	// Create invalid request (missing required fields)
	reqBody := map[string]interface{}{
		"session_id": "test-session",
		// Missing step and topic
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/task", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
}

// TestTaskEndpointProcessingError tests the HTTP endpoint with processing error
func TestTaskEndpointProcessingError(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock client
	mockClient := &MockGeminiClient{}

	// Create service with mock client
	service := &SummarizerService{
		geminiClient: mockClient,
		logger:       logrus.New(),
	}

	// Mock Gemini API error
	mockClient.On("Summarize", mock.Anything, "machine learning", "context about ML").Return((*llm.SummarizeResponse)(nil), fmt.Errorf("API error"))

	// Create router
	router := gin.New()
	router.POST("/task", func(c *gin.Context) {
		var req adk.TaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}

		if err := req.Validate(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		response, err := service.ProcessTask(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Task processing failed"})
			return
		}

		c.JSON(http.StatusOK, response)
	})

	// Create request
	reqBody := adk.TaskRequest{
		SessionID: "test-session",
		Step:      "summarizer",
		Topic:     "machine learning",
		Inputs: map[string]string{
			"topic":   "machine learning",
			"context": "context about ML",
		},
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/task", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
	assert.Equal(t, "Task processing failed", response["error"])

	mockClient.AssertExpectations(t)
}

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create router
	router := gin.New()
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "agent-summarizer",
			"timestamp": time.Now().UTC(),
		})
	})

	// Create request
	req := httptest.NewRequest("GET", "/healthz", nil)

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "agent-summarizer", response["service"])
	assert.Contains(t, response, "timestamp")
}

// TestLegacyEndpoint tests the legacy API endpoint
func TestLegacyEndpoint(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create router
	router := gin.New()
	api := router.Group("/api/v1")
	{
		api.POST("/summarize", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Use POST /task endpoint instead",
				"service": "agent-summarizer",
			})
		})
	}

	// Create request
	req := httptest.NewRequest("POST", "/api/v1/summarize", nil)

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Use POST /task endpoint instead", response["message"])
	assert.Equal(t, "agent-summarizer", response["service"])
}

// Benchmark tests
func BenchmarkProcessTask(b *testing.B) {
	// Create mock client
	mockClient := &MockGeminiClient{}

	// Create service with mock client
	service := &SummarizerService{
		geminiClient: mockClient,
		logger:       logrus.New(),
	}

	// Mock successful summarization
	expectedResult := &llm.SummarizeResponse{
		Outline:        []string{"Introduction to topic", "Key concepts"},
		Prerequisites:  []string{"Basic knowledge"},
		Misconceptions: []string{"Common mistake"},
		Citations:      []string{"doc1"},
	}

	mockClient.On("Summarize", mock.Anything, "machine learning", "context about ML").Return(expectedResult, nil)

	// Create task request
	req := adk.TaskRequest{
		SessionID: "test-session",
		Step:      "summarizer",
		Topic:     "machine learning",
		Inputs: map[string]string{
			"topic":   "machine learning",
			"context": "context about ML",
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.ProcessTask(ctx, req)
	}
}

func BenchmarkTaskEndpoint(b *testing.B) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock client
	mockClient := &MockGeminiClient{}

	// Create service with mock client
	service := &SummarizerService{
		geminiClient: mockClient,
		logger:       logrus.New(),
	}

	// Mock successful summarization
	expectedResult := &llm.SummarizeResponse{
		Outline:        []string{"Introduction to topic"},
		Prerequisites:  []string{"Basic knowledge"},
		Misconceptions: []string{"Common mistake"},
		Citations:      []string{"doc1"},
	}

	mockClient.On("Summarize", mock.Anything, "machine learning", "context about ML").Return(expectedResult, nil)

	// Create router
	router := gin.New()
	router.POST("/task", func(c *gin.Context) {
		var req adk.TaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}

		if err := req.Validate(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		response, err := service.ProcessTask(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Task processing failed"})
			return
		}

		c.JSON(http.StatusOK, response)
	})

	// Create request
	reqBody := adk.TaskRequest{
		SessionID: "test-session",
		Step:      "summarizer",
		Topic:     "machine learning",
		Inputs: map[string]string{
			"topic":   "machine learning",
			"context": "context about ML",
		},
	}

	jsonBody, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/task", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
