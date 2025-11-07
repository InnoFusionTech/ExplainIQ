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
	critiqueLessonFunc func(ctx context.Context, lessonJSON string) (*llm.CritiqueResponse, error)
}

func (m *MockGeminiClient) Summarize(ctx context.Context, topic, context string) (*llm.SummarizeResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *MockGeminiClient) ExplainWithOG(ctx context.Context, topic, outline, misconceptions, context string) (*llm.OGLesson, error) {
	return nil, errors.New("not implemented")
}

func (m *MockGeminiClient) CritiqueLesson(ctx context.Context, lessonJSON string) (*llm.CritiqueResponse, error) {
	if m.critiqueLessonFunc != nil {
		return m.critiqueLessonFunc(ctx, lessonJSON)
	}
	return nil, errors.New("mock error")
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

// TestCriticService_ProcessTask_Success tests successful task processing
func TestCriticService_ProcessTask_Success(t *testing.T) {
	// Create mock client
	mockClient := &MockGeminiClient{
		critiqueLessonFunc: func(ctx context.Context, lessonJSON string) (*llm.CritiqueResponse, error) {
			return &llm.CritiqueResponse{
				Issues: []llm.CritiqueIssue{
					{
						Section:  "big_picture",
						Problem:  "Too vague and doesn't provide clear context",
						Severity: "high",
					},
					{
						Section:  "metaphor",
						Problem:  "Metaphor is not clear enough",
						Severity: "medium",
					},
				},
				PatchPlan: []llm.PatchPlanItem{
					{
						Section:         "big_picture",
						Change:          "Provide more specific context and clear definition",
						ReplacementText: "Machine learning is a subset of artificial intelligence that enables computers to learn patterns from data without being explicitly programmed for each task.",
					},
					{
						Section:         "metaphor",
						Change:          "Use a clearer analogy",
						ReplacementText: "Like teaching a child to recognize animals by showing them many pictures until they can identify new animals on their own.",
					},
				},
			}, nil
		},
	}

	// Create service with mock client
	service := &CriticService{
		geminiClient: llm.GeminiClientInterface(mockClient),
		logger:       logrus.New(),
	}

	// Create test request
	lessonJSON := `{
		"big_picture": "Machine learning is cool",
		"metaphor": "It's like magic",
		"core_mechanism": "Algorithms work",
		"toy_example_code": "print('hello')",
		"memory_hook": "ML is fun",
		"real_life": "Used everywhere",
		"best_practices": "Be careful"
	}`

	req := adk.TaskRequest{
		SessionID: "test-session",
		Step:      "critic",
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
	assert.Contains(t, response.Artifacts, "critique")
	assert.Contains(t, response.Artifacts, "patch_plan")
	assert.Contains(t, response.Metrics, "issues_count")
	assert.Contains(t, response.Metrics, "patch_plan_count")
	assert.Contains(t, response.Metrics, "critical_issues")
	assert.Contains(t, response.Metrics, "high_issues")
	assert.Contains(t, response.Metrics, "medium_issues")
	assert.Contains(t, response.Metrics, "low_issues")

	// Verify metrics
	assert.Equal(t, 2, response.Metrics["issues_count"])
	assert.Equal(t, 2, response.Metrics["patch_plan_count"])
	assert.Equal(t, 0, response.Metrics["critical_issues"])
	assert.Equal(t, 1, response.Metrics["high_issues"])
	assert.Equal(t, 1, response.Metrics["medium_issues"])
	assert.Equal(t, 0, response.Metrics["low_issues"])

	// Verify critique JSON can be unmarshaled
	var issues []llm.CritiqueIssue
	err = json.Unmarshal([]byte(response.Artifacts["critique"]), &issues)
	assert.NoError(t, err)
	assert.Len(t, issues, 2)
	assert.Equal(t, "big_picture", issues[0].Section)
	assert.Equal(t, "Too vague and doesn't provide clear context", issues[0].Problem)
	assert.Equal(t, "high", issues[0].Severity)

	// Verify patch plan JSON can be unmarshaled
	var patchPlan []llm.PatchPlanItem
	err = json.Unmarshal([]byte(response.Artifacts["patch_plan"]), &patchPlan)
	assert.NoError(t, err)
	assert.Len(t, patchPlan, 2)
	assert.Equal(t, "big_picture", patchPlan[0].Section)
	assert.Equal(t, "Provide more specific context and clear definition", patchPlan[0].Change)
	assert.Contains(t, patchPlan[0].ReplacementText, "Machine learning is a subset of artificial intelligence")
}

// TestCriticService_ProcessTask_MissingLesson tests error handling for missing lesson
func TestCriticService_ProcessTask_MissingLesson(t *testing.T) {
	// Create mock client
	mockClient := &MockGeminiClient{}

	// Create service with mock client
	service := &CriticService{
		geminiClient: mockClient,
		logger:       logrus.New(),
	}

	// Create test request without lesson
	req := adk.TaskRequest{
		SessionID: "test-session",
		Step:      "critic",
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

// TestCriticService_ProcessTask_CritiqueError tests error handling for critique failure
func TestCriticService_ProcessTask_CritiqueError(t *testing.T) {
	// Create mock client that returns error
	mockClient := &MockGeminiClient{
		critiqueLessonFunc: func(ctx context.Context, lessonJSON string) (*llm.CritiqueResponse, error) {
			return nil, errors.New("critique failed")
		},
	}

	// Create service with mock client
	service := &CriticService{
		geminiClient: mockClient,
		logger:       logrus.New(),
	}

	// Create test request
	req := adk.TaskRequest{
		SessionID: "test-session",
		Step:      "critic",
		Topic:     "machine learning",
		Inputs: map[string]string{
			"lesson": `{"big_picture": "test"}`,
		},
	}

	// Process task
	response, err := service.ProcessTask(context.Background(), req)

	// Verify error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lesson critique failed")
	assert.Equal(t, adk.TaskResponse{}, response)
}

// TestCriticService_ProcessTask_EmptyCritique tests handling of empty critique
func TestCriticService_ProcessTask_EmptyCritique(t *testing.T) {
	// Create mock client
	mockClient := &MockGeminiClient{
		critiqueLessonFunc: func(ctx context.Context, lessonJSON string) (*llm.CritiqueResponse, error) {
			return &llm.CritiqueResponse{
				Issues:    []llm.CritiqueIssue{},
				PatchPlan: []llm.PatchPlanItem{},
			}, nil
		},
	}

	// Create service with mock client
	service := &CriticService{
		geminiClient: mockClient,
		logger:       logrus.New(),
	}

	// Create test request
	req := adk.TaskRequest{
		SessionID: "test-session",
		Step:      "critic",
		Topic:     "machine learning",
		Inputs: map[string]string{
			"lesson": `{"big_picture": "perfect lesson"}`,
		},
	}

	// Process task
	response, err := service.ProcessTask(context.Background(), req)

	// Verify no error occurred
	assert.NoError(t, err)
	assert.NotNil(t, response)

	// Verify empty results
	assert.Equal(t, 0, response.Metrics["issues_count"])
	assert.Equal(t, 0, response.Metrics["patch_plan_count"])
	assert.Equal(t, 0, response.Metrics["critical_issues"])
	assert.Equal(t, 0, response.Metrics["high_issues"])
	assert.Equal(t, 0, response.Metrics["medium_issues"])
	assert.Equal(t, 0, response.Metrics["low_issues"])
}

// TestCriticService_CountIssuesBySeverity tests the severity counting function
func TestCriticService_CountIssuesBySeverity(t *testing.T) {
	service := &CriticService{
		geminiClient: &MockGeminiClient{},
		logger:       logrus.New(),
	}

	issues := []llm.CritiqueIssue{
		{Section: "big_picture", Problem: "Issue 1", Severity: "critical"},
		{Section: "metaphor", Problem: "Issue 2", Severity: "high"},
		{Section: "core_mechanism", Problem: "Issue 3", Severity: "high"},
		{Section: "toy_example_code", Problem: "Issue 4", Severity: "medium"},
		{Section: "memory_hook", Problem: "Issue 5", Severity: "low"},
		{Section: "real_life", Problem: "Issue 6", Severity: "low"},
	}

	assert.Equal(t, 1, service.countIssuesBySeverity(issues, "critical"))
	assert.Equal(t, 2, service.countIssuesBySeverity(issues, "high"))
	assert.Equal(t, 1, service.countIssuesBySeverity(issues, "medium"))
	assert.Equal(t, 2, service.countIssuesBySeverity(issues, "low"))
	assert.Equal(t, 0, service.countIssuesBySeverity(issues, "nonexistent"))
}

// TestCriticService_NewCriticService tests service creation
func TestCriticService_NewCriticService(t *testing.T) {
	service := NewCriticService()

	assert.NotNil(t, service)
	assert.NotNil(t, service.geminiClient)
	assert.NotNil(t, service.logger)
}

// TestCriticService_HTTPHandlers tests HTTP handlers
func TestCriticService_HTTPHandlers(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create service
	service := NewCriticService()

	// Create router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Health check endpoint
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "agent-critic",
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
		assert.Contains(t, w.Body.String(), "agent-critic")
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

// TestCriticService_HTTPHandlers_ValidRequestWithMock tests HTTP handlers with proper mocking
func TestCriticService_HTTPHandlers_ValidRequestWithMock(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock client
	mockClient := &MockGeminiClient{
		critiqueLessonFunc: func(ctx context.Context, lessonJSON string) (*llm.CritiqueResponse, error) {
			return &llm.CritiqueResponse{
				Issues: []llm.CritiqueIssue{
					{
						Section:  "big_picture",
						Problem:  "Test issue",
						Severity: "medium",
					},
				},
				PatchPlan: []llm.PatchPlanItem{
					{
						Section:         "big_picture",
						Change:          "Test change",
						ReplacementText: "Test replacement",
					},
				},
			}, nil
		},
	}

	// Create service with mock client
	service := &CriticService{
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
		Step:      "critic",
		Topic:     "machine learning",
		Inputs: map[string]string{
			"lesson": `{"big_picture": "test lesson"}`,
		},
	}

	reqJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/task", strings.NewReader(string(reqJSON)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "critique")
	assert.Contains(t, w.Body.String(), "patch_plan")
	assert.Contains(t, w.Body.String(), "issues_count")
}
