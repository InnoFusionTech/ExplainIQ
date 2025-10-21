package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/creduntvitam/explainiq/internal/adk"
	"github.com/creduntvitam/explainiq/internal/elastic"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockElasticClient is a mock implementation of the elastic client
type MockElasticClient struct {
	mock.Mock
}

func (m *MockElasticClient) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockElasticClient) HybridSearch(ctx context.Context, index, query string, embedding []float32, size int) (*elastic.SearchResult, error) {
	args := m.Called(ctx, index, query, embedding, size)
	return args.Get(0).(*elastic.SearchResult), args.Error(1)
}

// MockElasticRetriever is a mock implementation of the elastic retriever
type MockElasticRetriever struct {
	mock.Mock
}

func (m *MockElasticRetriever) HybridSearch(ctx context.Context, index, query string, k int) ([]elastic.SearchResult, error) {
	args := m.Called(ctx, index, query, k)
	return args.Get(0).([]elastic.SearchResult), args.Error(1)
}

// MockEmbeddingClient is a mock implementation of the embedding client
type MockEmbeddingClient struct {
	mock.Mock
}

func (m *MockEmbeddingClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	args := m.Called(ctx, texts)
	return args.Get(0).([][]float32), args.Error(1)
}

// MockADKClient is a mock implementation of the ADK client
type MockADKClient struct {
	mock.Mock
}

func (m *MockADKClient) DoTask(ctx context.Context, url string, req adk.TaskRequest) (adk.TaskResponse, error) {
	args := m.Called(ctx, url, req)
	return args.Get(0).(adk.TaskResponse), args.Error(1)
}

func (m *MockADKClient) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// TestNewPipeline tests pipeline creation
func TestNewPipeline(t *testing.T) {
	config := DefaultPipelineConfig()

	// This test would require actual external dependencies, so we'll test the config creation
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 2*time.Second, config.RetryDelay)
	assert.Equal(t, 5*time.Minute, config.StepTimeout)
	assert.Equal(t, 5, config.ContextTopK)
	assert.Equal(t, "lessons", config.ElasticIndex)
	assert.NotEmpty(t, config.AgentBaseURLs)
}

// TestPipelineHappyPath tests the happy path scenario
func TestPipelineHappyPath(t *testing.T) {
	// Create mock components
	mockElasticClient := &MockElasticClient{}
	mockElasticRetriever := &MockElasticRetriever{}
	mockEmbeddingClient := &MockEmbeddingClient{}
	mockADKClients := map[string]*MockADKClient{
		"summarizer": &MockADKClient{},
		"explainer":  &MockADKClient{},
		"visualizer": &MockADKClient{},
		"critic":     &MockADKClient{},
	}

	// Create pipeline with mocked components
	pipeline := &Pipeline{
		config:           DefaultPipelineConfig(),
		logger:           logrus.New(),
		elasticClient:    mockElasticClient,
		elasticRetriever: mockElasticRetriever,
		embeddingClient:  mockEmbeddingClient,
		adkClients:       make(map[string]*adk.Client),
	}

	// Create orchestrator
	orchestrator := NewOrchestrator()
	session := orchestrator.CreateSession("test topic")

	// Mock context retrieval
	contextDocs := []elastic.SearchResult{
		{
			Doc: elastic.Doc{
				ID:      "doc1",
				Topic:   "test topic",
				Section: "introduction",
				Text:    "This is test content about the topic",
			},
			Score:   0.95,
			Snippet: "This is test content about the topic",
		},
	}
	mockElasticRetriever.On("HybridSearch", mock.Anything, "lessons", "test topic", 5).Return(contextDocs, nil)

	// Mock successful agent responses
	successResponse := adk.TaskResponse{
		Artifacts: map[string]string{
			"summary": "Test summary",
			"lesson":  "Test lesson content",
		},
		Metrics: map[string]interface{}{
			"tokens_used": 100,
		},
	}

	// Mock all agents to return success
	for _, mockClient := range mockADKClients {
		mockClient.On("DoTask", mock.Anything, "/process", mock.AnythingOfType("adk.TaskRequest")).Return(successResponse, nil)
		mockClient.On("Health", mock.Anything).Return(nil)
	}

	// Mock elastic client health
	mockElasticClient.On("Health", mock.Anything).Return(nil)

	// Execute pipeline
	ctx := context.Background()
	err := pipeline.runPipeline(ctx, session.ID, orchestrator)

	// Verify no error occurred
	assert.NoError(t, err)

	// Verify session was completed
	updatedSession, exists := orchestrator.GetSession(session.ID)
	require.True(t, exists)
	assert.Equal(t, "completed", updatedSession.Status)
	assert.NotNil(t, updatedSession.Result)

	// Verify all mocks were called
	mockElasticRetriever.AssertExpectations(t)
	for _, mockClient := range mockADKClients {
		mockClient.AssertExpectations(t)
	}
	mockElasticClient.AssertExpectations(t)
}

