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

// OGLesson represents an OG (Original Gangster) lesson
type OGLesson struct {
	BigPicture     string `json:"big_picture"`
	Metaphor       string `json:"metaphor"`
	CoreMechanism  string `json:"core_mechanism"`
	ToyExampleCode string `json:"toy_example_code"`
	MemoryHook     string `json:"memory_hook"`
	RealLife       string `json:"real_life"`
	BestPractices  string `json:"best_practices"`
}

// ExplainerService represents the explainer service
type ExplainerService struct{}

// NewExplainerService creates a new explainer service
func NewExplainerService() *ExplainerService {
	return &ExplainerService{}
}

// ProcessTask processes an explanation task
func (s *ExplainerService) ProcessTask(ctx context.Context, req TaskRequest) (TaskResponse, error) {
	// Extract inputs
	topic, exists := req.Inputs["topic"]
	if !exists {
		return TaskResponse{}, fmt.Errorf("topic input is required")
	}

	outline, exists := req.Inputs["outline"]
	if !exists {
		outline = "No outline provided"
	}

	misconceptions, exists := req.Inputs["misconceptions"]
	if !exists {
		misconceptions = "No misconceptions provided"
	}

	context, exists := req.Inputs["context"]
	if !exists {
		context = "No additional context provided"
	}

	// Use inputs in the result (in real implementation, this would be used in the prompt)
	_ = outline
	_ = misconceptions
	_ = context

	// Simulate explanation (in real implementation, this would call Gemini)
	lesson := OGLesson{
		BigPicture:     fmt.Sprintf("The big picture of %s is understanding how it works at a fundamental level", topic),
		Metaphor:       fmt.Sprintf("Think of %s like a recipe - you need the right ingredients (data) and steps (algorithms)", topic),
		CoreMechanism:  fmt.Sprintf("The core mechanism of %s involves processing input data through various transformations", topic),
		ToyExampleCode: fmt.Sprintf("// Simple %s example\nfunction process%s(data) {\n  return data.map(x => x * 2);\n}", topic, topic),
		MemoryHook:     fmt.Sprintf("Remember %s by thinking: 'Data in, magic happens, results out'", topic),
		RealLife:       fmt.Sprintf("In real life, %s is used in recommendation systems, image recognition, and more", topic),
		BestPractices:  fmt.Sprintf("Best practices for %s include: start simple, validate your data, and iterate", topic),
	}

	// Create response
	response := TaskResponse{
		SessionID: req.SessionID,
		Status:    "completed",
		Outputs: map[string]interface{}{
			"lesson": lesson,
		},
		Artifacts: map[string]interface{}{
			"lesson": lesson,
		},
	}

	return response, nil
}

func main() {
	// Create service
	service := NewExplainerService()

	// Setup routes
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":    "healthy",
			"service":   "agent-explainer",
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

	fmt.Println("Starting explainer service on :8082")
	log.Fatal(http.ListenAndServe(":8082", nil))
}



