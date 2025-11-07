package agent

import (
	"context"
	"net/http"
	"time"

	"github.com/InnoFusionTech/ExplainIQ/internal/adk"
	"github.com/InnoFusionTech/ExplainIQ/internal/auth"
	"github.com/InnoFusionTech/ExplainIQ/internal/config"
	"github.com/InnoFusionTech/ExplainIQ/internal/constants"
	"github.com/InnoFusionTech/ExplainIQ/internal/logger"
	"github.com/InnoFusionTech/ExplainIQ/internal/server"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// TaskProcessor defines the interface for processing tasks
type TaskProcessor interface {
	ProcessTask(ctx context.Context, req adk.TaskRequest) (adk.TaskResponse, error)
}

// ServiceConfig holds configuration for agent services
type ServiceConfig struct {
	ServiceName   string
	DefaultPort   string
	DefaultURL    string
	Processor     TaskProcessor
	RequireAuth   bool
}

// StartAgentService starts an agent service with the given configuration
func StartAgentService(cfg ServiceConfig) error {
	// Load configuration
	appConfig := config.LoadForService(cfg.DefaultPort, cfg.DefaultURL)
	
	// Create logger
	log := logger.New(logger.Config{
		Level: appConfig.LogLevel,
	})

	// Set Gin mode
	if appConfig.GinMode != "" {
		gin.SetMode(appConfig.GinMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create auth client
	authClient := auth.NewClient(appConfig.ServiceURL)

	// Setup router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Health check endpoint (no auth required)
	router.GET(constants.EndpointHealthz, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   cfg.ServiceName,
			"timestamp": time.Now().UTC(),
		})
	})

	// Task processing endpoint
	taskHandler := func(c *gin.Context) {
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
		response, err := cfg.Processor.ProcessTask(c.Request.Context(), req)
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
	}

	// Apply auth middleware if required
	if cfg.RequireAuth {
		router.POST(constants.EndpointTask, auth.ServiceAuthMiddleware(authClient), taskHandler)
	} else {
		router.POST(constants.EndpointTask, taskHandler)
	}

	// Legacy API endpoint for backward compatibility
	api := router.Group("/api/v1")
	{
		api.POST("/"+cfg.ServiceName, func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Use POST /task endpoint instead",
				"service": cfg.ServiceName,
			})
		})
	}

	// Create and start server
	srv := server.New(server.Config{
		Addr:            ":" + appConfig.Port,
		Handler:         router,
		ReadTimeout:     appConfig.ReadTimeout,
		WriteTimeout:    appConfig.WriteTimeout,
		IdleTimeout:     time.Second * 60,
		ShutdownTimeout: appConfig.ShutdownTimeout,
		Logger:          log,
	})

	log.Infof("%s service starting on port %s", cfg.ServiceName, appConfig.Port)
	return srv.StartAndWait()
}