// TestPipelineFailingAgent tests the scenario where an agent fails
func TestPipelineFailingAgent(t *testing.T) {
	// Create mock components
	mockElasticClient := &MockElasticClient{}
	mockElasticRetriever := &MockElasticRetriever{}
	mockEmbeddingClient := &MockEmbeddingClient{}
	mockADKClients := map[string]*MockADKClient{
		"summarizer": &MockADKClient{},
		"explainer":  &MockADKClient{},
		"visualizer": &MockADKClient{},
		"critic":     &MockADKClient{},
	}

	// Create pipeline with mocked components
	pipeline := &Pipeline{
		config:           DefaultPipelineConfig(),
		logger:           logrus.New(),
		elasticClient:    mockElasticClient,
		elasticRetriever: mockElasticRetriever,
		embeddingClient:  mockEmbeddingClient,
		adkClients:       make(map[string]*adk.Client),
	}

	// Create orchestrator
	orchestrator := NewOrchestrator()
	session := orchestrator.CreateSession("test topic")

	// Mock context retrieval
	contextDocs := []elastic.SearchResult{
		{
			Doc: elastic.Doc{
				ID:      "doc1",
				Topic:   "test topic",
				Section: "introduction",
				Text:    "This is test content about the topic",
			},
			Score:   0.95,
			Snippet: "This is test content about the topic",
		},
	}
	mockElasticRetriever.On("HybridSearch", mock.Anything, "lessons", "test topic", 5).Return(contextDocs, nil)

	// Mock successful responses for most agents
	successResponse := adk.TaskResponse{
		Artifacts: map[string]string{
			"summary": "Test summary",
			"lesson":  "Test lesson content",
		},
		Metrics: map[string]interface{}{
			"tokens_used": 100,
		},
	}

	// Mock summarizer to succeed
	mockADKClients["summarizer"].On("DoTask", mock.Anything, "/process", mock.AnythingOfType("adk.TaskRequest")).Return(successResponse, nil)
	mockADKClients["summarizer"].On("Health", mock.Anything).Return(nil)

	// Mock explainer to fail with retryable error
	retryableError := &adk.TaskError{
		Code:    "NETWORK_ERROR",
		Message: "Network timeout",
		Details: "Connection timed out",
	}
	mockADKClients["explainer"].On("DoTask", mock.Anything, "/process", mock.AnythingOfType("adk.TaskRequest")).Return(adk.TaskResponse{}, retryableError).Times(4) // 1 initial + 3 retries
	mockADKClients["explainer"].On("Health", mock.Anything).Return(nil)

	// Mock visualizer and critic to succeed
	mockADKClients["visualizer"].On("DoTask", mock.Anything, "/process", mock.AnythingOfType("adk.TaskRequest")).Return(successResponse, nil)
	mockADKClients["visualizer"].On("Health", mock.Anything).Return(nil)
	mockADKClients["critic"].On("DoTask", mock.Anything, "/process", mock.AnythingOfType("adk.TaskRequest")).Return(successResponse, nil)
	mockADKClients["critic"].On("Health", mock.Anything).Return(nil)

	// Mock elastic client health
	mockElasticClient.On("Health", mock.Anything).Return(nil)

	// Execute pipeline
	ctx := context.Background()
	err := pipeline.runPipeline(ctx, session.ID, orchestrator)

	// Verify error occurred due to explainer failure
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "explainer")

	// Verify session was marked as failed
	updatedSession, exists := orchestrator.GetSession(session.ID)
	require.True(t, exists)
	assert.Equal(t, "failed", updatedSession.Status)

	// Verify all mocks were called
	mockElasticRetriever.AssertExpectations(t)
	for _, mockClient := range mockADKClients {
		mockClient.AssertExpectations(t)
	}
	mockElasticClient.AssertExpectations(t)
}

