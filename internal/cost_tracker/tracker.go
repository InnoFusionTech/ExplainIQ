package cost_tracker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/InnoFusionTech/ExplainIQ/internal/storage"
	"github.com/sirupsen/logrus"
)

// CostTracker tracks costs for sessions
type CostTracker struct {
	storage storage.Storage
	logger  *logrus.Logger
}

// CostEntry represents a cost entry for a session
type CostEntry struct {
	SessionID     string                 `json:"session_id"`
	UserID        string                 `json:"user_id,omitempty"`
	IPAddress     string                 `json:"ip_address,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
	Operation     string                 `json:"operation"` // "llm_call", "imagen_call", etc.
	Model         string                 `json:"model,omitempty"`
	InputTokens   int                    `json:"input_tokens,omitempty"`
	OutputTokens  int                    `json:"output_tokens,omitempty"`
	Images        int                    `json:"images,omitempty"`
	EstimatedCost float64                `json:"estimated_cost"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// SessionCosts represents the total costs for a session
type SessionCosts struct {
	SessionID      string    `json:"session_id"`
	UserID         string    `json:"user_id,omitempty"`
	IPAddress      string    `json:"ip_address,omitempty"`
	TotalLLMCost   float64   `json:"total_llm_cost"`
	TotalImageCost float64   `json:"total_image_cost"`
	TotalCost      float64   `json:"total_cost"`
	LLMCalls       int       `json:"llm_calls"`
	ImageCalls     int       `json:"image_calls"`
	LastUpdated    time.Time `json:"last_updated"`
	CreatedAt      time.Time `json:"created_at"`
}

// CostLimits represents the cost limits for a session
type CostLimits struct {
	MaxLLMCost    float64 `json:"max_llm_cost"`
	MaxImageCost  float64 `json:"max_image_cost"`
	MaxTotalCost  float64 `json:"max_total_cost"`
	MaxLLMCalls   int     `json:"max_llm_calls"`
	MaxImageCalls int     `json:"max_image_calls"`
}

// DefaultCostLimits returns the default cost limits
func DefaultCostLimits() CostLimits {
	return CostLimits{
		MaxLLMCost:    10.0, // $10 max for LLM calls
		MaxImageCost:  5.0,  // $5 max for image generation
		MaxTotalCost:  15.0, // $15 max total cost
		MaxLLMCalls:   100,  // 100 LLM calls max
		MaxImageCalls: 50,   // 50 image calls max
	}
}

// NewCostTracker creates a new cost tracker
func NewCostTracker(storage storage.Storage) *CostTracker {
	return &CostTracker{
		storage: storage,
		logger:  logrus.New(),
	}
}

// TrackLLMCall tracks an LLM call cost
func (ct *CostTracker) TrackLLMCall(ctx context.Context, sessionID, userID, ipAddress, model string, inputTokens, outputTokens int) error {
	// Estimate cost based on model and tokens
	cost := ct.estimateLLMCost(model, inputTokens, outputTokens)

	entry := CostEntry{
		SessionID:     sessionID,
		UserID:        userID,
		IPAddress:     ipAddress,
		Timestamp:     time.Now(),
		Operation:     "llm_call",
		Model:         model,
		InputTokens:   inputTokens,
		OutputTokens:  outputTokens,
		EstimatedCost: cost,
		Metadata: map[string]interface{}{
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
		},
	}

	return ct.recordCostEntry(ctx, entry)
}

// TrackImageCall tracks an image generation call cost
func (ct *CostTracker) TrackImageCall(ctx context.Context, sessionID, userID, ipAddress string, imageCount int) error {
	// Estimate cost based on image count
	cost := ct.estimateImageCost(imageCount)

	entry := CostEntry{
		SessionID:     sessionID,
		UserID:        userID,
		IPAddress:     ipAddress,
		Timestamp:     time.Now(),
		Operation:     "imagen_call",
		Images:        imageCount,
		EstimatedCost: cost,
		Metadata: map[string]interface{}{
			"image_count": imageCount,
		},
	}

	return ct.recordCostEntry(ctx, entry)
}

// GetSessionCosts retrieves the total costs for a session
func (ct *CostTracker) GetSessionCosts(ctx context.Context, sessionID string) (*SessionCosts, error) {
	key := fmt.Sprintf("session_costs:%s", sessionID)

	data, err := ct.storage.Get(ctx, key)
	if err != nil {
		// Return zero costs if not found
		return &SessionCosts{
			SessionID:      sessionID,
			TotalLLMCost:   0,
			TotalImageCost: 0,
			TotalCost:      0,
			LLMCalls:       0,
			ImageCalls:     0,
			LastUpdated:    time.Now(),
			CreatedAt:      time.Now(),
		}, nil
	}

	var costs SessionCosts
	if err := json.Unmarshal(data, &costs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session costs: %w", err)
	}

	return &costs, nil
}

// CheckCostLimits checks if the session has exceeded cost limits
func (ct *CostTracker) CheckCostLimits(ctx context.Context, sessionID string, limits CostLimits) (*SessionCosts, bool, error) {
	costs, err := ct.GetSessionCosts(ctx, sessionID)
	if err != nil {
		return nil, false, err
	}

	// Check if any limits are exceeded
	exceeded := costs.TotalLLMCost > limits.MaxLLMCost ||
		costs.TotalImageCost > limits.MaxImageCost ||
		costs.TotalCost > limits.MaxTotalCost ||
		costs.LLMCalls > limits.MaxLLMCalls ||
		costs.ImageCalls > limits.MaxImageCalls

	return costs, exceeded, nil
}

// recordCostEntry records a cost entry and updates session totals
func (ct *CostTracker) recordCostEntry(ctx context.Context, entry CostEntry) error {
	// Store the individual cost entry
	entryKey := fmt.Sprintf("cost_entry:%s:%d", entry.SessionID, entry.Timestamp.UnixNano())
	entryData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cost entry: %w", err)
	}

	if err := ct.storage.Set(ctx, entryKey, entryData); err != nil {
		return fmt.Errorf("failed to store cost entry: %w", err)
	}

	// Update session totals
	return ct.updateSessionCosts(ctx, entry)
}

