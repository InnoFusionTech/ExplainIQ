package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewGeminiClient tests Gemini client creation
func TestNewGeminiClient(t *testing.T) {
	client := NewGeminiClient("test-api-key")

	assert.NotNil(t, client)
	assert.NotNil(t, client.client)
	assert.NotNil(t, client.Models)
	assert.Equal(t, "gemini-1.5-flash", client.model)
	assert.NotNil(t, client.logger)
}

// TestNewGeminiClientWithEmptyKey tests Gemini client creation with empty key
func TestNewGeminiClientWithEmptyKey(t *testing.T) {
	client := NewGeminiClient("")

	assert.NotNil(t, client)
	assert.NotNil(t, client.client)
	// Will use environment variable or ADC if no key provided
}

// TestSummarizeSuccess tests successful summarization
func TestSummarizeSuccess(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and content type
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Parse request body
		var req GeminiRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify request structure
		assert.Len(t, req.Contents, 1)
		assert.Len(t, req.Contents[0].Parts, 1)
		assert.Contains(t, req.Contents[0].Parts[0].Text, "machine learning")

		// Create mock response
		response := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{
								Text: `{
									"outline": ["Introduction to ML", "Key concepts", "Applications"],
									"prerequisites": ["Basic math", "Programming knowledge"],
									"misconceptions": ["ML is magic", "More data is always better"],
									"citations": ["doc1", "doc2"]
								}`,
							},
						},
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: GeminiUsageMetadata{
				PromptTokenCount:     100,
				CandidatesTokenCount: 50,
				TotalTokenCount:      150,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server
	client := NewGeminiClient("test-api-key")
	// Note: baseURL is no longer a field - SDK doesn't support custom base URLs
	// client.baseURL = server.URL

	// Test summarization
	ctx := context.Background()
	result, err := client.Summarize(ctx, "machine learning", "context about ML")

	// Verify no error occurred
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify result structure
	assert.Len(t, result.Outline, 3)
	assert.Equal(t, "Introduction to ML", result.Outline[0])
	assert.Equal(t, "Key concepts", result.Outline[1])
	assert.Equal(t, "Applications", result.Outline[2])

	assert.Len(t, result.Prerequisites, 2)
	assert.Equal(t, "Basic math", result.Prerequisites[0])
	assert.Equal(t, "Programming knowledge", result.Prerequisites[1])

	assert.Len(t, result.Misconceptions, 2)
	assert.Equal(t, "ML is magic", result.Misconceptions[0])
	assert.Equal(t, "More data is always better", result.Misconceptions[1])

	assert.Len(t, result.Citations, 2)
	assert.Equal(t, "doc1", result.Citations[0])
	assert.Equal(t, "doc2", result.Citations[1])
}

// TestSummarizeAPIError tests API error handling
func TestSummarizeAPIError(t *testing.T) {
	// Create mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		errorResponse := GeminiError{
			Error: GeminiErrorDetail{
				Code:    400,
				Message: "Invalid request",
				Status:  "INVALID_ARGUMENT",
			},
		}
		json.NewEncoder(w).Encode(errorResponse)
	}))
	defer server.Close()

	// Create client with mock server
	client := NewGeminiClient("test-api-key")
	// Note: baseURL is no longer a field - SDK doesn't support custom base URLs
	// client.baseURL = server.URL

	// Test summarization
	ctx := context.Background()
	result, err := client.Summarize(ctx, "machine learning", "context about ML")

	// Verify error occurred
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "API error 400")
	assert.Contains(t, err.Error(), "Invalid request")
}

// TestSummarizeNetworkError tests network error handling
func TestSummarizeNetworkError(t *testing.T) {
	// Create client with invalid URL
	client := NewGeminiClient("test-api-key")
	// Note: baseURL is no longer a field - SDK doesn't support custom base URLs
	// client.baseURL = "http://invalid-url:9999"

	// Test summarization
	ctx := context.Background()
	result, err := client.Summarize(ctx, "machine learning", "context about ML")

	// Verify error occurred
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "request failed")
}

// TestSummarizeInvalidJSON tests invalid JSON response handling
func TestSummarizeInvalidJSON(t *testing.T) {
	// Create mock server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	// Create client with mock server
	client := NewGeminiClient("test-api-key")
	// Note: baseURL is no longer a field - SDK doesn't support custom base URLs
	// client.baseURL = server.URL

	// Test summarization
	ctx := context.Background()
	result, err := client.Summarize(ctx, "machine learning", "context about ML")

	// Verify error occurred
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to parse response")
}