// TestPipelineContextRetrieval tests context retrieval functionality
func TestPipelineContextRetrieval(t *testing.T) {
	// Create mock components
	mockElasticRetriever := &MockElasticRetriever{}

	// Create pipeline with mocked components
	pipeline := &Pipeline{
		config:           DefaultPipelineConfig(),
		logger:           logrus.New(),
		elasticRetriever: mockElasticRetriever,
	}

	// Mock context retrieval
	contextDocs := []elastic.SearchResult{
		{
			Doc: elastic.Doc{
				ID:      "doc1",
				Topic:   "machine learning",
				Section: "introduction",
				Text:    "Machine learning is a subset of artificial intelligence",
			},
			Score:   0.95,
			Snippet: "Machine learning is a subset of artificial intelligence",
		},
		{
			Doc: elastic.Doc{
				ID:      "doc2",
				Topic:   "machine learning",
				Section: "algorithms",
				Text:    "Common algorithms include linear regression and neural networks",
			},
			Score:   0.87,
			Snippet: "Common algorithms include linear regression and neural networks",
		},
	}
	mockElasticRetriever.On("HybridSearch", mock.Anything, "lessons", "machine learning", 5).Return(contextDocs, nil)

	// Test context retrieval
	ctx := context.Background()
	docs, err := pipeline.getContext(ctx, "test-session", "machine learning")

	// Verify no error occurred
	assert.NoError(t, err)
	assert.Len(t, docs, 2)
	assert.Equal(t, "doc1", docs[0].Doc.ID)
	assert.Equal(t, "doc2", docs[1].Doc.ID)

	// Test context formatting
	formattedContext := pipeline.formatContext(docs)
	assert.Contains(t, formattedContext, "Document 1")
	assert.Contains(t, formattedContext, "Document 2")
	assert.Contains(t, formattedContext, "Machine learning is a subset")
	assert.Contains(t, formattedContext, "Common algorithms include")

	mockElasticRetriever.AssertExpectations(t)
}

// TestPipelineCriticPatch tests critic patch application
func TestPipelineCriticPatch(t *testing.T) {
	// Create pipeline
	pipeline := &Pipeline{
		config: DefaultPipelineConfig(),
		logger: logrus.New(),
	}

	// Create orchestrator and session
	orchestrator := NewOrchestrator()
	session := orchestrator.CreateSession("test topic")

	// Create initial session result
	session.Result = &SessionResult{
		Lesson: `{"title": "Test Lesson", "content": "Original content"}`,
	}
	orchestrator.UpdateSession(session)

	// Mock critic output
	criticOutput := map[string]string{
		"lesson":      `{"title": "Test Lesson", "content": "Improved content", "suggestions": ["Add examples", "Clarify concepts"]}`,
		"suggestions": "Add more examples and clarify key concepts",
	}

	// Apply critic patch
	ctx := context.Background()
	err := pipeline.applyCriticPatch(ctx, session.ID, criticOutput, orchestrator)

	// Verify no error occurred
	assert.NoError(t, err)

	// Verify session was updated
	updatedSession, exists := orchestrator.GetSession(session.ID)
	require.True(t, exists)
	assert.NotNil(t, updatedSession.Result)

	// Parse the updated lesson JSON
	var lesson map[string]interface{}
	err = json.Unmarshal([]byte(updatedSession.Result.Lesson), &lesson)
	assert.NoError(t, err)

	// Verify critic suggestions were added
	assert.True(t, lesson["critic_reviewed"].(bool))
	assert.Equal(t, "Add more examples and clarify key concepts", lesson["critic_suggestions"])
}

// TestPipelineSSEEvents tests SSE event broadcasting during pipeline execution
func TestPipelineSSEEvents(t *testing.T) {
	// Create orchestrator
	orchestrator := NewOrchestrator()
	session := orchestrator.CreateSession("test topic")

	// Add client to receive events
	client := make(chan SSEEvent, 50)
	orchestrator.AddClient(session.ID, client)
	defer orchestrator.RemoveClient(session.ID, client)

	// Create a simple pipeline that will generate events
	// We'll use the actual pipeline but with a short timeout for testing
	config := DefaultPipelineConfig()
	config.StepTimeout = 1 * time.Second // Short timeout for testing
	config.MaxRetries = 1                // Reduce retries for faster testing

	// Note: This test would require actual external dependencies
	// In a real test environment, you would mock the external services
	// For now, we'll test the event broadcasting mechanism

	// Start pipeline execution in goroutine
	go func() {
		// This would normally call pipeline.runPipeline, but we'll simulate it
		// by broadcasting some test events
		orchestrator.BroadcastEvent(session.ID, SSEEvent{
			Type:      "step-start",
			SessionID: session.ID,
			StepID:    "step-1",
			Data: map[string]interface{}{
				"step_name": "summarizer",
			},
			Timestamp: time.Now(),
		})

		orchestrator.BroadcastEvent(session.ID, SSEEvent{
			Type:      "step-complete",
			SessionID: session.ID,
			StepID:    "step-1",
			Data: map[string]interface{}{
				"step_name": "summarizer",
				"status":    "completed",
			},
			Timestamp: time.Now(),
		})
	}()

	// Collect events
	var events []SSEEvent
	timeout := time.After(2 * time.Second)

	for {
		select {
		case event := <-client:
			events = append(events, event)
			if len(events) >= 2 {
				goto done
			}
		case <-timeout:
			t.Fatal("Did not receive expected events within timeout")
		}
	}

done:
	// Verify we received the expected events
	assert.Len(t, events, 2)
	assert.Equal(t, "step-start", events[0].Type)
	assert.Equal(t, "step-complete", events[1].Type)
	assert.Equal(t, session.ID, events[0].SessionID)
	assert.Equal(t, session.ID, events[1].SessionID)
}

