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
	"github.com/explainiq/agent/internal/config"
	"github.com/explainiq/agent/internal/cost_tracker"
	"github.com/explainiq/agent/internal/llm"
	"github.com/explainiq/agent/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SummarizerService represents the summarizer service
type SummarizerService struct {
	geminiClient llm.GeminiClientInterface
	costTracker  *cost_tracker.CostTracker
	logger       *logrus.Logger
}

// NewSummarizerService creates a new summarizer service
func NewSummarizerService() *SummarizerService {
	geminiClient := llm.NewGeminiClient("")

	// Create storage client for cost tracking
	storageClient, err := storage.NewFirestoreClient(context.Background(), os.Getenv("GCP_PROJECT_ID"))
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create storage client")
	}

	// Create cost tracker
	costTracker := cost_tracker.NewCostTracker(storageClient)

	return &SummarizerService{
		geminiClient: geminiClient,
		costTracker:  costTracker,
		logger:       logrus.New(),
	}
}

// ProcessTask processes a summarization task
func (s *SummarizerService) ProcessTask(ctx context.Context, req adk.TaskRequest) (adk.TaskResponse, error) {
	s.logger.WithFields(logrus.Fields{
		"session_id": req.SessionID,
		"step":       req.Step,
		"topic":      req.Topic,
	}).Info("Processing summarization task")

	// Extract topic and context from inputs
	topic, exists := req.Inputs["topic"]
	if !exists || topic == "" {
		return adk.TaskResponse{}, fmt.Errorf("topic is required in inputs")
	}

	context, exists := req.Inputs["context"]
	if !exists {
		context = "" // Context is optional
	}

	// Perform summarization
	result, err := s.geminiClient.Summarize(ctx, topic, context)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"session_id": req.SessionID,
			"error":      err,
		}).Error("Summarization failed")
		return adk.TaskResponse{}, fmt.Errorf("summarization failed: %w", err)
	}

	// Track LLM call cost (estimate tokens)
	inputTokens := len(topic) + len(context)                                                     // Rough estimate
	outputTokens := len(result.Outline) + len(result.Prerequisites) + len(result.Misconceptions) // Rough estimate

	// Get client IP from context if available
	ipAddress := ""
	if ginCtx, ok := ctx.Value("gin_context").(*gin.Context); ok {
		ipAddress = ginCtx.ClientIP()
	}

	// Track the cost
	if err := s.costTracker.TrackLLMCall(ctx, req.SessionID, "", ipAddress, "gemini-pro", inputTokens, outputTokens); err != nil {
		s.logger.WithFields(logrus.Fields{
			"session_id": req.SessionID,
			"error":      err,
		}).Warn("Failed to track LLM call cost")
	}

	// Convert result to artifacts
	artifacts := make(map[string]string)

	// Convert arrays to JSON strings
	if outlineJSON, err := json.Marshal(result.Outline); err == nil {
		artifacts["outline"] = string(outlineJSON)
	}

	if prereqJSON, err := json.Marshal(result.Prerequisites); err == nil {
		artifacts["prerequisites"] = string(prereqJSON)
	}

	if misconJSON, err := json.Marshal(result.Misconceptions); err == nil {
		artifacts["misconceptions"] = string(misconJSON)
	}

	if citationsJSON, err := json.Marshal(result.Citations); err == nil {
		artifacts["citations"] = string(citationsJSON)
	}

	// Create response
	response := adk.TaskResponse{
		Artifacts: artifacts,
		Metrics: map[string]interface{}{
			"outline_count":        len(result.Outline),
			"prerequisites_count":  len(result.Prerequisites),
			"misconceptions_count": len(result.Misconceptions),
			"citations_count":      len(result.Citations),
		},
	}

	s.logger.WithFields(logrus.Fields{
		"session_id":     req.SessionID,
		"outline_count":  len(result.Outline),
		"prereq_count":   len(result.Prerequisites),
		"miscon_count":   len(result.Misconceptions),
		"citation_count": len(result.Citations),
	}).Info("Summarization task completed successfully")

	return response, nil
}

func main() {
	// Load environment variables from .env file
	if err := config.LoadEnvFiles(); err != nil {
		logrus.Warnf("Failed to load .env file: %v", err)
	}

	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)

	// Set Gin mode
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create summarizer service
	service := NewSummarizerService()

	// Create auth client
	serviceURL := os.Getenv("SERVICE_URL")
	if serviceURL == "" {
		serviceURL = "http://localhost:8081"
	}
	authClient := auth.NewClient(serviceURL)

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Health check endpoint (no auth required)
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "agent-summarizer",
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
		api.POST("/summarize", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Use POST /task endpoint instead",
				"service": "agent-summarizer",
			})
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Infof("Agent Summarizer service starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down agent-summarizer service...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Info("Agent Summarizer service exited")
}
