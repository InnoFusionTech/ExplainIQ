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

// VisualizerService represents the visualizer service
type VisualizerService struct {
	geminiClient llm.GeminiClientInterface
	logger       *logrus.Logger
}

// NewVisualizerService creates a new visualizer service
func NewVisualizerService() *VisualizerService {
	geminiClient := llm.NewGeminiClient("")

	return &VisualizerService{
		geminiClient: geminiClient,
		logger:       logrus.New(),
	}
}

// ProcessTask processes a visualization task
func (s *VisualizerService) ProcessTask(ctx context.Context, req adk.TaskRequest) (adk.TaskResponse, error) {
	s.logger.WithFields(logrus.Fields{
		"session_id": req.SessionID,
		"step":       req.Step,
		"topic":      req.Topic,
	}).Info("Processing visualization task")

	// Extract lesson JSON from inputs
	lessonJSON, exists := req.Inputs["lesson"]
	if !exists || lessonJSON == "" {
		return adk.TaskResponse{}, fmt.Errorf("lesson JSON is required in inputs")
	}

	// Generate visualizations
	visualizeResponse, err := s.geminiClient.VisualizeCore(ctx, lessonJSON, req.SessionID)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"session_id": req.SessionID,
			"error":      err,
		}).Error("Visualization generation failed")
		return adk.TaskResponse{}, fmt.Errorf("visualization generation failed: %w", err)
	}

	// Convert images and captions to JSON strings
	imagesJSON, err := json.Marshal(visualizeResponse.Images)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"session_id": req.SessionID,
			"error":      err,
		}).Error("Failed to marshal images")
		return adk.TaskResponse{}, fmt.Errorf("failed to marshal images: %w", err)
	}

	captionsJSON, err := json.Marshal(visualizeResponse.Captions)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"session_id": req.SessionID,
			"error":      err,
		}).Error("Failed to marshal captions")
		return adk.TaskResponse{}, fmt.Errorf("failed to marshal captions: %w", err)
	}

	// Create response
	response := adk.TaskResponse{
		Artifacts: map[string]string{
			"images":   string(imagesJSON),
			"captions": string(captionsJSON),
		},
		Metrics: map[string]interface{}{
			"images_count":   len(visualizeResponse.Images),
			"captions_count": len(visualizeResponse.Captions),
		},
	}

	s.logger.WithFields(logrus.Fields{
		"session_id":     req.SessionID,
		"images_count":   len(visualizeResponse.Images),
		"captions_count": len(visualizeResponse.Captions),
	}).Info("Visualization task completed successfully")

	return response, nil
}

func main() {
	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)

	// Set Gin mode
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create visualizer service
	service := NewVisualizerService()

	// Create auth client
	serviceURL := os.Getenv("SERVICE_URL")
	if serviceURL == "" {
		serviceURL = "http://localhost:8084"
	}
	authClient := auth.NewClient(serviceURL)

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Health check endpoint (no auth required)
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "agent-visualizer",
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
		api.POST("/visualize", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Use POST /task endpoint instead",
				"service": "agent-visualizer",
			})
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Infof("Agent Visualizer service starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down agent-visualizer service...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Info("Agent Visualizer service exited")
}
