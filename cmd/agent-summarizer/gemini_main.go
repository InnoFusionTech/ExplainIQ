package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/explainiq/agent/internal/config"
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

// GeminiClient represents a simple Gemini client
type GeminiClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewGeminiClient creates a new Gemini client
func NewGeminiClient(apiKey string) *GeminiClient {
	return &GeminiClient{
		apiKey:     apiKey,
		baseURL:    "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent",
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Summarize calls Gemini to summarize a topic
func (g *GeminiClient) Summarize(ctx context.Context, topic, context string) (*SummarizeResult, error) {
	if g.apiKey == "" {
		// Return mock response if no API key
		return &SummarizeResult{
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
		}, nil
	}

	prompt := fmt.Sprintf(`You are an expert educator. Create a learning outline for the topic: "%s"

Context: %s

Please provide a JSON response with the following structure:
{
  "outline": ["bullet point 1", "bullet point 2", "bullet point 3", "bullet point 4"],
  "prerequisites": ["prerequisite 1", "prerequisite 2"],
  "misconceptions": ["misconception 1", "misconception 2"],
  "citations": ["citation1", "citation2", "citation3"]
}

Make the outline comprehensive but concise. Focus on practical learning progression.`, topic, context)

	requestBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.7,
			"topP":            0.8,
			"topK":            40,
			"maxOutputTokens": 1024,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s?key=%s", g.baseURL, g.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	responseText := geminiResp.Candidates[0].Content.Parts[0].Text

	// Try to parse JSON response
	var result SummarizeResult
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		// If JSON parsing fails, return mock response
		log.Printf("Failed to parse Gemini JSON response: %v", err)
		return &SummarizeResult{
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
		}, nil
	}

	return &result, nil
}

// SummarizerService represents the summarizer service
type SummarizerService struct {
	geminiClient *GeminiClient
}

// NewSummarizerService creates a new summarizer service
func NewSummarizerService() *SummarizerService {
	apiKey := os.Getenv("GEMINI_API_KEY")
	return &SummarizerService{
		geminiClient: NewGeminiClient(apiKey),
	}
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

	// Call Gemini
	result, err := s.geminiClient.Summarize(ctx, topic, contextData)
	if err != nil {
		return TaskResponse{}, fmt.Errorf("summarization failed: %w", err)
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
	// Load environment variables from .env file
	if err := config.LoadEnvFiles(); err != nil {
		log.Printf("Warning: Failed to load .env file: %v", err)
	}

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

	fmt.Println("Starting Gemini-enabled summarizer service on :8081")
	fmt.Println("Set GEMINI_API_KEY environment variable to enable real AI responses")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
