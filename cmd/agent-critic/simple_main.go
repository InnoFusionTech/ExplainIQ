package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// TaskRequest represents a task request
type TaskRequest struct {
	SessionID string            `json:"session_id"`
	Inputs    map[string]string `json:"inputs"`
}

// TaskResponse represents a task response
type TaskResponse struct {
	SessionID string                 `json:"session_id"`
	Status    string                 `json:"status"`
	Outputs   map[string]interface{} `json:"outputs"`
	Artifacts map[string]interface{} `json:"artifacts"`
}

// Issue represents a critique issue
type Issue struct {
	Section  string `json:"section"`
	Problem  string `json:"problem"`
	Severity string `json:"severity"`
}

// PatchPlan represents a patch plan item
type PatchPlan struct {
	Section         string `json:"section"`
	Change          string `json:"change"`
	ReplacementText string `json:"replacement_text"`
}

// CriticService represents the critic service
type CriticService struct{}

// NewCriticService creates a new critic service
func NewCriticService() *CriticService {
	return &CriticService{}
}

// ProcessTask processes a critique task
func (s *CriticService) ProcessTask(ctx context.Context, req TaskRequest) (TaskResponse, error) {
	// Extract inputs
	lessonJSON, exists := req.Inputs["lesson"]
	if !exists {
		return TaskResponse{}, fmt.Errorf("lesson input is required")
	}

	// Use lessonJSON in the result (in real implementation, this would be parsed and analyzed)
	_ = lessonJSON

	// Simulate critique (in real implementation, this would call Gemini)
	issues := []Issue{
		{
			Section:  "big_picture",
			Problem:  "Could be more specific about the target audience",
			Severity: "medium",
		},
		{
			Section:  "metaphor",
			Problem:  "Metaphor might be too abstract for beginners",
			Severity: "low",
		},
	}

	patchPlan := []PatchPlan{
		{
			Section:         "big_picture",
			Change:          "Add target audience clarification",
			ReplacementText: "The big picture of this topic is understanding how it works at a fundamental level, specifically designed for beginners who want to grasp the core concepts.",
		},
		{
			Section:         "metaphor",
			Change:          "Simplify the metaphor",
			ReplacementText: "Think of this topic like cooking - you need ingredients (data) and follow a recipe (algorithm) to get a meal (result).",
		},
	}

	// Create response
	response := TaskResponse{
		SessionID: req.SessionID,
		Status:    "completed",
		Outputs: map[string]interface{}{
			"issues":     issues,
			"patch_plan": patchPlan,
		},
		Artifacts: map[string]interface{}{
			"critique":   issues,
			"patch_plan": patchPlan,
		},
	}

	return response, nil
}

func main() {
	// Create service
	service := NewCriticService()

	// Setup routes
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":    "healthy",
			"service":   "agent-critic",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	http.HandleFunc("/task", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req TaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Process task
		response, err := service.ProcessTask(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	fmt.Println("Starting critic service on :8083")
	log.Fatal(http.ListenAndServe(":8083", nil))
}