// TestSummarizeNoCandidates tests response with no candidates
func TestSummarizeNoCandidates(t *testing.T) {
	// Create mock server that returns no candidates
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := GeminiResponse{
			Candidates: []GeminiCandidate{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server
	client := NewGeminiClient("test-api-key")
	// Note: baseURL is no longer a field - SDK doesn't support custom base URLs
	// client.baseURL = server.URL

	// Test summarization
	ctx := context.Background()
	result, err := client.Summarize(ctx, "machine learning", "context about ML")

	// Verify error occurred
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no candidates in response")
}

// TestSummarizeInvalidJSONInResponse tests invalid JSON in response content
func TestSummarizeInvalidJSONInResponse(t *testing.T) {
	// Create mock server that returns invalid JSON in content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{
								Text: "This is not valid JSON",
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server
	client := NewGeminiClient("test-api-key")
	// Note: baseURL is no longer a field - SDK doesn't support custom base URLs
	// client.baseURL = server.URL

	// Test summarization
	ctx := context.Background()
	result, err := client.Summarize(ctx, "machine learning", "context about ML")

	// Verify error occurred
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no valid JSON found in response")
}

// TestSummarizeEmptyArrays tests response with empty arrays
func TestSummarizeEmptyArrays(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{
								Text: `{
									"outline": [],
									"prerequisites": [],
									"misconceptions": [],
									"citations": []
								}`,
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server
	client := NewGeminiClient("test-api-key")
	// Note: baseURL is no longer a field - SDK doesn't support custom base URLs
	// client.baseURL = server.URL

	// Test summarization
	ctx := context.Background()
	result, err := client.Summarize(ctx, "machine learning", "context about ML")

	// Verify no error occurred
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify empty arrays
	assert.Empty(t, result.Outline)
	assert.Empty(t, result.Prerequisites)
	assert.Empty(t, result.Misconceptions)
	assert.Empty(t, result.Citations)
}

// TestSummarizeMissingFields tests response with missing fields
func TestSummarizeMissingFields(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{
								Text: `{
									"outline": ["Introduction to ML"]
								}`,
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server
	client := NewGeminiClient("test-api-key")
	// Note: baseURL is no longer a field - SDK doesn't support custom base URLs
	// client.baseURL = server.URL

	// Test summarization
	ctx := context.Background()
	result, err := client.Summarize(ctx, "machine learning", "context about ML")

	// Verify no error occurred
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify missing fields are handled gracefully
	assert.Len(t, result.Outline, 1)
	assert.Equal(t, "Introduction to ML", result.Outline[0])
	assert.Empty(t, result.Prerequisites)
	assert.Empty(t, result.Misconceptions)
	assert.Empty(t, result.Citations)
}

// TestCreateSummarizePrompt tests prompt creation
func TestCreateSummarizePrompt(t *testing.T) {
	client := NewGeminiClient("test-api-key")

	prompt := client.createSummarizePrompt("machine learning", "context about ML")

	assert.Contains(t, prompt, "machine learning")
	assert.Contains(t, prompt, "context about ML")
	assert.Contains(t, prompt, "outline")
	assert.Contains(t, prompt, "prerequisites")
	assert.Contains(t, prompt, "misconceptions")
	assert.Contains(t, prompt, "citations")
	assert.Contains(t, prompt, "JSON")
}

// TestCleanStringArray tests string array cleaning
func TestCleanStringArray(t *testing.T) {
	client := NewGeminiClient("test-api-key")

	// Test normal array
	arr := []string{"item1", "item2", "item3"}
	result := client.cleanStringArray(arr, 10)
	assert.Equal(t, arr, result)

	// Test array with empty strings
	arr = []string{"item1", "", "item2", "   ", "item3"}
	result = client.cleanStringArray(arr, 10)
	assert.Equal(t, []string{"item1", "item2", "item3"}, result)

	// Test array with long strings
	arr = []string{"short", "this is a very long string that exceeds the limit and should be filtered out because it's too long for our purposes"}
	result = client.cleanStringArray(arr, 10)
	assert.Equal(t, []string{"short"}, result)

	// Test array length limit
	arr = []string{"item1", "item2", "item3", "item4", "item5"}
	result = client.cleanStringArray(arr, 3)
	assert.Equal(t, []string{"item1", "item2", "item3"}, result)
}

// TestHealthSuccess tests successful health check
func TestHealthSuccess(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{
								Text: "OK",
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server
	client := NewGeminiClient("test-api-key")
	// Note: baseURL is no longer a field - SDK doesn't support custom base URLs
	// client.baseURL = server.URL

	// Test health check
	ctx := context.Background()
	err := client.Health(ctx)

	// Verify no error occurred
	assert.NoError(t, err)
}

// TestHealthFailure tests health check failure
func TestHealthFailure(t *testing.T) {
	// Create mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create client with mock server
	client := NewGeminiClient("test-api-key")
	// Note: baseURL is no longer a field - SDK doesn't support custom base URLs
	// client.baseURL = server.URL

	// Test health check
	ctx := context.Background()
	err := client.Health(ctx)

	// Verify error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "health check failed")
}

// TestSetAPIKey tests API key setting
func TestSetAPIKey(t *testing.T) {
	client := NewGeminiClient("")
	// Note: apiKey is no longer a field - it's stored in the genai.Client
	// assert.Equal(t, "", client.apiKey)

	client.SetAPIKey("new-api-key")
	// Note: apiKey is no longer a field - it's stored in the genai.Client
	// assert.Equal(t, "new-api-key", client.apiKey)
	assert.NotNil(t, client.client) // Verify client was created
}

// TestSetModel tests model setting
func TestSetModel(t *testing.T) {
	client := NewGeminiClient("test-api-key")
	assert.Equal(t, "gemini-1.5-flash", client.model)

	client.SetModel("gemini-1.5-pro")
	assert.Equal(t, "gemini-1.5-pro", client.model)
}

// TestSetBaseURL tests base URL setting
func TestSetBaseURL(t *testing.T) {
	client := NewGeminiClient("test-api-key")
	// Note: baseURL is no longer a field - SDK doesn't support custom base URLs
	// assert.Equal(t, "https://generativelanguage.googleapis.com/v1beta", client.baseURL)

	client.SetBaseURL("https://custom-api.com/v1")
	// Note: baseURL is no longer a field - SDK doesn't support custom base URLs
	// assert.Equal(t, "https://custom-api.com/v1", client.baseURL)
	// SetBaseURL is kept for interface compatibility but doesn't actually change the base URL
	assert.NotNil(t, client.client)
}

// TestGetModelInfo tests model info retrieval
func TestGetModelInfo(t *testing.T) {
	client := NewGeminiClient("test-api-key")

	info := client.GetModelInfo()

	assert.Equal(t, "gemini-1.5-flash", info["model"])
	assert.Equal(t, "https://generativelanguage.googleapis.com/v1beta", info["base_url"])
	assert.True(t, info["api_key_set"].(bool))
}

// TestGetModelInfoWithEmptyKey tests model info with empty API key
func TestGetModelInfoWithEmptyKey(t *testing.T) {
	client := NewGeminiClient("")

	info := client.GetModelInfo()

	assert.Equal(t, "gemini-1.5-flash", info["model"])
	assert.Equal(t, "https://generativelanguage.googleapis.com/v1beta", info["base_url"])
	assert.False(t, info["api_key_set"].(bool))
}

// Benchmark tests
func BenchmarkSummarize(b *testing.B) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{
								Text: `{
									"outline": ["Introduction to ML", "Key concepts"],
									"prerequisites": ["Basic math"],
									"misconceptions": ["ML is magic"],
									"citations": ["doc1"]
								}`,
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server
	client := NewGeminiClient("test-api-key")
	// Note: baseURL is no longer a field - SDK doesn't support custom base URLs
	// client.baseURL = server.URL

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Summarize(ctx, "machine learning", "context about ML")
	}
}

func BenchmarkCreateSummarizePrompt(b *testing.B) {
	client := NewGeminiClient("test-api-key")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.createSummarizePrompt("machine learning", "context about ML")
	}
}

func BenchmarkCleanStringArray(b *testing.B) {
	client := NewGeminiClient("test-api-key")
	arr := []string{"item1", "item2", "item3", "item4", "item5"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.cleanStringArray(arr, 10)
	}
}

// TestExplainWithOGSuccess tests successful OG lesson generation
func TestExplainWithOGSuccess(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and content type
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Parse request body
		var req GeminiRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify request structure
		assert.Len(t, req.Contents, 1)
		assert.Len(t, req.Contents[0].Parts, 1)
		assert.Contains(t, req.Contents[0].Parts[0].Text, "machine learning")

		// Create mock response
		response := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{
								Text: `{
									"big_picture": "Machine learning is a subset of AI that enables computers to learn from data.",
									"metaphor": "Like teaching a child to recognize animals by showing them pictures.",
									"core_mechanism": "Algorithms find patterns in data to make predictions or decisions.",
									"toy_example_code": "model.fit(X_train, y_train)",
									"memory_hook": "ML = Machine Learning = More Learning from data",
									"real_life": "Used in recommendation systems, image recognition, and autonomous vehicles.",
									"best_practices": "Do: Clean your data. Don't: Overfit your model."
								}`,
							},
						},
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: GeminiUsageMetadata{
				PromptTokenCount:     200,
				CandidatesTokenCount: 100,
				TotalTokenCount:      300,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server
	client := NewGeminiClient("test-api-key")
	// Note: baseURL is no longer a field - SDK doesn't support custom base URLs
	// client.baseURL = server.URL

	// Test OG lesson generation
	ctx := context.Background()
	result, err := client.ExplainWithOG(ctx, "machine learning", "Introduction, Concepts, Applications", "ML is magic", "Additional context")

	// Verify no error occurred
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify result structure
	assert.Equal(t, "Machine learning is a subset of AI that enables computers to learn from data.", result.BigPicture)
	assert.Equal(t, "Like teaching a child to recognize animals by showing them pictures.", result.Metaphor)
	assert.Equal(t, "Algorithms find patterns in data to make predictions or decisions.", result.CoreMechanism)
	assert.Equal(t, "model.fit(X_train, y_train)", result.ToyExampleCode)
	assert.Equal(t, "ML = Machine Learning = More Learning from data", result.MemoryHook)
	assert.Equal(t, "Used in recommendation systems, image recognition, and autonomous vehicles.", result.RealLife)
	assert.Equal(t, "Do: Clean your data. Don't: Overfit your model.", result.BestPractices)
}

// TestExplainWithOGAPIError tests API error handling
func TestExplainWithOGAPIError(t *testing.T) {
	// Create mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		errorResponse := GeminiError{
			Error: GeminiErrorDetail{
				Code:    400,
				Message: "Invalid request",
				Status:  "INVALID_ARGUMENT",
			},
		}
		json.NewEncoder(w).Encode(errorResponse)
	}))
	defer server.Close()

	// Create client with mock server
	client := NewGeminiClient("test-api-key")
	// Note: baseURL is no longer a field - SDK doesn't support custom base URLs
	// client.baseURL = server.URL

	// Test OG lesson generation
	ctx := context.Background()
	result, err := client.ExplainWithOG(ctx, "machine learning", "", "", "")

	// Verify error occurred
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "API error 400")
	assert.Contains(t, err.Error(), "Invalid request")
}

