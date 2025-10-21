package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AuthMiddleware creates a Gin middleware for JWT authentication
func AuthMiddleware(authClient *Client, required bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication for health checks and public endpoints
		if isPublicEndpoint(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			if required {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "Authorization header required",
					"message": "Missing Authorization header",
				})
				c.Abort()
				return
			}
			c.Next()
			return
		}

		// Check for Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			if required {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "Invalid authorization format",
					"message": "Authorization header must be 'Bearer <token>'",
				})
				c.Abort()
				return
			}
			c.Next()
			return
		}

		token := parts[1]

		// Validate the JWT token
		claims, err := authClient.ValidateGoogleJWT(c.Request.Context(), token)
		if err != nil {
			authClient.logger.WithFields(logrus.Fields{
				"error": err.Error(),
				"path":  c.Request.URL.Path,
			}).Warn("JWT validation failed")

			if required {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "Invalid token",
					"message": "JWT token validation failed",
				})
				c.Abort()
				return
			}
			c.Next()
			return
		}

		// Add claims to context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("claims", claims)

		authClient.logger.WithFields(logrus.Fields{
			"user_id": claims.UserID,
			"email":   claims.Email,
			"path":    c.Request.URL.Path,
		}).Debug("User authenticated successfully")

		c.Next()
	}
}

// ServiceAuthMiddleware creates middleware for service-to-service authentication
func ServiceAuthMiddleware(authClient *Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication for health checks
		if isPublicEndpoint(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Extract service token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Service authentication required",
				"message": "Missing Authorization header",
			})
			c.Abort()
			return
		}

		// Check for Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid authorization format",
				"message": "Authorization header must be 'Bearer <token>'",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate the service token
		claims, err := authClient.ValidateGoogleJWT(c.Request.Context(), token)
		if err != nil {
			authClient.logger.WithFields(logrus.Fields{
				"error": err.Error(),
				"path":  c.Request.URL.Path,
			}).Warn("Service token validation failed")

			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid service token",
				"message": "Service token validation failed",
			})
			c.Abort()
			return
		}

		// Add service claims to context
		c.Set("service_user_id", claims.UserID)
		c.Set("service_email", claims.Email)
		c.Set("service_claims", claims)

		authClient.logger.WithFields(logrus.Fields{
			"service_user_id": claims.UserID,
			"service_email":   claims.Email,
			"path":            c.Request.URL.Path,
		}).Debug("Service authenticated successfully")

		c.Next()
	}
}

// GetUserFromContext extracts user information from Gin context
func GetUserFromContext(c *gin.Context) (*Claims, bool) {
	claims, exists := c.Get("claims")
	if !exists {
		return nil, false
	}

	userClaims, ok := claims.(*Claims)
	return userClaims, ok
}

// GetServiceFromContext extracts service information from Gin context
func GetServiceFromContext(c *gin.Context) (*Claims, bool) {
	claims, exists := c.Get("service_claims")
	if !exists {
		return nil, false
	}

	serviceClaims, ok := claims.(*Claims)
	return serviceClaims, ok
}

// isPublicEndpoint checks if the endpoint should skip authentication
func isPublicEndpoint(path string) bool {
	publicPaths := []string{
		"/healthz",
		"/health",
		"/metrics",
		"/api/v1/health",
		"/api/v1/status",
	}

	for _, publicPath := range publicPaths {
		if strings.HasPrefix(path, publicPath) {
			return true
		}
	}

	return false
}

// RequireAuth is a helper function to check if user is authenticated
func RequireAuth(c *gin.Context) bool {
	_, exists := c.Get("claims")
	return exists
}

// RequireServiceAuth is a helper function to check if service is authenticated
func RequireServiceAuth(c *gin.Context) bool {
	_, exists := c.Get("service_claims")
	return exists
}
