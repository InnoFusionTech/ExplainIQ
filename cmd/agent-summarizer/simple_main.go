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

// SummarizeResult represents the result of summarization
type SummarizeResult struct {
	Outline        []string `json:"outline"`
	Prerequisites  []string `json:"prerequisites"`
	Misconceptions []string `json:"misconceptions"`
	Citations      []string `json:"citations"`
}

// SummarizerService represents the summarizer service
type SummarizerService struct{}

// NewSummarizerService creates a new summarizer service
func NewSummarizerService() *SummarizerService {
	return &SummarizerService{}
}

// ProcessTask processes a summarization task
func (s *SummarizerService) ProcessTask(ctx context.Context, req TaskRequest) (TaskResponse, error) {
	// Extract inputs
	topic, exists := req.Inputs["topic"]
	if !exists {
		return TaskResponse{}, fmt.Errorf("topic input is required")
	}

	contextData, exists := req.Inputs["context"]
	if !exists {
		contextData = "No additional context provided"
	}

	// Use contextData in the result (in real implementation, this would be used in the prompt)
	_ = contextData

	// Simulate summarization (in real implementation, this would call Gemini)
	result := SummarizeResult{
		Outline: []string{
			fmt.Sprintf("Introduction to %s", topic),
			fmt.Sprintf("Core concepts of %s", topic),
			fmt.Sprintf("Applications of %s", topic),
			fmt.Sprintf("Best practices for %s", topic),
		},
		Prerequisites: []string{
			"Basic understanding of programming",
			"Familiarity with data structures",
		},
		Misconceptions: []string{
			fmt.Sprintf("%s is too complex for beginners", topic),
			fmt.Sprintf("%s requires advanced mathematics", topic),
		},
		Citations: []string{"doc1", "doc2", "doc3"},
	}

	// Create response
	response := TaskResponse{
		SessionID: req.SessionID,
		Status:    "completed",
		Outputs: map[string]interface{}{
			"outline":        result.Outline,
			"prerequisites":  result.Prerequisites,
			"misconceptions": result.Misconceptions,
		},
		Artifacts: map[string]interface{}{
			"outline":        result.Outline,
			"prerequisites":  result.Prerequisites,
			"misconceptions": result.Misconceptions,
			"citations":      result.Citations,
		},
	}

	return response, nil
}

func main() {
	// Create service
	service := NewSummarizerService()

	// Setup routes
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":    "healthy",
			"service":   "agent-summarizer",
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

	fmt.Println("Starting summarizer service on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