// updateSessionCosts updates the total costs for a session
func (ct *CostTracker) updateSessionCosts(ctx context.Context, entry CostEntry) error {
	key := fmt.Sprintf("session_costs:%s", entry.SessionID)

	// Get existing costs
	costs, err := ct.GetSessionCosts(ctx, entry.SessionID)
	if err != nil {
		return err
	}

	// Update costs based on operation
	switch entry.Operation {
	case "llm_call":
		costs.TotalLLMCost += entry.EstimatedCost
		costs.LLMCalls++
	case "imagen_call":
		costs.TotalImageCost += entry.EstimatedCost
		costs.ImageCalls++
	}

	// Update totals
	costs.TotalCost = costs.TotalLLMCost + costs.TotalImageCost
	costs.LastUpdated = time.Now()

	// Set user ID and IP if not already set
	if costs.UserID == "" {
		costs.UserID = entry.UserID
	}
	if costs.IPAddress == "" {
		costs.IPAddress = entry.IPAddress
	}

	// Store updated costs
	costsData, err := json.Marshal(costs)
	if err != nil {
		return fmt.Errorf("failed to marshal session costs: %w", err)
	}

	return ct.storage.Set(ctx, key, costsData)
}

// estimateLLMCost estimates the cost of an LLM call
func (ct *CostTracker) estimateLLMCost(model string, inputTokens, outputTokens int) float64 {
	// Simplified cost estimation (in USD)
	// In production, you'd use actual pricing from the LLM provider

	var inputCostPer1K, outputCostPer1K float64

	switch model {
	case "gemini-pro":
		inputCostPer1K = 0.0005  // $0.50 per 1M tokens
		outputCostPer1K = 0.0015 // $1.50 per 1M tokens
	case "gemini-pro-vision":
		inputCostPer1K = 0.0005
		outputCostPer1K = 0.0015
	default:
		inputCostPer1K = 0.001 // Default pricing
		outputCostPer1K = 0.002
	}

	inputCost := float64(inputTokens) / 1000.0 * inputCostPer1K
	outputCost := float64(outputTokens) / 1000.0 * outputCostPer1K

	return inputCost + outputCost
}

// estimateImageCost estimates the cost of image generation
func (ct *CostTracker) estimateImageCost(imageCount int) float64 {
	// Simplified cost estimation for Imagen (in USD)
	// In production, you'd use actual pricing from the image generation service

	costPerImage := 0.02 // $0.02 per image
	return float64(imageCount) * costPerImage
}

// GetQuotaRemaining calculates the remaining quota for a session
func (ct *CostTracker) GetQuotaRemaining(ctx context.Context, sessionID string, limits CostLimits) (map[string]interface{}, error) {
	costs, err := ct.GetSessionCosts(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"llm_cost_remaining":    limits.MaxLLMCost - costs.TotalLLMCost,
		"image_cost_remaining":  limits.MaxImageCost - costs.TotalImageCost,
		"total_cost_remaining":  limits.MaxTotalCost - costs.TotalCost,
		"llm_calls_remaining":   limits.MaxLLMCalls - costs.LLMCalls,
		"image_calls_remaining": limits.MaxImageCalls - costs.ImageCalls,
		"total_llm_cost":        costs.TotalLLMCost,
		"total_image_cost":      costs.TotalImageCost,
		"total_cost":            costs.TotalCost,
		"llm_calls":             costs.LLMCalls,
		"image_calls":           costs.ImageCalls,
	}, nil
}