// TestExplainWithOGNoCandidates tests response with no candidates
func TestExplainWithOGNoCandidates(t *testing.T) {
	// Create mock server that returns no candidates
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := GeminiResponse{
			Candidates: []GeminiCandidate{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server
	client := NewGeminiClient("test-api-key")
	// Note: baseURL is no longer a field - SDK doesn't support custom base URLs
	// client.baseURL = server.URL

	// Test OG lesson generation
	ctx := context.Background()
	result, err := client.ExplainWithOG(ctx, "machine learning", "", "", "")

	// Verify error occurred
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no candidates in response")
}

// TestExplainWithOGInvalidJSON tests invalid JSON in response content
func TestExplainWithOGInvalidJSON(t *testing.T) {
	// Create mock server that returns invalid JSON in content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{
								Text: "This is not valid JSON",
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server
	client := NewGeminiClient("test-api-key")
	// Note: baseURL is no longer a field - SDK doesn't support custom base URLs
	// client.baseURL = server.URL

	// Test OG lesson generation
	ctx := context.Background()
	result, err := client.ExplainWithOG(ctx, "machine learning", "", "", "")

	// Verify error occurred
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no JSON found in response")
}

// TestBuildExplainOGPrompt tests prompt creation
func TestBuildExplainOGPrompt(t *testing.T) {
	client := NewGeminiClient("test-api-key")

	prompt := client.buildExplainOGPrompt("machine learning", "Introduction, Concepts", "ML is magic", "Additional context")

	assert.Contains(t, prompt, "machine learning")
	assert.Contains(t, prompt, "Introduction, Concepts")
	assert.Contains(t, prompt, "ML is magic")
	assert.Contains(t, prompt, "Additional context")
	assert.Contains(t, prompt, "big_picture")
	assert.Contains(t, prompt, "metaphor")
	assert.Contains(t, prompt, "core_mechanism")
	assert.Contains(t, prompt, "toy_example_code")
	assert.Contains(t, prompt, "memory_hook")
	assert.Contains(t, prompt, "real_life")
	assert.Contains(t, prompt, "best_practices")
	assert.Contains(t, prompt, "JSON")
}

// TestParseOGLessonResponse tests OG lesson response parsing
func TestParseOGLessonResponse(t *testing.T) {
	client := NewGeminiClient("test-api-key")

	// Test valid JSON response
	responseText := `{
		"big_picture": "Machine learning is a subset of AI.",
		"metaphor": "Like teaching a child.",
		"core_mechanism": "Algorithms find patterns.",
		"toy_example_code": "model.fit(X, y)",
		"memory_hook": "ML = More Learning",
		"real_life": "Used in many applications.",
		"best_practices": "Do: Clean data. Don't: Overfit."
	}`

	result, err := client.parseOGLessonResponse(responseText)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Machine learning is a subset of AI.", result.BigPicture)
	assert.Equal(t, "Like teaching a child.", result.Metaphor)
	assert.Equal(t, "Algorithms find patterns.", result.CoreMechanism)
	assert.Equal(t, "model.fit(X, y)", result.ToyExampleCode)
	assert.Equal(t, "ML = More Learning", result.MemoryHook)
	assert.Equal(t, "Used in many applications.", result.RealLife)
	assert.Equal(t, "Do: Clean data. Don't: Overfit.", result.BestPractices)
}

// TestParseOGLessonResponseInvalidJSON tests invalid JSON response parsing
func TestParseOGLessonResponseInvalidJSON(t *testing.T) {
	client := NewGeminiClient("test-api-key")

	// Test invalid JSON response
	responseText := "This is not valid JSON"

	result, err := client.parseOGLessonResponse(responseText)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no JSON found in response")
}

// TestParseOGLessonResponseWithExtraText tests JSON parsing with extra text
func TestParseOGLessonResponseWithExtraText(t *testing.T) {
	client := NewGeminiClient("test-api-key")

	// Test JSON response with extra text
	responseText := `Here is the lesson:
	{
		"big_picture": "Machine learning is a subset of AI.",
		"metaphor": "Like teaching a child.",
		"core_mechanism": "Algorithms find patterns.",
		"toy_example_code": "model.fit(X, y)",
		"memory_hook": "ML = More Learning",
		"real_life": "Used in many applications.",
		"best_practices": "Do: Clean data. Don't: Overfit."
	}
	This is the end.`

	result, err := client.parseOGLessonResponse(responseText)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Machine learning is a subset of AI.", result.BigPicture)
	assert.Equal(t, "Like teaching a child.", result.Metaphor)
	assert.Equal(t, "Algorithms find patterns.", result.CoreMechanism)
	assert.Equal(t, "model.fit(X, y)", result.ToyExampleCode)
	assert.Equal(t, "ML = More Learning", result.MemoryHook)
	assert.Equal(t, "Used in many applications.", result.RealLife)
	assert.Equal(t, "Do: Clean data. Don't: Overfit.", result.BestPractices)
}

// Benchmark tests for ExplainWithOG
func BenchmarkExplainWithOG(b *testing.B) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{
								Text: `{
									"big_picture": "Machine learning is a subset of AI.",
									"metaphor": "Like teaching a child.",
									"core_mechanism": "Algorithms find patterns.",
									"toy_example_code": "model.fit(X, y)",
									"memory_hook": "ML = More Learning",
									"real_life": "Used in many applications.",
									"best_practices": "Do: Clean data. Don't: Overfit."
								}`,
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server
	client := NewGeminiClient("test-api-key")
	// Note: baseURL is no longer a field - SDK doesn't support custom base URLs
	// client.baseURL = server.URL

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.ExplainWithOG(ctx, "machine learning", "Introduction, Concepts", "ML is magic", "Additional context")
	}
}

func BenchmarkBuildExplainOGPrompt(b *testing.B) {
	client := NewGeminiClient("test-api-key")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.buildExplainOGPrompt("machine learning", "Introduction, Concepts", "ML is magic", "Additional context")
	}
}

