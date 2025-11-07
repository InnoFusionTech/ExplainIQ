package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/InnoFusionTech/ExplainIQ/internal/adk"
	"github.com/InnoFusionTech/ExplainIQ/internal/agent"
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

	// Check if authentication is required (default to true for production, false for local dev)
	requireAuth := true
	if requireAuthStr := os.Getenv("REQUIRE_AUTH"); requireAuthStr != "" {
		if val, err := strconv.ParseBool(requireAuthStr); err == nil {
			requireAuth = val
		}
	}

	// Start agent service using shared infrastructure
	if err := agent.StartAgentService(agent.ServiceConfig{
		ServiceName: constants.ServiceCritic,
		DefaultPort: constants.DefaultPortCritic,
		DefaultURL:  constants.DefaultURLCritic,
		Processor:   service,
		RequireAuth: requireAuth,
	}); err != nil {
		service.logger.Fatalf("Failed to start service: %v", err)
	}
}
