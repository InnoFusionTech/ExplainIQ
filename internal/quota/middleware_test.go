package quota

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/InnoFusionTech/ExplainIQ/internal/cost_tracker"
	"github.com/InnoFusionTech/ExplainIQ/internal/rate_limiter"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockStorage is a mock implementation of storage.Storage
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockStorage) Set(ctx context.Context, key string, value []byte) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *MockStorage) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockStorage) List(ctx context.Context, prefix string) ([]string, error) {
	args := m.Called(ctx, prefix)
	return args.Get(0).([]string), args.Error(1)
}

func TestQuotaMiddleware_RateLimitExceeded(t *testing.T) {
	// Create rate limiter with very low limits
	rateLimiter := rate_limiter.NewLimiter(0.1, 1) // 0.1 requests per second, burst of 1

	// Create mock storage
	mockStorage := &MockStorage{}

	// Create cost tracker
	costTracker := cost_tracker.NewCostTracker(mockStorage)

	// Create quota manager
	quotaManager := NewQuotaManager(rateLimiter, costTracker)

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create router with quota middleware
	router := gin.New()
	router.Use(quotaManager.QuotaMiddleware())

	// Add test route
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// First request should succeed
	req1 := httptest.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Second request should be rate limited
	req2 := httptest.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusTooManyRequests, w2.Code)
}

func TestQuotaMiddleware_CostLimitExceeded(t *testing.T) {
	// Create rate limiter with high limits
	rateLimiter := rate_limiter.NewLimiter(100.0, 1000)

	// Create mock storage
	mockStorage := &MockStorage{}

	// Create cost tracker
	costTracker := cost_tracker.NewCostTracker(mockStorage)

	// Create quota manager with low cost limits
	quotaManager := NewQuotaManager(rateLimiter, costTracker)
	quotaManager.SetCostLimits(cost_tracker.CostLimits{
		MaxLLMCost:    0.01, // Very low limit
		MaxImageCost:  0.01,
		MaxTotalCost:  0.01,
		MaxLLMCalls:   1,
		MaxImageCalls: 1,
	})

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create router with quota middleware
	router := gin.New()
	router.Use(quotaManager.QuotaMiddleware())

	// Add test route
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Mock storage to return existing high costs
	mockStorage.On("Get", mock.Anything, "session_costs:test-session").Return([]byte(`{
		"session_id": "test-session",
		"total_llm_cost": 0.02,
		"total_image_cost": 0.0,
		"total_cost": 0.02,
		"llm_calls": 2,
		"image_calls": 0,
		"last_updated": "2023-01-01T00:00:00Z",
		"created_at": "2023-01-01T00:00:00Z"
	}`), nil)

	// Request should be cost limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Session-ID", "test-session")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 429 due to cost limit
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestQuotaManager_TrackLLMCall(t *testing.T) {
	// Create rate limiter
	rateLimiter := rate_limiter.NewLimiter(100.0, 1000)

	// Create mock storage
	mockStorage := &MockStorage{}

	// Create cost tracker
	costTracker := cost_tracker.NewCostTracker(mockStorage)

	// Create quota manager
	quotaManager := NewQuotaManager(rateLimiter, costTracker)

	// Mock storage calls
	mockStorage.On("Get", mock.Anything, "session_costs:test-session").Return([]byte(`{
		"session_id": "test-session",
		"total_llm_cost": 0.0,
		"total_image_cost": 0.0,
		"total_cost": 0.0,
		"llm_calls": 0,
		"image_calls": 0,
		"last_updated": "2023-01-01T00:00:00Z",
		"created_at": "2023-01-01T00:00:00Z"
	}`), nil)

	mockStorage.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).Return(nil)

	// Track LLM call
	err := quotaManager.TrackLLMCall(context.Background(), "test-session", "user123", "127.0.0.1", "gemini-pro", 100, 50)

	assert.NoError(t, err)
	mockStorage.AssertExpectations(t)
}

func TestQuotaManager_TrackImageCall(t *testing.T) {
	// Create rate limiter
	rateLimiter := rate_limiter.NewLimiter(100.0, 1000)

	// Create mock storage
	mockStorage := &MockStorage{}

	// Create cost tracker
	costTracker := cost_tracker.NewCostTracker(mockStorage)

	// Create quota manager
	quotaManager := NewQuotaManager(rateLimiter, costTracker)

	// Mock storage calls
	mockStorage.On("Get", mock.Anything, "session_costs:test-session").Return([]byte(`{
		"session_id": "test-session",
		"total_llm_cost": 0.0,
		"total_image_cost": 0.0,
		"total_cost": 0.0,
		"llm_calls": 0,
		"image_calls": 0,
		"last_updated": "2023-01-01T00:00:00Z",
		"created_at": "2023-01-01T00:00:00Z"
	}`), nil)

	mockStorage.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).Return(nil)

	// Track image call
	err := quotaManager.TrackImageCall(context.Background(), "test-session", "user123", "127.0.0.1", 2)

	assert.NoError(t, err)
	mockStorage.AssertExpectations(t)
}

func TestQuotaManager_GetQuotaInfo(t *testing.T) {
	// Create rate limiter
	rateLimiter := rate_limiter.NewLimiter(100.0, 1000)

	// Create mock storage
	mockStorage := &MockStorage{}

	// Create cost tracker
	costTracker := cost_tracker.NewCostTracker(mockStorage)

	// Create quota manager
	quotaManager := NewQuotaManager(rateLimiter, costTracker)

	// Mock storage to return session costs
	mockStorage.On("Get", mock.Anything, "session_costs:test-session").Return([]byte(`{
		"session_id": "test-session",
		"total_llm_cost": 0.05,
		"total_image_cost": 0.02,
		"total_cost": 0.07,
		"llm_calls": 5,
		"image_calls": 1,
		"last_updated": "2023-01-01T00:00:00Z",
		"created_at": "2023-01-01T00:00:00Z"
	}`), nil)

	// Get quota info
	quotaInfo, err := quotaManager.GetQuotaInfo(context.Background(), "test-session")

	assert.NoError(t, err)
	assert.NotNil(t, quotaInfo)
	assert.Equal(t, "test-session", quotaInfo["session_id"])

	// Check that quota remaining is calculated correctly
	quotaRemaining := quotaInfo["quota_remaining"].(map[string]interface{})
	assert.Equal(t, 9.95, quotaRemaining["llm_cost_remaining"])    // 10.0 - 0.05
	assert.Equal(t, 4.98, quotaRemaining["image_cost_remaining"])  // 5.0 - 0.02
	assert.Equal(t, 14.93, quotaRemaining["total_cost_remaining"]) // 15.0 - 0.07

	mockStorage.AssertExpectations(t)
}

func TestQuotaManager_Cleanup(t *testing.T) {
	// Create rate limiter
	rateLimiter := rate_limiter.NewLimiter(100.0, 1000)

	// Create mock storage
	mockStorage := &MockStorage{}

	// Create cost tracker
	costTracker := cost_tracker.NewCostTracker(mockStorage)

	// Create quota manager
	quotaManager := NewQuotaManager(rateLimiter, costTracker)

	// Test cleanup (should not panic)
	quotaManager.Cleanup()
}



