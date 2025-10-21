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

// ExplainerService represents the explainer service
type ExplainerService struct {
	geminiClient llm.GeminiClientInterface
	logger       *logrus.Logger
}

// NewExplainerService creates a new explainer service
func NewExplainerService() *ExplainerService {
	geminiClient := llm.NewGeminiClient("")

	return &ExplainerService{
		geminiClient: geminiClient,
		logger:       logrus.New(),
	}
}

// ProcessTask processes an explanation task
func (s *ExplainerService) ProcessTask(ctx context.Context, req adk.TaskRequest) (adk.TaskResponse, error) {
	s.logger.WithFields(logrus.Fields{
		"session_id": req.SessionID,
		"step":       req.Step,
		"topic":      req.Topic,
	}).Info("Processing explanation task")

	// Extract required inputs
	topic, exists := req.Inputs["topic"]
	if !exists || topic == "" {
		return adk.TaskResponse{}, fmt.Errorf("topic is required in inputs")
	}

	outline, exists := req.Inputs["outline"]
	if !exists {
		outline = "" // Outline is optional
	}

	misconceptions, exists := req.Inputs["misconceptions"]
	if !exists {
		misconceptions = "" // Misconceptions are optional
	}

	context, exists := req.Inputs["context"]
	if !exists {
		context = "" // Context is optional
	}

	// Generate OG lesson
	ogLesson, err := s.geminiClient.ExplainWithOG(ctx, topic, outline, misconceptions, context)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"session_id": req.SessionID,
			"error":      err,
		}).Error("OG lesson generation failed")
		return adk.TaskResponse{}, fmt.Errorf("OG lesson generation failed: %w", err)
	}

	// Convert OGLesson to JSON string
	lessonJSON, err := json.Marshal(ogLesson)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"session_id": req.SessionID,
			"error":      err,
		}).Error("Failed to marshal OG lesson")
		return adk.TaskResponse{}, fmt.Errorf("failed to marshal OG lesson: %w", err)
	}

	// Create response
	response := adk.TaskResponse{
		Artifacts: map[string]string{
			"lesson": string(lessonJSON),
		},
		Metrics: map[string]interface{}{
			"big_picture_length":      len(ogLesson.BigPicture),
			"metaphor_length":         len(ogLesson.Metaphor),
			"core_mechanism_length":   len(ogLesson.CoreMechanism),
			"toy_example_code_length": len(ogLesson.ToyExampleCode),
			"memory_hook_length":      len(ogLesson.MemoryHook),
			"real_life_length":        len(ogLesson.RealLife),
			"best_practices_length":   len(ogLesson.BestPractices),
		},
	}

	s.logger.WithFields(logrus.Fields{
		"session_id": req.SessionID,
		"topic":      topic,
	}).Info("Explanation task completed successfully")

	return response, nil
}

func main() {
	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)

	// Set Gin mode
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create explainer service
	service := NewExplainerService()

	// Create auth client
	serviceURL := os.Getenv("SERVICE_URL")
	if serviceURL == "" {
		serviceURL = "http://localhost:8082"
	}
	authClient := auth.NewClient(serviceURL)

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Health check endpoint (no auth required)
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "agent-explainer",
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
		api.POST("/explain", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Use POST /task endpoint instead",
				"service": "agent-explainer",
			})
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Infof("Agent Explainer service starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down agent-explainer service...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Info("Agent Explainer service exited")
}
