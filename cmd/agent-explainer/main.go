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
	// Create explainer service
	service := NewExplainerService()

	// Create Google ADK agent from TaskProcessor
	// Wrap logrus.Logger in an adapter to match the expected interface
	loggerAdapter := adkgoogle.NewLoggerAdapter(service.logger)
	adkAgent, err := adkgoogle.CreateAgent(
		constants.ServiceExplainer,
		"Agent that generates comprehensive explanations using OG (Open Generation) format including big picture, metaphor, core mechanism, toy examples, memory hooks, real-life applications, and best practices",
		service,
		loggerAdapter,
	)
	if err != nil {
		service.logger.Fatalf("Failed to create Google ADK agent: %v", err)
	}

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = constants.DefaultPortExplainer
	}

	// Create and start A2A server
	server, err := adkgoogle.NewA2AServer(adkAgent, port, service.logger)
	if err != nil {
		service.logger.Fatalf("Failed to create A2A server: %v", err)
	}

	service.logger.Infof("Starting Google ADK A2A server for %s on port %s", constants.ServiceExplainer, port)
	service.logger.Infof("AgentCard available at: %s", server.GetAgentCardURL())
	if err := server.Start(); err != nil {
		service.logger.Fatalf("Failed to start A2A server: %v", err)
	}
}
