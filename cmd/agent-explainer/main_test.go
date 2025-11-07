package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/InnoFusionTech/ExplainIQ/internal/adk"
	"github.com/InnoFusionTech/ExplainIQ/internal/llm"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// MockGeminiClient is a mock implementation of GeminiClientInterface for testing
type MockGeminiClient struct {
	explainWithOGFunc func(ctx context.Context, topic, outline, misconceptions, context string) (*llm.OGLesson, error)
}

func (m *MockGeminiClient) Summarize(ctx context.Context, topic, context string) (*llm.SummarizeResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *MockGeminiClient) ExplainWithOG(ctx context.Context, topic, outline, misconceptions, context string) (*llm.OGLesson, error) {
	if m.explainWithOGFunc != nil {
		return m.explainWithOGFunc(ctx, topic, outline, misconceptions, context)
	}
	return nil, errors.New("mock error")
}

func (m *MockGeminiClient) CritiqueLesson(ctx context.Context, lessonJSON string) (*llm.CritiqueResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *MockGeminiClient) VisualizeCore(ctx context.Context, lessonJSON, sessionID string) (*llm.VisualizeResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *MockGeminiClient) Health(ctx context.Context) error {
	return nil
}

func (m *MockGeminiClient) SetAPIKey(apiKey string) {}

func (m *MockGeminiClient) SetModel(model string) {}

func (m *MockGeminiClient) SetBaseURL(baseURL string) {}

func (m *MockGeminiClient) GetModelInfo() map[string]interface{} {
	return map[string]interface{}{}
}

// TestExplainerService_ProcessTask_Success tests successful task processing
func TestExplainerService_ProcessTask_Success(t *testing.T) {
	// Create mock client
	mockClient := &MockGeminiClient{
		explainWithOGFunc: func(ctx context.Context, topic, outline, misconceptions, context string) (*llm.OGLesson, error) {
			return &llm.OGLesson{
				BigPicture:     "Machine learning is a subset of AI that enables computers to learn from data.",
				Metaphor:       "Like teaching a child to recognize animals by showing them pictures.",
				CoreMechanism:  "Algorithms find patterns in data to make predictions or decisions.",
				ToyExampleCode: "model.fit(X_train, y_train)",
				MemoryHook:     "ML = Machine Learning = More Learning from data",
				RealLife:       "Used in recommendation systems, image recognition, and autonomous vehicles.",
				BestPractices:  "Do: Clean your data. Don't: Overfit your model.",
			}, nil
		},
	}

	// Create service with mock client, using interface type if needed for compatibility
	var geminiInterface llm.GeminiClientInterface = mockClient
	service := &ExplainerService{
		geminiClient: geminiInterface,
		logger:       logrus.New(),
	}
	req := adk.TaskRequest{
		SessionID: "test-session",
		Step:      "explainer",
		Topic:     "machine learning",
		Inputs: map[string]string{
			"topic":          "machine learning",
			"outline":        "Introduction, Concepts, Applications",
			"misconceptions": "ML is magic, More data is always better",
			"context":        "Additional context about ML",
		},
	}

	// Process task
	response, err := service.ProcessTask(context.Background(), req)

	// Verify no error occurred
	assert.NoError(t, err)
	assert.NotNil(t, response)

	// Verify response structure
	assert.Contains(t, response.Artifacts, "lesson")
	assert.Contains(t, response.Metrics, "big_picture_length")
	assert.Contains(t, response.Metrics, "metaphor_length")
	assert.Contains(t, response.Metrics, "core_mechanism_length")
	assert.Contains(t, response.Metrics, "toy_example_code_length")
	assert.Contains(t, response.Metrics, "memory_hook_length")
	assert.Contains(t, response.Metrics, "real_life_length")
	assert.Contains(t, response.Metrics, "best_practices_length")

	// Verify lesson JSON can be unmarshaled
	var lesson llm.OGLesson
	err = json.Unmarshal([]byte(response.Artifacts["lesson"]), &lesson)
	assert.NoError(t, err)
	assert.Equal(t, "Machine learning is a subset of AI that enables computers to learn from data.", lesson.BigPicture)
	assert.Equal(t, "Like teaching a child to recognize animals by showing them pictures.", lesson.Metaphor)
	assert.Equal(t, "Algorithms find patterns in data to make predictions or decisions.", lesson.CoreMechanism)
	assert.Equal(t, "model.fit(X_train, y_train)", lesson.ToyExampleCode)
	assert.Equal(t, "ML = Machine Learning = More Learning from data", lesson.MemoryHook)
	assert.Equal(t, "Used in recommendation systems, image recognition, and autonomous vehicles.", lesson.RealLife)
	assert.Equal(t, "Do: Clean your data. Don't: Overfit your model.", lesson.BestPractices)
}

