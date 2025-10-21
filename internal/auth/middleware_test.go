package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware_NoAuthRequired(t *testing.T) {
	// Create auth client
	authClient := NewClient("http://localhost:8080")

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create router with middleware
	router := gin.New()
	router.Use(AuthMiddleware(authClient, false))

	// Add test route
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Test request without auth
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_AuthRequired_NoToken(t *testing.T) {
	// Create auth client
	authClient := NewClient("http://localhost:8080")

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create router with middleware
	router := gin.New()
	router.Use(AuthMiddleware(authClient, true))

	// Add test route
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Test request without auth
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_AuthRequired_InvalidToken(t *testing.T) {
	// Create auth client
	authClient := NewClient("http://localhost:8080")

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create router with middleware
	router := gin.New()
	router.Use(AuthMiddleware(authClient, true))

	// Add test route
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Test request with invalid token
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestServiceAuthMiddleware_NoToken(t *testing.T) {
	// Create auth client
	authClient := NewClient("http://localhost:8080")

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create router with middleware
	router := gin.New()
	router.Use(ServiceAuthMiddleware(authClient))

	// Add test route
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Test request without auth
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIsPublicEndpoint(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/healthz", true},
		{"/health", true},
		{"/metrics", true},
		{"/api/v1/health", true},
		{"/api/v1/status", true},
		{"/api/v1/sessions", false},
		{"/test", false},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			result := isPublicEndpoint(test.path)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestGetUserFromContext(t *testing.T) {
	// Create auth client
	authClient := NewClient("http://localhost:8080")

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create router with middleware
	router := gin.New()
	router.Use(AuthMiddleware(authClient, false))

	// Add test route that checks context
	router.GET("/test", func(c *gin.Context) {
		claims, exists := GetUserFromContext(c)
		if exists {
			c.JSON(http.StatusOK, gin.H{"user_id": claims.UserID})
		} else {
			c.JSON(http.StatusOK, gin.H{"user_id": nil})
		}
	})

	// Test request without auth
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireAuth(t *testing.T) {
	// Create auth client
	authClient := NewClient("http://localhost:8080")

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create router with middleware
	router := gin.New()
	router.Use(AuthMiddleware(authClient, false))

	// Add test route that checks auth
	router.GET("/test", func(c *gin.Context) {
		authenticated := RequireAuth(c)
		c.JSON(http.StatusOK, gin.H{"authenticated": authenticated})
	})

	// Test request without auth
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}



