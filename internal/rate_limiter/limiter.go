package rate_limiter

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// Limiter represents a rate limiter for IP addresses
type Limiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	logger   *logrus.Logger
}

// NewLimiter creates a new rate limiter
func NewLimiter(requestsPerSecond float64, burst int) *Limiter {
	return &Limiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(requestsPerSecond),
		burst:    burst,
		logger:   logrus.New(),
	}
}

// GetLimiter returns or creates a limiter for the given IP
func (l *Limiter) GetLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	limiter, exists := l.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(l.rate, l.burst)
		l.limiters[ip] = limiter
		l.logger.WithFields(logrus.Fields{
			"ip":    ip,
			"rate":  l.rate,
			"burst": l.burst,
		}).Debug("Created new rate limiter for IP")
	}

	return limiter
}

// Allow checks if the request is allowed for the given IP
func (l *Limiter) Allow(ip string) bool {
	limiter := l.GetLimiter(ip)
	allowed := limiter.Allow()

	l.logger.WithFields(logrus.Fields{
		"ip":      ip,
		"allowed": allowed,
		"tokens":  limiter.Tokens(),
	}).Debug("Rate limit check")

	return allowed
}

// Wait waits for the limiter to allow the request
func (l *Limiter) Wait(ctx context.Context, ip string) error {
	limiter := l.GetLimiter(ip)
	return limiter.Wait(ctx)
}

// Cleanup removes old limiters to prevent memory leaks
func (l *Limiter) Cleanup(maxAge time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for ip, limiter := range l.limiters {
		// Simple cleanup based on token availability
		// In production, you might want to track last access time
		if limiter.Tokens() >= float64(l.burst) {
			delete(l.limiters, ip)
			l.logger.WithField("ip", ip).Debug("Cleaned up rate limiter")
		}
	}
}

// RateLimitMiddleware creates a Gin middleware for rate limiting
func RateLimitMiddleware(limiter *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client IP
		ip := c.ClientIP()

		// Check if request is allowed
		if !limiter.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"message":     "Too many requests from your IP. Please try again later.",
				"retry_after": 60, // seconds
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetClientIP extracts the real client IP from the request
func GetClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header first
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if idx := len(xff); idx > 0 {
			for i, char := range xff {
				if char == ',' {
					idx = i
					break
				}
			}
			return xff[:idx]
		}
	}

	// Check X-Real-IP header
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return c.ClientIP()
}