// TestPipelineConfiguration tests pipeline configuration
func TestPipelineConfiguration(t *testing.T) {
	config := DefaultPipelineConfig()

	// Test default values
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 2*time.Second, config.RetryDelay)
	assert.Equal(t, 5*time.Minute, config.StepTimeout)
	assert.Equal(t, 5, config.ContextTopK)
	assert.Equal(t, "lessons", config.ElasticIndex)

	// Test agent URLs
	assert.Contains(t, config.AgentBaseURLs, "summarizer")
	assert.Contains(t, config.AgentBaseURLs, "explainer")
	assert.Contains(t, config.AgentBaseURLs, "visualizer")
	assert.Contains(t, config.AgentBaseURLs, "critic")

	// Test custom configuration
	customConfig := PipelineConfig{
		MaxRetries:   5,
		RetryDelay:   3 * time.Second,
		StepTimeout:  10 * time.Minute,
		ContextTopK:  10,
		ElasticIndex: "custom-index",
		AgentBaseURLs: map[string]string{
			"custom-agent": "http://localhost:9999",
		},
	}

	assert.Equal(t, 5, customConfig.MaxRetries)
	assert.Equal(t, 3*time.Second, customConfig.RetryDelay)
	assert.Equal(t, 10*time.Minute, customConfig.StepTimeout)
	assert.Equal(t, 10, customConfig.ContextTopK)
	assert.Equal(t, "custom-index", customConfig.ElasticIndex)
	assert.Contains(t, customConfig.AgentBaseURLs, "custom-agent")
}

// TestPipelineStepResult tests pipeline step result structure
func TestPipelineStepResult(t *testing.T) {
	stepResult := PipelineStepResult{
		StepName:   "test-step",
		Status:     "completed",
		Output:     map[string]string{"result": "success"},
		Duration:   5 * time.Second,
		RetryCount: 0,
		Metadata:   map[string]interface{}{"tokens": 100},
	}

	assert.Equal(t, "test-step", stepResult.StepName)
	assert.Equal(t, "completed", stepResult.Status)
	assert.Equal(t, "success", stepResult.Output["result"])
	assert.Equal(t, 5*time.Second, stepResult.Duration)
	assert.Equal(t, 0, stepResult.RetryCount)
	assert.Equal(t, 100, stepResult.Metadata["tokens"])
}

// TestPipelineResult tests pipeline result structure
func TestPipelineResult(t *testing.T) {
	stepResults := []PipelineStepResult{
		{
			StepName: "step1",
			Status:   "completed",
			Output:   map[string]string{"result1": "value1"},
		},
		{
			StepName: "step2",
			Status:   "completed",
			Output:   map[string]string{"result2": "value2"},
		},
	}

	result := PipelineResult{
		SessionID:   "test-session",
		Status:      "completed",
		Steps:       stepResults,
		FinalResult: map[string]interface{}{"final": "result"},
		Duration:    10 * time.Second,
		CompletedAt: time.Now(),
	}

	assert.Equal(t, "test-session", result.SessionID)
	assert.Equal(t, "completed", result.Status)
	assert.Len(t, result.Steps, 2)
	assert.Equal(t, "step1", result.Steps[0].StepName)
	assert.Equal(t, "step2", result.Steps[1].StepName)
	assert.Equal(t, "result", result.FinalResult["final"])
	assert.Equal(t, 10*time.Second, result.Duration)
	assert.False(t, result.CompletedAt.IsZero())
}

// Benchmark tests
func BenchmarkPipelineContextFormatting(b *testing.B) {
	pipeline := &Pipeline{
		config: DefaultPipelineConfig(),
		logger: logrus.New(),
	}

	// Create test context documents
	docs := make([]elastic.SearchResult, 10)
	for i := 0; i < 10; i++ {
		docs[i] = elastic.SearchResult{
			Doc: elastic.Doc{
				ID:      fmt.Sprintf("doc%d", i),
				Topic:   "test topic",
				Section: "section",
				Text:    "This is test content for benchmarking",
			},
			Score:   float64(i) / 10.0,
			Snippet: "This is test content for benchmarking",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pipeline.formatContext(docs)
	}
}

func BenchmarkPipelineStepResultCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = PipelineStepResult{
			StepName:   "test-step",
			Status:     "completed",
			Output:     map[string]string{"result": "success"},
			Duration:   5 * time.Second,
			RetryCount: 0,
			Metadata:   map[string]interface{}{"tokens": 100},
		}
	}
}



