package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/explainiq/agent/internal/adk"
	"github.com/explainiq/agent/internal/auth"
	"github.com/explainiq/agent/internal/llm"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// CriticService represents the critic service
type CriticService struct {
	geminiClient llm.GeminiClientInterface
	logger       *logrus.Logger
}

// NewCriticService creates a new critic service
func NewCriticService() *CriticService {
	geminiClient := llm.NewGeminiClient("")

	return &CriticService{
		geminiClient: geminiClient,
		logger:       logrus.New(),
	}
}

// ProcessTask processes a critique task
func (s *CriticService) ProcessTask(ctx context.Context, req adk.TaskRequest) (adk.TaskResponse, error) {
	s.logger.WithFields(logrus.Fields{
		"session_id": req.SessionID,
		"step":       req.Step,
		"topic":      req.Topic,
	}).Info("Processing critique task")

	// Extract lesson JSON from inputs
	lessonJSON, exists := req.Inputs["lesson"]
	if !exists || lessonJSON == "" {
		return adk.TaskResponse{}, fmt.Errorf("lesson JSON is required in inputs")
	}

	// Perform critique
	critiqueResponse, err := s.geminiClient.CritiqueLesson(ctx, lessonJSON)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"session_id": req.SessionID,
			"error":      err,
		}).Error("Lesson critique failed")
		return adk.TaskResponse{}, fmt.Errorf("lesson critique failed: %w", err)
	}

	// Convert critique to JSON strings
	critiqueJSON, err := json.Marshal(critiqueResponse.Issues)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"session_id": req.SessionID,
			"error":      err,
		}).Error("Failed to marshal critique issues")
		return adk.TaskResponse{}, fmt.Errorf("failed to marshal critique issues: %w", err)
	}

	patchPlanJSON, err := json.Marshal(critiqueResponse.PatchPlan)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"session_id": req.SessionID,
			"error":      err,
		}).Error("Failed to marshal patch plan")
		return adk.TaskResponse{}, fmt.Errorf("failed to marshal patch plan: %w", err)
	}

	// Create response
	response := adk.TaskResponse{
		Artifacts: map[string]string{
			"critique":   string(critiqueJSON),
			"patch_plan": string(patchPlanJSON),
		},
		Metrics: map[string]interface{}{
			"issues_count":     len(critiqueResponse.Issues),
			"patch_plan_count": len(critiqueResponse.PatchPlan),
			"critical_issues":  s.countIssuesBySeverity(critiqueResponse.Issues, "critical"),
			"high_issues":      s.countIssuesBySeverity(critiqueResponse.Issues, "high"),
			"medium_issues":    s.countIssuesBySeverity(critiqueResponse.Issues, "medium"),
			"low_issues":       s.countIssuesBySeverity(critiqueResponse.Issues, "low"),
		},
	}

	s.logger.WithFields(logrus.Fields{
		"session_id":       req.SessionID,
		"issues_count":     len(critiqueResponse.Issues),
		"patch_plan_count": len(critiqueResponse.PatchPlan),
	}).Info("Critique task completed successfully")

	return response, nil
}

// countIssuesBySeverity counts issues by severity level
func (s *CriticService) countIssuesBySeverity(issues []llm.CritiqueIssue, severity string) int {
	count := 0
	for _, issue := range issues {
		if issue.Severity == severity {
			count++
		}
	}
	return count
}

func main() {
	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)

	// Set Gin mode
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create critic service
	service := NewCriticService()

	// Create auth client
	serviceURL := os.Getenv("SERVICE_URL")
	if serviceURL == "" {
		serviceURL = "http://localhost:8083"
	}
	authClient := auth.NewClient(serviceURL)

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Health check endpoint (no auth required)
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "agent-critic",
			"timestamp": time.Now().UTC(),
		})
	})

	// Task processing endpoint (auth required)
	router.POST("/task", auth.ServiceAuthMiddleware(authClient), func(c *gin.Context) {
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
			log.WithFields(logrus.Fields{
				"session_id": req.SessionID,
				"error":      err,
			}).Error("Task processing failed")

			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Task processing failed",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, response)
	})

	// Legacy API endpoint for backward compatibility
	api := router.Group("/api/v1")
	{
		api.POST("/critique", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Use POST /task endpoint instead",
				"service": "agent-critic",
			})
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Infof("Agent Critic service starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down agent-critic service...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Info("Agent Critic service exited")
}