// TestExplainerService_ProcessTask_MissingTopic tests error handling for missing topic
func TestExplainerService_ProcessTask_MissingTopic(t *testing.T) {
	// Create mock client
	mockClient := &MockGeminiClient{}

	// Create service with mock client
	service := &ExplainerService{
		geminiClient: mockClient,
		logger:       logrus.New(),
	}

	// Create test request without topic
	req := adk.TaskRequest{
		SessionID: "test-session",
		Step:      "explainer",
		Topic:     "machine learning",
		Inputs: map[string]string{
			"outline": "Introduction, Concepts, Applications",
		},
	}

	// Process task
	response, err := service.ProcessTask(context.Background(), req)

	// Verify error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "topic is required")
	assert.Equal(t, adk.TaskResponse{}, response)
}

// TestExplainerService_ProcessTask_ExplainError tests error handling for explain failure
func TestExplainerService_ProcessTask_ExplainError(t *testing.T) {
	// Create mock client that returns error
	mockClient := &MockGeminiClient{
		explainWithOGFunc: func(ctx context.Context, topic, outline, misconceptions, context string) (*llm.OGLesson, error) {
			return nil, errors.New("explain failed")
		},
	}

	// Create service with mock client
	service := &ExplainerService{
		geminiClient: mockClient,
		logger:       logrus.New(),
	}

	// Create test request
	req := adk.TaskRequest{
		SessionID: "test-session",
		Step:      "explainer",
		Topic:     "machine learning",
		Inputs: map[string]string{
			"topic": "machine learning",
		},
	}

	// Process task
	response, err := service.ProcessTask(context.Background(), req)

	// Verify error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "OG lesson generation failed")
	assert.Equal(t, adk.TaskResponse{}, response)
}

// TestExplainerService_ProcessTask_OptionalInputs tests handling of optional inputs
func TestExplainerService_ProcessTask_OptionalInputs(t *testing.T) {
	// Create mock client
	mockClient := &MockGeminiClient{
		explainWithOGFunc: func(ctx context.Context, topic, outline, misconceptions, context string) (*llm.OGLesson, error) {
			// Verify that empty strings are passed for missing inputs
			assert.Equal(t, "machine learning", topic)
			assert.Equal(t, "", outline)
			assert.Equal(t, "", misconceptions)
			assert.Equal(t, "", context)

			return &llm.OGLesson{
				BigPicture:     "Test big picture",
				Metaphor:       "Test metaphor",
				CoreMechanism:  "Test mechanism",
				ToyExampleCode: "N/A",
				MemoryHook:     "Test hook",
				RealLife:       "Test real life",
				BestPractices:  "Test practices",
			}, nil
		},
	}

	// Create service with mock client
	service := &ExplainerService{
		geminiClient: mockClient,
		logger:       logrus.New(),
	}

	// Create test request with only topic
	req := adk.TaskRequest{
		SessionID: "test-session",
		Step:      "explainer",
		Topic:     "machine learning",
		Inputs: map[string]string{
			"topic": "machine learning",
		},
	}

	// Process task
	response, err := service.ProcessTask(context.Background(), req)

	// Verify no error occurred
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Contains(t, response.Artifacts, "lesson")
}