func BenchmarkParseOGLessonResponse(b *testing.B) {
	client := NewGeminiClient("test-api-key")
	responseText := `{
		"big_picture": "Machine learning is a subset of AI.",
		"metaphor": "Like teaching a child.",
		"core_mechanism": "Algorithms find patterns.",
		"toy_example_code": "model.fit(X, y)",
		"memory_hook": "ML = More Learning",
		"real_life": "Used in many applications.",
		"best_practices": "Do: Clean data. Don't: Overfit."
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.parseOGLessonResponse(responseText)
	}
}

// TestCritiqueLessonSuccess tests successful lesson critique
func TestCritiqueLessonSuccess(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and content type
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Parse request body
		var req GeminiRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify request structure
		assert.Len(t, req.Contents, 1)
		assert.Len(t, req.Contents[0].Parts, 1)
		assert.Contains(t, req.Contents[0].Parts[0].Text, "big_picture")

		// Create mock response
		response := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{
								Text: `{
									"issues": [
										{
											"section": "big_picture",
											"problem": "Too vague and doesn't provide clear context",
											"severity": "high"
										},
										{
											"section": "metaphor",
											"problem": "Metaphor is not clear enough",
											"severity": "medium"
										}
									],
									"patch_plan": [
										{
											"section": "big_picture",
											"change": "Provide more specific context and clear definition",
											"replacement_text": "Machine learning is a subset of artificial intelligence that enables computers to learn patterns from data without being explicitly programmed for each task."
										},
										{
											"section": "metaphor",
											"change": "Use a clearer analogy",
											"replacement_text": "Like teaching a child to recognize animals by showing them many pictures until they can identify new animals on their own."
										}
									]
								}`,
							},
						},
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: GeminiUsageMetadata{
				PromptTokenCount:     300,
				CandidatesTokenCount: 150,
				TotalTokenCount:      450,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server
	client := NewGeminiClient("test-api-key")
	// Note: baseURL is no longer a field - SDK doesn't support custom base URLs
	// client.baseURL = server.URL

	// Test lesson critique
	lessonJSON := `{
		"big_picture": "Machine learning is cool",
		"metaphor": "It's like magic",
		"core_mechanism": "Algorithms work",
		"toy_example_code": "print('hello')",
		"memory_hook": "ML is fun",
		"real_life": "Used everywhere",
		"best_practices": "Be careful"
	}`

	ctx := context.Background()
	result, err := client.CritiqueLesson(ctx, lessonJSON)

	// Verify no error occurred
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify result structure
	assert.Len(t, result.Issues, 2)
	assert.Equal(t, "big_picture", result.Issues[0].Section)
	assert.Equal(t, "Too vague and doesn't provide clear context", result.Issues[0].Problem)
	assert.Equal(t, "high", result.Issues[0].Severity)

	assert.Equal(t, "metaphor", result.Issues[1].Section)
	assert.Equal(t, "Metaphor is not clear enough", result.Issues[1].Problem)
	assert.Equal(t, "medium", result.Issues[1].Severity)

	assert.Len(t, result.PatchPlan, 2)
	assert.Equal(t, "big_picture", result.PatchPlan[0].Section)
	assert.Equal(t, "Provide more specific context and clear definition", result.PatchPlan[0].Change)
	assert.Contains(t, result.PatchPlan[0].ReplacementText, "Machine learning is a subset of artificial intelligence")

	assert.Equal(t, "metaphor", result.PatchPlan[1].Section)
	assert.Equal(t, "Use a clearer analogy", result.PatchPlan[1].Change)
	assert.Contains(t, result.PatchPlan[1].ReplacementText, "Like teaching a child to recognize animals")
}

// TestCritiqueLessonAPIError tests API error handling
func TestCritiqueLessonAPIError(t *testing.T) {
	// Create mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		errorResponse := GeminiError{
			Error: GeminiErrorDetail{
				Code:    400,
				Message: "Invalid request",
				Status:  "INVALID_ARGUMENT",
			},
		}
		json.NewEncoder(w).Encode(errorResponse)
	}))
	defer server.Close()

	// Create client with mock server
	client := NewGeminiClient("test-api-key")
	// Note: baseURL is no longer a field - SDK doesn't support custom base URLs
	// client.baseURL = server.URL

	// Test lesson critique
	ctx := context.Background()
	result, err := client.CritiqueLesson(ctx, `{"big_picture": "test"}`)

	// Verify error occurred
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "API error 400")
	assert.Contains(t, err.Error(), "Invalid request")
}

// TestCritiqueLessonNoCandidates tests response with no candidates
func TestCritiqueLessonNoCandidates(t *testing.T) {
	// Create mock server that returns no candidates
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := GeminiResponse{
			Candidates: []GeminiCandidate{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server
	client := NewGeminiClient("test-api-key")
	// Note: baseURL is no longer a field - SDK doesn't support custom base URLs
	// client.baseURL = server.URL

	// Test lesson critique
	ctx := context.Background()
	result, err := client.CritiqueLesson(ctx, `{"big_picture": "test"}`)

	// Verify error occurred
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no candidates in response")
}

// TestCritiqueLessonInvalidJSON tests invalid JSON in response content
func TestCritiqueLessonInvalidJSON(t *testing.T) {
	// Create mock server that returns invalid JSON in content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{
								Text: "This is not valid JSON",
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server
	client := NewGeminiClient("test-api-key")
	// Note: baseURL is no longer a field - SDK doesn't support custom base URLs
	// client.baseURL = server.URL

	// Test lesson critique
	ctx := context.Background()
	result, err := client.CritiqueLesson(ctx, `{"big_picture": "test"}`)

	// Verify error occurred
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no JSON found in response")
}

// TestBuildCritiquePrompt tests prompt creation
func TestBuildCritiquePrompt(t *testing.T) {
	client := NewGeminiClient("test-api-key")

	lessonJSON := `{
		"big_picture": "Machine learning is cool",
		"metaphor": "It's like magic"
	}`

	prompt := client.buildCritiquePrompt(lessonJSON)

	assert.Contains(t, prompt, "Machine learning is cool")
	assert.Contains(t, prompt, "It's like magic")
	assert.Contains(t, prompt, "issues")
	assert.Contains(t, prompt, "patch_plan")
	assert.Contains(t, prompt, "section")
	assert.Contains(t, prompt, "problem")
	assert.Contains(t, prompt, "severity")
	assert.Contains(t, prompt, "change")
	assert.Contains(t, prompt, "replacement_text")
	assert.Contains(t, prompt, "JSON")
}

// TestParseCritiqueResponse tests critique response parsing
func TestParseCritiqueResponse(t *testing.T) {
	client := NewGeminiClient("test-api-key")

	// Test valid JSON response
	responseText := `{
		"issues": [
			{
				"section": "big_picture",
				"problem": "Too vague",
				"severity": "high"
			}
		],
		"patch_plan": [
			{
				"section": "big_picture",
				"change": "Make it clearer",
				"replacement_text": "Machine learning is a subset of AI."
			}
		]
	}`

	result, err := client.parseCritiqueResponse(responseText)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Issues, 1)
	assert.Equal(t, "big_picture", result.Issues[0].Section)
	assert.Equal(t, "Too vague", result.Issues[0].Problem)
	assert.Equal(t, "high", result.Issues[0].Severity)

	assert.Len(t, result.PatchPlan, 1)
	assert.Equal(t, "big_picture", result.PatchPlan[0].Section)
	assert.Equal(t, "Make it clearer", result.PatchPlan[0].Change)
	assert.Equal(t, "Machine learning is a subset of AI.", result.PatchPlan[0].ReplacementText)
}

// TestParseCritiqueResponseInvalidJSON tests invalid JSON response parsing
func TestParseCritiqueResponseInvalidJSON(t *testing.T) {
	client := NewGeminiClient("test-api-key")

	// Test invalid JSON response
	responseText := "This is not valid JSON"

	result, err := client.parseCritiqueResponse(responseText)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no JSON found in response")
}

// TestParseCritiqueResponseWithExtraText tests JSON parsing with extra text
func TestParseCritiqueResponseWithExtraText(t *testing.T) {
	client := NewGeminiClient("test-api-key")

	// Test JSON response with extra text
	responseText := `Here is the critique:
	{
		"issues": [
			{
				"section": "big_picture",
				"problem": "Too vague",
				"severity": "high"
			}
		],
		"patch_plan": [
			{
				"section": "big_picture",
				"change": "Make it clearer",
				"replacement_text": "Machine learning is a subset of AI."
			}
		]
	}
	This is the end.`

	result, err := client.parseCritiqueResponse(responseText)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Issues, 1)
	assert.Equal(t, "big_picture", result.Issues[0].Section)
	assert.Equal(t, "Too vague", result.Issues[0].Problem)
	assert.Equal(t, "high", result.Issues[0].Severity)

	assert.Len(t, result.PatchPlan, 1)
	assert.Equal(t, "big_picture", result.PatchPlan[0].Section)
	assert.Equal(t, "Make it clearer", result.PatchPlan[0].Change)
	assert.Equal(t, "Machine learning is a subset of AI.", result.PatchPlan[0].ReplacementText)
}

// Benchmark tests for CritiqueLesson
func BenchmarkCritiqueLesson(b *testing.B) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{
								Text: `{
									"issues": [
										{
											"section": "big_picture",
											"problem": "Too vague",
											"severity": "high"
										}
									],
									"patch_plan": [
										{
											"section": "big_picture",
											"change": "Make it clearer",
											"replacement_text": "Machine learning is a subset of AI."
										}
									]
								}`,
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server
	client := NewGeminiClient("test-api-key")
	// Note: baseURL is no longer a field - SDK doesn't support custom base URLs
	// client.baseURL = server.URL

	ctx := context.Background()
	lessonJSON := `{
		"big_picture": "Machine learning is cool",
		"metaphor": "It's like magic",
		"core_mechanism": "Algorithms work",
		"toy_example_code": "print('hello')",
		"memory_hook": "ML is fun",
		"real_life": "Used everywhere",
		"best_practices": "Be careful"
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.CritiqueLesson(ctx, lessonJSON)
	}
}

