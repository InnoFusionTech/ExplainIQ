package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/InnoFusionTech/ExplainIQ/internal/adk"
	adkgoogle "github.com/InnoFusionTech/ExplainIQ/internal/adk/google"
	"github.com/InnoFusionTech/ExplainIQ/internal/constants"
	"github.com/InnoFusionTech/ExplainIQ/internal/llm"
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
	// Create visualizer service
	service := NewVisualizerService()

	// Create Google ADK agent from TaskProcessor
	adkAgent, err := adkgoogle.CreateAgent(
		constants.ServiceVisualizer,
		"Agent that generates visualizations for lessons including images and captions",
		service,
		service.logger,
	)
	if err != nil {
		service.logger.Fatalf("Failed to create Google ADK agent: %v", err)
	}

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = constants.DefaultPortVisualizer
	}

	// Create and start A2A server
	server, err := adkgoogle.NewA2AServer(adkAgent, port, service.logger)
	if err != nil {
		service.logger.Fatalf("Failed to create A2A server: %v", err)
	}

	service.logger.Infof("Starting Google ADK A2A server for %s on port %s", constants.ServiceVisualizer, port)
	service.logger.Infof("AgentCard available at: %s", server.GetAgentCardURL())
	if err := server.Start(); err != nil {
		service.logger.Fatalf("Failed to start A2A server: %v", err)
	}
}