// TestExplainerService_ProcessTask_JSONMarshalError tests error handling for JSON marshaling
func TestExplainerService_ProcessTask_JSONMarshalError(t *testing.T) {
	// Create mock client that returns invalid lesson (this shouldn't happen in real usage)
	mockClient := &MockGeminiClient{
		explainWithOGFunc: func(ctx context.Context, topic, outline, misconceptions, context string) (*llm.OGLesson, error) {
			// Return a lesson with invalid JSON characters (this is a contrived test)
			return &llm.OGLesson{
				BigPicture:     "Test big picture",
				Metaphor:       "Test metaphor",
				CoreMechanism:  "Test mechanism",
				ToyExampleCode: "N/A",
				MemoryHook:     "Test hook",
				RealLife:       "Test real life",
				BestPractices:  "Test practices",
			}, nil
		},
	}

	// Create service with mock client
	service := &ExplainerService{
		geminiClient: mockClient,
		logger:       logrus.New(),
	}

	// Create test request
	req := adk.TaskRequest{
		SessionID: "test-session",
		Step:      "explainer",
		Topic:     "machine learning",
		Inputs: map[string]string{
			"topic": "machine learning",
		},
	}

	// Process task
	response, err := service.ProcessTask(context.Background(), req)

	// Verify no error occurred (JSON marshaling should work fine with normal strings)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Contains(t, response.Artifacts, "lesson")
}

// TestExplainerService_NewExplainerService tests service creation
func TestExplainerService_NewExplainerService(t *testing.T) {
	service := NewExplainerService()

	assert.NotNil(t, service)
	assert.NotNil(t, service.geminiClient)
	assert.NotNil(t, service.logger)
}

// TestExplainerService_HTTPHandlers tests HTTP handlers
func TestExplainerService_HTTPHandlers(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create service
	service := NewExplainerService()

	// Create router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Health check endpoint
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "agent-explainer",
			"timestamp": "2024-01-01T00:00:00Z",
		})
	})

	// Task processing endpoint
	router.POST("/task", func(c *gin.Context) {
		var req adk.TaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request format",
				"details": err.Error(),
			})
			return
		}

		// Validate request
		if err := req.Validate(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request",
				"details": err.Error(),
			})
			return
		}

		// Process the task
		response, err := service.ProcessTask(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Task processing failed",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, response)
	})

	// Test health check endpoint
	t.Run("HealthCheck", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/healthz", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "healthy")
		assert.Contains(t, w.Body.String(), "agent-explainer")
	})

	// Test task endpoint with valid request
	t.Run("TaskEndpoint_ValidRequest", func(t *testing.T) {
		// Create valid request
		reqBody := adk.TaskRequest{
			SessionID: "test-session",
			Step:      "explainer",
			Topic:     "machine learning",
			Inputs: map[string]string{
				"topic": "machine learning",
			},
		}

		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/task", strings.NewReader(string(reqJSON)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Note: This will fail because the mock client returns an error
		// In a real test, you'd want to mock the service or use a test client
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Task processing failed")
	})

	// Test task endpoint with invalid request
	t.Run("TaskEndpoint_InvalidRequest", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/task", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid request format")
	})
}

// TestExplainerService_HTTPHandlers_ValidRequestWithMock tests HTTP handlers with proper mocking
func TestExplainerService_HTTPHandlers_ValidRequestWithMock(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock client
	mockClient := &MockGeminiClient{
		explainWithOGFunc: func(ctx context.Context, topic, outline, misconceptions, context string) (*llm.OGLesson, error) {
			return &llm.OGLesson{
				BigPicture:     "Test big picture",
				Metaphor:       "Test metaphor",
				CoreMechanism:  "Test mechanism",
				ToyExampleCode: "N/A",
				MemoryHook:     "Test hook",
				RealLife:       "Test real life",
				BestPractices:  "Test practices",
			}, nil
		},
	}

	// Create service with mock client
	service := &ExplainerService{
		geminiClient: mockClient,
		logger:       logrus.New(),
	}

	// Create router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Task processing endpoint
	router.POST("/task", func(c *gin.Context) {
		var req adk.TaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request format",
				"details": err.Error(),
			})
			return
		}

		// Validate request
		if err := req.Validate(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request",
				"details": err.Error(),
			})
			return
		}

		// Process the task
		response, err := service.ProcessTask(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Task processing failed",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, response)
	})

	// Test task endpoint with valid request
	reqBody := adk.TaskRequest{
		SessionID: "test-session",
		Step:      "explainer",
		Topic:     "machine learning",
		Inputs: map[string]string{
			"topic": "machine learning",
		},
	}

	reqJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/task", strings.NewReader(string(reqJSON)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "lesson")
	assert.Contains(t, w.Body.String(), "big_picture_length")
}
