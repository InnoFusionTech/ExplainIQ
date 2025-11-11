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
	// Create critic service
	service := NewCriticService()

	// Create Google ADK agent from TaskProcessor
	// Wrap logrus.Logger in an adapter to match the expected interface
	loggerAdapter := adkgoogle.NewLoggerAdapter(service.logger)
	adkAgent, err := adkgoogle.CreateAgent(
		constants.ServiceCritic,
		"Agent that critiques lessons and identifies issues with severity levels, providing patch plans for improvements",
		service,
		loggerAdapter,
	)
	if err != nil {
		service.logger.Fatalf("Failed to create Google ADK agent: %v", err)
	}

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = constants.DefaultPortCritic
	}

	// Create and start A2A server
	server, err := adkgoogle.NewA2AServer(adkAgent, port, service.logger)
	if err != nil {
		service.logger.Fatalf("Failed to create A2A server: %v", err)
	}

	service.logger.Infof("Starting Google ADK A2A server for %s on port %s", constants.ServiceCritic, port)
	service.logger.Infof("AgentCard available at: %s", server.GetAgentCardURL())
	if err := server.Start(); err != nil {
		service.logger.Fatalf("Failed to start A2A server: %v", err)
	}
}
