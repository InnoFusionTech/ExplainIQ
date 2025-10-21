package quota

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/creduntvitam/explainiq/internal/cost_tracker"
	"github.com/creduntvitam/explainiq/internal/rate_limiter"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// QuotaManager manages rate limiting and cost tracking
type QuotaManager struct {
	RateLimiter *rate_limiter.Limiter
	costTracker *cost_tracker.CostTracker
	costLimits  cost_tracker.CostLimits
	logger      *logrus.Logger
}

// NewQuotaManager creates a new quota manager
func NewQuotaManager(rateLimiter *rate_limiter.Limiter, costTracker *cost_tracker.CostTracker) *QuotaManager {
	return &QuotaManager{
		RateLimiter: rateLimiter,
		costTracker: costTracker,
		costLimits:  cost_tracker.DefaultCostLimits(),
		logger:      logrus.New(),
	}
}

// SetCostLimits sets custom cost limits
func (qm *QuotaManager) SetCostLimits(limits cost_tracker.CostLimits) {
	qm.costLimits = limits
}

// QuotaMiddleware creates a Gin middleware for quota management
func (qm *QuotaManager) QuotaMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client IP
		ip := rate_limiter.GetClientIP(c)

		// Check rate limit
		if !qm.RateLimiter.Allow(ip) {
			qm.logger.WithFields(logrus.Fields{
				"ip":     ip,
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
			}).Warn("Rate limit exceeded")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"message":     "Too many requests from your IP. Please try again later.",
				"retry_after": 60,
				"quota_type":  "rate_limit",
			})
			c.Abort()
			return
		}

		// Get session ID from context or create one
		sessionID := c.GetString("session_id")
		if sessionID == "" {
			// For requests without session ID, we'll track by IP
			sessionID = fmt.Sprintf("ip:%s", ip)
		}

		// Check cost limits
		costs, exceeded, err := qm.costTracker.CheckCostLimits(c.Request.Context(), sessionID, qm.costLimits)
		if err != nil {
			qm.logger.WithFields(logrus.Fields{
				"session_id": sessionID,
				"error":      err,
			}).Error("Failed to check cost limits")
			// Continue without cost checking if there's an error
		} else if exceeded {
			qm.logger.WithFields(logrus.Fields{
				"session_id":     sessionID,
				"total_cost":     costs.TotalCost,
				"max_cost":       qm.costLimits.MaxTotalCost,
				"llm_cost":       costs.TotalLLMCost,
				"max_llm_cost":   qm.costLimits.MaxLLMCost,
				"image_cost":     costs.TotalImageCost,
				"max_image_cost": qm.costLimits.MaxImageCost,
			}).Warn("Cost limit exceeded")

			// Get remaining quota for response
			quotaRemaining, _ := qm.costTracker.GetQuotaRemaining(c.Request.Context(), sessionID, qm.costLimits)

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":           "Cost limit exceeded",
				"message":         "You have exceeded your usage quota. Please try again later or contact support.",
				"quota_type":      "cost_limit",
				"quota_remaining": quotaRemaining,
				"current_costs": gin.H{
					"total_cost":  costs.TotalCost,
					"llm_cost":    costs.TotalLLMCost,
					"image_cost":  costs.TotalImageCost,
					"llm_calls":   costs.LLMCalls,
					"image_calls": costs.ImageCalls,
				},
			})
			c.Abort()
			return
		}

		// Add quota information to context
		if costs != nil {
			quotaRemaining, _ := qm.costTracker.GetQuotaRemaining(c.Request.Context(), sessionID, qm.costLimits)
			c.Set("quota_remaining", quotaRemaining)
			c.Set("current_costs", costs)
		}

		c.Next()
	}
}

// TrackLLMCall tracks an LLM call and checks limits
func (qm *QuotaManager) TrackLLMCall(ctx context.Context, sessionID, userID, ipAddress, model string, inputTokens, outputTokens int) error {
	// Track the cost
	if err := qm.costTracker.TrackLLMCall(ctx, sessionID, userID, ipAddress, model, inputTokens, outputTokens); err != nil {
		return fmt.Errorf("failed to track LLM call: %w", err)
	}

	// Check if limits are exceeded after tracking
	costs, exceeded, err := qm.costTracker.CheckCostLimits(ctx, sessionID, qm.costLimits)
	if err != nil {
		return fmt.Errorf("failed to check cost limits: %w", err)
	}

	if exceeded {
		return fmt.Errorf("cost limit exceeded after LLM call: total_cost=%.4f, max_cost=%.4f",
			costs.TotalCost, qm.costLimits.MaxTotalCost)
	}

	return nil
}

// TrackImageCall tracks an image generation call and checks limits
func (qm *QuotaManager) TrackImageCall(ctx context.Context, sessionID, userID, ipAddress string, imageCount int) error {
	// Track the cost
	if err := qm.costTracker.TrackImageCall(ctx, sessionID, userID, ipAddress, imageCount); err != nil {
		return fmt.Errorf("failed to track image call: %w", err)
	}

	// Check if limits are exceeded after tracking
	costs, exceeded, err := qm.costTracker.CheckCostLimits(ctx, sessionID, qm.costLimits)
	if err != nil {
		return fmt.Errorf("failed to check cost limits: %w", err)
	}

	if exceeded {
		return fmt.Errorf("cost limit exceeded after image call: total_cost=%.4f, max_cost=%.4f",
			costs.TotalCost, qm.costLimits.MaxTotalCost)
	}

	return nil
}

// GetQuotaInfo returns quota information for a session
func (qm *QuotaManager) GetQuotaInfo(ctx context.Context, sessionID string) (map[string]interface{}, error) {
	costs, err := qm.costTracker.GetSessionCosts(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	quotaRemaining, err := qm.costTracker.GetQuotaRemaining(ctx, sessionID, qm.costLimits)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"session_id":      sessionID,
		"current_costs":   costs,
		"quota_remaining": quotaRemaining,
		"limits":          qm.costLimits,
	}, nil
}

// QuotaInfoMiddleware adds quota information to SSE metadata
func (qm *QuotaManager) QuotaInfoMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get session ID
		sessionID := c.GetString("session_id")
		if sessionID == "" {
			ip := rate_limiter.GetClientIP(c)
			sessionID = fmt.Sprintf("ip:%s", ip)
		}

		// Get quota information
		quotaInfo, err := qm.GetQuotaInfo(c.Request.Context(), sessionID)
		if err != nil {
			qm.logger.WithFields(logrus.Fields{
				"session_id": sessionID,
				"error":      err,
			}).Error("Failed to get quota info")
		} else {
			// Add quota info to context for SSE metadata
			c.Set("quota_info", quotaInfo)
		}

		c.Next()
	}
}

// Cleanup performs periodic cleanup of rate limiters
func (qm *QuotaManager) Cleanup() {
	qm.RateLimiter.Cleanup(1 * time.Hour)
}