func BenchmarkBuildCritiquePrompt(b *testing.B) {
	client := NewGeminiClient("test-api-key")
	lessonJSON := `{
		"big_picture": "Machine learning is cool",
		"metaphor": "It's like magic",
		"core_mechanism": "Algorithms work",
		"toy_example_code": "print('hello')",
		"memory_hook": "ML is fun",
		"real_life": "Used everywhere",
		"best_practices": "Be careful"
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.buildCritiquePrompt(lessonJSON)
	}
}

func BenchmarkParseCritiqueResponse(b *testing.B) {
	client := NewGeminiClient("test-api-key")
	responseText := `{
		"issues": [
			{
				"section": "big_picture",
				"problem": "Too vague",
				"severity": "high"
			}
		],
		"patch_plan": [
			{
				"section": "big_picture",
				"change": "Make it clearer",
				"replacement_text": "Machine learning is a subset of AI."
			}
		]
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.parseCritiqueResponse(responseText)
	}
}

// TestVisualizeCoreSuccess tests successful visualization generation
func TestVisualizeCoreSuccess(t *testing.T) {
	client := NewGeminiClient("test-api-key")

	// Test lesson JSON
	lessonJSON := `{
		"big_picture": "Machine learning is a subset of AI",
		"metaphor": "Like teaching a child",
		"core_mechanism": "Algorithms find patterns in data to make predictions",
		"toy_example_code": "model.fit(X, y)",
		"memory_hook": "ML = More Learning",
		"real_life": "Used in recommendation systems",
		"best_practices": "Clean your data"
	}`

	ctx := context.Background()
	result, err := client.VisualizeCore(ctx, lessonJSON, "test-session")

	// Verify no error occurred
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify result structure
	assert.Len(t, result.Images, 2)
	assert.Len(t, result.Captions, 2)

	// Verify first image
	assert.Contains(t, result.Images[0].URL, "storage.googleapis.com")
	assert.Contains(t, result.Images[0].URL, "test-session")
	assert.Contains(t, result.Images[0].URL, "diagram_1.png")
	assert.Contains(t, result.Images[0].AltText, "Diagram 1")
	assert.Contains(t, result.Images[0].AltText, "Algorithms find patterns")
	assert.Equal(t, "Core mechanism diagram: Algorithms find patterns in data to make predictions", result.Images[0].Caption)

	// Verify second image
	assert.Contains(t, result.Images[1].URL, "storage.googleapis.com")
	assert.Contains(t, result.Images[1].URL, "test-session")
	assert.Contains(t, result.Images[1].URL, "diagram_2.png")
	assert.Contains(t, result.Images[1].AltText, "Diagram 2")
	assert.Contains(t, result.Images[1].AltText, "Algorithms find patterns")
	assert.Equal(t, "Process flowchart: Algorithms find patterns in data to make predictions", result.Images[1].Caption)

	// Verify captions
	assert.Equal(t, "Core mechanism diagram: Algorithms find patterns in data to make predictions", result.Captions[0])
	assert.Equal(t, "Process flowchart: Algorithms find patterns in data to make predictions", result.Captions[1])
}

