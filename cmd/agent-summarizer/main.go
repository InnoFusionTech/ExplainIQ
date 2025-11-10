package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/InnoFusionTech/ExplainIQ/internal/adk"
	adkgoogle "github.com/InnoFusionTech/ExplainIQ/internal/adk/google"
	"github.com/InnoFusionTech/ExplainIQ/internal/constants"
	"github.com/InnoFusionTech/ExplainIQ/internal/cost_tracker"
	"github.com/InnoFusionTech/ExplainIQ/internal/llm"
	"github.com/InnoFusionTech/ExplainIQ/internal/storage"
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

	// Create storage client for cost tracking (optional)
	var costTracker *cost_tracker.CostTracker
	gcpProjectID := os.Getenv("GCP_PROJECT_ID")
	if gcpProjectID != "" {
		storageClient, err := storage.NewFirestoreClient(context.Background(), gcpProjectID, "sessions")
		if err != nil {
			logrus.WithError(err).Warn("Failed to create Firestore client, continuing without cost tracking")
		} else {
			costTracker = cost_tracker.NewCostTracker(storageClient)
		}
	} else {
		logrus.Warn("GCP_PROJECT_ID not set, continuing without cost tracking")
	}

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
	if s.costTracker != nil {
		inputTokens := len(topic) + len(context)                                                     // Rough estimate
		outputTokens := len(result.Outline) + len(result.Prerequisites) + len(result.Misconceptions) // Rough estimate

		// Track the cost
		if err := s.costTracker.TrackLLMCall(ctx, req.SessionID, "", "", "gemini-pro", inputTokens, outputTokens); err != nil {
			s.logger.WithFields(logrus.Fields{
				"session_id": req.SessionID,
				"error":      err,
			}).Warn("Failed to track LLM call cost")
		}
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
	// Create summarizer service
	service := NewSummarizerService()

	// Create Google ADK agent from TaskProcessor
	adkAgent, err := adkgoogle.CreateAgent(
		constants.ServiceSummarizer,
		"Agent that summarizes topics and extracts key information including outline, prerequisites, misconceptions, and citations",
		service,
		service.logger,
	)
	if err != nil {
		service.logger.Fatalf("Failed to create Google ADK agent: %v", err)
	}

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = constants.DefaultPortSummarizer
	}

	// Create and start A2A server
	server, err := adkgoogle.NewA2AServer(adkAgent, port, service.logger)
	if err != nil {
		service.logger.Fatalf("Failed to create A2A server: %v", err)
	}

	service.logger.Infof("Starting Google ADK A2A server for %s on port %s", constants.ServiceSummarizer, port)
	service.logger.Infof("AgentCard available at: %s", server.GetAgentCardURL())
	if err := server.Start(); err != nil {
		service.logger.Fatalf("Failed to start A2A server: %v", err)
	}
}
