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
	visualizeCoreFunc func(ctx context.Context, lessonJSON, sessionID string) (*llm.VisualizeResponse, error)
}

func (m *MockGeminiClient) Summarize(ctx context.Context, topic, context string) (*llm.SummarizeResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *MockGeminiClient) ExplainWithOG(ctx context.Context, topic, outline, misconceptions, context string) (*llm.OGLesson, error) {
	return nil, errors.New("not implemented")
}

func (m *MockGeminiClient) CritiqueLesson(ctx context.Context, lessonJSON string) (*llm.CritiqueResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *MockGeminiClient) VisualizeCore(ctx context.Context, lessonJSON, sessionID string) (*llm.VisualizeResponse, error) {
	if m.visualizeCoreFunc != nil {
		return m.visualizeCoreFunc(ctx, lessonJSON, sessionID)
	}
	return nil, errors.New("mock error")
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

// TestVisualizerService_ProcessTask_Success tests successful task processing
func TestVisualizerService_ProcessTask_Success(t *testing.T) {
	// Create mock client
	mockClient := &MockGeminiClient{
		visualizeCoreFunc: func(ctx context.Context, lessonJSON, sessionID string) (*llm.VisualizeResponse, error) {
			return &llm.VisualizeResponse{
				Images: []llm.ImageRef{
					{
						URL:     "https://storage.googleapis.com/test-bucket/sessions/test-session/diagram_1.png",
						AltText: "Diagram 1 illustrating machine learning algorithms",
						Caption: "Core mechanism diagram: machine learning algorithms",
					},
					{
						URL:     "https://storage.googleapis.com/test-bucket/sessions/test-session/diagram_2.png",
						AltText: "Diagram 2 illustrating machine learning algorithms",
						Caption: "Process flowchart: machine learning algorithms",
					},
				},
				Captions: []string{
					"Core mechanism diagram: machine learning algorithms",
					"Process flowchart: machine learning algorithms",
				},
			}, nil
		},
	}

	// Create service with mock client
	service := &VisualizerService{
		geminiClient: mockClient,
		logger:       logrus.New(),
	}

	// Create test request
	lessonJSON := `{
		"big_picture": "Machine learning is a subset of AI",
		"metaphor": "Like teaching a child",
		"core_mechanism": "Algorithms find patterns in data to make predictions",
		"toy_example_code": "model.fit(X, y)",
		"memory_hook": "ML = More Learning",
		"real_life": "Used in recommendation systems",
		"best_practices": "Clean your data"
	}`

	req := adk.TaskRequest{
		SessionID: "test-session",
		Step:      "visualizer",
		Topic:     "machine learning",
		Inputs: map[string]string{
			"lesson": lessonJSON,
		},
	}

	// Process task
	response, err := service.ProcessTask(context.Background(), req)

	// Verify no error occurred
	assert.NoError(t, err)
	assert.NotNil(t, response)

	// Verify response structure
	assert.Contains(t, response.Artifacts, "images")
	assert.Contains(t, response.Artifacts, "captions")
	assert.Contains(t, response.Metrics, "images_count")
	assert.Contains(t, response.Metrics, "captions_count")

	// Verify metrics
	assert.Equal(t, 2, response.Metrics["images_count"])
	assert.Equal(t, 2, response.Metrics["captions_count"])

	// Verify images JSON can be unmarshaled
	var images []llm.ImageRef
	err = json.Unmarshal([]byte(response.Artifacts["images"]), &images)
	assert.NoError(t, err)
	assert.Len(t, images, 2)
	assert.Equal(t, "https://storage.googleapis.com/test-bucket/sessions/test-session/diagram_1.png", images[0].URL)
	assert.Equal(t, "Diagram 1 illustrating machine learning algorithms", images[0].AltText)
	assert.Equal(t, "Core mechanism diagram: machine learning algorithms", images[0].Caption)

	// Verify captions JSON can be unmarshaled
	var captions []string
	err = json.Unmarshal([]byte(response.Artifacts["captions"]), &captions)
	assert.NoError(t, err)
	assert.Len(t, captions, 2)
	assert.Equal(t, "Core mechanism diagram: machine learning algorithms", captions[0])
	assert.Equal(t, "Process flowchart: machine learning algorithms", captions[1])
}

// TestVisualizerService_ProcessTask_MissingLesson tests error handling for missing lesson
func TestVisualizerService_ProcessTask_MissingLesson(t *testing.T) {
	// Create mock client
	mockClient := &MockGeminiClient{}

	// Create service with mock client
	service := &VisualizerService{
		geminiClient: mockClient,
		logger:       logrus.New(),
	}

	// Create test request without lesson
	req := adk.TaskRequest{
		SessionID: "test-session",
		Step:      "visualizer",
		Topic:     "machine learning",
		Inputs: map[string]string{
			"other_input": "some value",
		},
	}

	// Process task
	response, err := service.ProcessTask(context.Background(), req)

	// Verify error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lesson JSON is required")
	assert.Equal(t, adk.TaskResponse{}, response)
}

// TestVisualizerService_ProcessTask_VisualizationError tests error handling for visualization failure
func TestVisualizerService_ProcessTask_VisualizationError(t *testing.T) {
	// Create mock client that returns error
	mockClient := &MockGeminiClient{
		visualizeCoreFunc: func(ctx context.Context, lessonJSON, sessionID string) (*llm.VisualizeResponse, error) {
			return nil, errors.New("visualization failed")
		},
	}

	// Create service with mock client
	service := &VisualizerService{
		geminiClient: mockClient,
		logger:       logrus.New(),
	}

	// Create test request
	req := adk.TaskRequest{
		SessionID: "test-session",
		Step:      "visualizer",
		Topic:     "machine learning",
		Inputs: map[string]string{
			"lesson": `{"core_mechanism": "test mechanism"}`,
		},
	}

	// Process task
	response, err := service.ProcessTask(context.Background(), req)

	// Verify error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "visualization generation failed")
	assert.Equal(t, adk.TaskResponse{}, response)
}

// TestVisualizerService_ProcessTask_EmptyVisualization tests handling of empty visualization
func TestVisualizerService_ProcessTask_EmptyVisualization(t *testing.T) {
	// Create mock client
	mockClient := &MockGeminiClient{
		visualizeCoreFunc: func(ctx context.Context, lessonJSON, sessionID string) (*llm.VisualizeResponse, error) {
			return &llm.VisualizeResponse{
				Images:   []llm.ImageRef{},
				Captions: []string{},
			}, nil
		},
	}

	// Create service with mock client
	service := &VisualizerService{
		geminiClient: mockClient,
		logger:       logrus.New(),
	}

	// Create test request
	req := adk.TaskRequest{
		SessionID: "test-session",
		Step:      "visualizer",
		Topic:     "machine learning",
		Inputs: map[string]string{
			"lesson": `{"core_mechanism": "test mechanism"}`,
		},
	}

	// Process task
	response, err := service.ProcessTask(context.Background(), req)

	// Verify no error occurred
	assert.NoError(t, err)
	assert.NotNil(t, response)

	// Verify empty results
	assert.Equal(t, 0, response.Metrics["images_count"])
	assert.Equal(t, 0, response.Metrics["captions_count"])
}

// TestVisualizerService_NewVisualizerService tests service creation
func TestVisualizerService_NewVisualizerService(t *testing.T) {
	service := NewVisualizerService()

	assert.NotNil(t, service)
	assert.NotNil(t, service.geminiClient)
	assert.NotNil(t, service.logger)
}

// TestVisualizerService_HTTPHandlers tests HTTP handlers
func TestVisualizerService_HTTPHandlers(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create service
	service := NewVisualizerService()

	// Create router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Health check endpoint
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "agent-visualizer",
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
		assert.Contains(t, w.Body.String(), "agent-visualizer")
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

// TestVisualizerService_HTTPHandlers_ValidRequestWithMock tests HTTP handlers with proper mocking
func TestVisualizerService_HTTPHandlers_ValidRequestWithMock(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock client
	mockClient := &MockGeminiClient{
		visualizeCoreFunc: func(ctx context.Context, lessonJSON, sessionID string) (*llm.VisualizeResponse, error) {
			return &llm.VisualizeResponse{
				Images: []llm.ImageRef{
					{
						URL:     "https://storage.googleapis.com/test-bucket/sessions/test-session/diagram_1.png",
						AltText: "Test diagram",
						Caption: "Test caption",
					},
				},
				Captions: []string{"Test caption"},
			}, nil
		},
	}

	// Create service with mock client
	service := &VisualizerService{
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
		Step:      "visualizer",
		Topic:     "machine learning",
		Inputs: map[string]string{
			"lesson": `{"core_mechanism": "test mechanism"}`,
		},
	}

	reqJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/task", strings.NewReader(string(reqJSON)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "images")
	assert.Contains(t, w.Body.String(), "captions")
	assert.Contains(t, w.Body.String(), "images_count")
}