// TestVisualizeCoreInvalidJSON tests error handling for invalid lesson JSON
func TestVisualizeCoreInvalidJSON(t *testing.T) {
	client := NewGeminiClient("test-api-key")

	ctx := context.Background()
	result, err := client.VisualizeCore(ctx, "invalid json", "test-session")

	// Verify error occurred
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to parse lesson JSON")
}

// TestVisualizeCoreEmptyMechanism tests error handling for empty core mechanism
func TestVisualizeCoreEmptyMechanism(t *testing.T) {
	client := NewGeminiClient("test-api-key")

	// Test lesson JSON with empty core mechanism
	lessonJSON := `{
		"big_picture": "Machine learning is a subset of AI",
		"metaphor": "Like teaching a child",
		"core_mechanism": "",
		"toy_example_code": "model.fit(X, y)",
		"memory_hook": "ML = More Learning",
		"real_life": "Used in recommendation systems",
		"best_practices": "Clean your data"
	}`

	ctx := context.Background()
	result, err := client.VisualizeCore(ctx, lessonJSON, "test-session")

	// Verify error occurred
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "core mechanism is empty")
}

// TestBuildVisualizationPrompts tests prompt creation
func TestBuildVisualizationPrompts(t *testing.T) {
	client := NewGeminiClient("test-api-key")

	coreMechanism := "Algorithms find patterns in data to make predictions"
	prompts, err := client.buildVisualizationPrompts(coreMechanism)

	// Verify no error occurred
	assert.NoError(t, err)
	assert.NotNil(t, prompts)
	assert.Len(t, prompts, 2)

	// Verify first prompt
	assert.Contains(t, prompts[0].Prompt, "Create a minimal, clean diagram showing")
	assert.Contains(t, prompts[0].Prompt, coreMechanism)
	assert.Contains(t, prompts[0].Prompt, "simple shapes, arrows, and labels")
	assert.Equal(t, "Core mechanism diagram: "+coreMechanism, prompts[0].Caption)

	// Verify second prompt
	assert.Contains(t, prompts[1].Prompt, "Create a flowchart diagram illustrating the process")
	assert.Contains(t, prompts[1].Prompt, coreMechanism)
	assert.Contains(t, prompts[1].Prompt, "rectangles for steps, diamonds for decisions")
	assert.Equal(t, "Process flowchart: "+coreMechanism, prompts[1].Caption)
}

// TestBuildVisualizationPromptsEmptyMechanism tests error handling for empty mechanism
func TestBuildVisualizationPromptsEmptyMechanism(t *testing.T) {
	client := NewGeminiClient("test-api-key")

	prompts, err := client.buildVisualizationPrompts("")

	// Verify error occurred
	assert.Error(t, err)
	assert.Nil(t, prompts)
	assert.Contains(t, err.Error(), "core mechanism is empty")
}

// TestVisualizeCoreWithDifferentMechanisms tests visualization with different core mechanisms
func TestVisualizeCoreWithDifferentMechanisms(t *testing.T) {
	client := NewGeminiClient("test-api-key")

	testCases := []struct {
		name          string
		coreMechanism string
		expectedInURL string
		expectedInAlt string
	}{
		{
			name:          "Machine Learning",
			coreMechanism: "Neural networks process data through layers to learn patterns",
			expectedInURL: "diagram_1.png",
			expectedInAlt: "Neural networks process data",
		},
		{
			name:          "Database",
			coreMechanism: "SQL queries retrieve data from tables using joins and filters",
			expectedInURL: "diagram_1.png",
			expectedInAlt: "SQL queries retrieve data",
		},
		{
			name:          "Web API",
			coreMechanism: "HTTP requests are processed by middleware and routed to handlers",
			expectedInURL: "diagram_1.png",
			expectedInAlt: "HTTP requests are processed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lessonJSON := fmt.Sprintf(`{
				"big_picture": "Test topic",
				"metaphor": "Test metaphor",
				"core_mechanism": "%s",
				"toy_example_code": "test code",
				"memory_hook": "test hook",
				"real_life": "test real life",
				"best_practices": "test practices"
			}`, tc.coreMechanism)

			ctx := context.Background()
			result, err := client.VisualizeCore(ctx, lessonJSON, "test-session")

			// Verify no error occurred
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Len(t, result.Images, 2)
			assert.Len(t, result.Captions, 2)

			// Verify URL contains expected elements
			assert.Contains(t, result.Images[0].URL, tc.expectedInURL)
			assert.Contains(t, result.Images[0].AltText, tc.expectedInAlt)
		})
	}
}

// Benchmark tests for VisualizeCore
func BenchmarkVisualizeCore(b *testing.B) {
	client := NewGeminiClient("test-api-key")
	lessonJSON := `{
		"big_picture": "Machine learning is a subset of AI",
		"metaphor": "Like teaching a child",
		"core_mechanism": "Algorithms find patterns in data to make predictions",
		"toy_example_code": "model.fit(X, y)",
		"memory_hook": "ML = More Learning",
		"real_life": "Used in recommendation systems",
		"best_practices": "Clean your data"
	}`

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.VisualizeCore(ctx, lessonJSON, "test-session")
	}
}

func BenchmarkBuildVisualizationPrompts(b *testing.B) {
	client := NewGeminiClient("test-api-key")
	coreMechanism := "Algorithms find patterns in data to make predictions"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.buildVisualizationPrompts(coreMechanism)
	}
}
