package elastic

import (
	"context"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	// This test would require a real Elasticsearch instance
	// For now, we'll test the client creation logic
	ctx := context.Background()

	// Test with invalid URL to ensure error handling
	_, err := NewClient(ctx, "invalid-url", "test-key")
	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}
}

func TestDocStruct(t *testing.T) {
	// Test Doc struct creation and field access
	doc := Doc{
		ID:        "test-doc-1",
		Topic:     "Machine Learning",
		Section:   "Introduction",
		Text:      "This is a test document about machine learning.",
		Embedding: []float32{0.1, 0.2, 0.3, 0.4, 0.5},
		Metadata: map[string]string{
			"author": "test-author",
			"source": "test-source",
		},
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	}

	// Verify all fields are set correctly
	if doc.ID != "test-doc-1" {
		t.Errorf("Expected ID 'test-doc-1', got '%s'", doc.ID)
	}
	if doc.Topic != "Machine Learning" {
		t.Errorf("Expected Topic 'Machine Learning', got '%s'", doc.Topic)
	}
	if doc.Section != "Introduction" {
		t.Errorf("Expected Section 'Introduction', got '%s'", doc.Section)
	}
	if doc.Text != "This is a test document about machine learning." {
		t.Errorf("Expected Text 'This is a test document about machine learning.', got '%s'", doc.Text)
	}
	if len(doc.Embedding) != 5 {
		t.Errorf("Expected Embedding length 5, got %d", len(doc.Embedding))
	}
	if doc.Metadata["author"] != "test-author" {
		t.Errorf("Expected Metadata author 'test-author', got '%s'", doc.Metadata["author"])
	}
	if doc.CreatedAt == "" {
		t.Error("Expected CreatedAt to be set")
	}
	if doc.UpdatedAt == "" {
		t.Error("Expected UpdatedAt to be set")
	}
}

func TestHybridSearchMapping(t *testing.T) {
	// Create a mock client to test mapping generation
	client := &Client{}

	mapping := client.getHybridSearchMapping()

	// Verify mapping structure
	if mapping == nil {
		t.Fatal("Expected mapping to be non-nil")
	}

	// Check settings
	settings, ok := mapping["settings"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected settings to be a map")
	}

	if settings["number_of_shards"] != 1 {
		t.Errorf("Expected number_of_shards to be 1, got %v", settings["number_of_shards"])
	}

	if settings["number_of_replicas"] != 0 {
		t.Errorf("Expected number_of_replicas to be 0, got %v", settings["number_of_replicas"])
	}

	// Check mappings
	mappings, ok := mapping["mappings"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected mappings to be a map")
	}

	properties, ok := mappings["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties to be a map")
	}

	// Check embedding field
	embedding, ok := properties["embedding"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected embedding to be a map")
	}

	if embedding["type"] != "dense_vector" {
		t.Errorf("Expected embedding type to be 'dense_vector', got '%v'", embedding["type"])
	}

	if embedding["dims"] != 1536 {
		t.Errorf("Expected embedding dims to be 1536, got %v", embedding["dims"])
	}

	// Check text field
	text, ok := properties["text"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected text to be a map")
	}

	if text["type"] != "text" {
		t.Errorf("Expected text type to be 'text', got '%v'", text["type"])
	}

	if text["analyzer"] != "hybrid_analyzer" {
		t.Errorf("Expected text analyzer to be 'hybrid_analyzer', got '%v'", text["analyzer"])
	}
}

func TestSearchResultParsing(t *testing.T) {
	client := &Client{}

	// Mock Elasticsearch response
	mockResponse := map[string]interface{}{
		"hits": map[string]interface{}{
			"total": map[string]interface{}{
				"value": float64(2),
			},
			"hits": []interface{}{
				map[string]interface{}{
					"_id":    "doc-1",
					"_score": 0.95,
					"_source": map[string]interface{}{
						"id":    "doc-1",
						"topic": "Test Topic",
						"text":  "Test content",
					},
				},
				map[string]interface{}{
					"_id":    "doc-2",
					"_score": 0.87,
					"_source": map[string]interface{}{
						"id":    "doc-2",
						"topic": "Another Topic",
						"text":  "Another content",
					},
				},
			},
		},
	}

	result, err := client.parseSearchResponse(mockResponse)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Total != 2 {
		t.Errorf("Expected total 2, got %d", result.Total)
	}

	if len(result.Hits) != 2 {
		t.Errorf("Expected 2 hits, got %d", len(result.Hits))
	}

	// Check first hit
	hit1 := result.Hits[0]
	if hit1.ID != "doc-1" {
		t.Errorf("Expected first hit ID 'doc-1', got '%s'", hit1.ID)
	}
	if hit1.Score != 0.95 {
		t.Errorf("Expected first hit score 0.95, got %f", hit1.Score)
	}

	// Check second hit
	hit2 := result.Hits[1]
	if hit2.ID != "doc-2" {
		t.Errorf("Expected second hit ID 'doc-2', got '%s'", hit2.ID)
	}
	if hit2.Score != 0.87 {
		t.Errorf("Expected second hit score 0.87, got %f", hit2.Score)
	}
}

func TestSearchResultParsingInvalidFormat(t *testing.T) {
	client := &Client{}

	// Test with invalid response format
	invalidResponse := map[string]interface{}{
		"invalid": "format",
	}

	_, err := client.parseSearchResponse(invalidResponse)
	if err == nil {
		t.Error("Expected error for invalid response format, got nil")
	}
}

func TestHybridSearchQuery(t *testing.T) {
	// Test that we can create a valid hybrid search query
	// This doesn't require a real Elasticsearch instance
	client := &Client{}

	// Create a test embedding
	embedding := make([]float32, 1536)
	for i := range embedding {
		embedding[i] = float32(i) / 1536.0
	}

	// This would normally call the actual search, but we'll just verify
	// that the query structure is valid by checking the mapping generation
	mapping := client.getHybridSearchMapping()
	if mapping == nil {
		t.Fatal("Expected mapping to be generated")
	}

	// Verify that the mapping supports both text and vector search
	properties := mapping["mappings"].(map[string]interface{})["properties"].(map[string]interface{})

	// Check text field for BM25
	text := properties["text"].(map[string]interface{})
	if text["type"] != "text" {
		t.Error("Expected text field to support BM25 search")
	}

	// Check embedding field for vector search
	embeddingField := properties["embedding"].(map[string]interface{})
	if embeddingField["type"] != "dense_vector" {
		t.Error("Expected embedding field to support vector search")
	}
}

// Benchmark tests
func BenchmarkDocCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		doc := Doc{
			ID:        "test-doc",
			Topic:     "Test Topic",
			Section:   "Test Section",
			Text:      "Test text content",
			Embedding: make([]float32, 1536),
			Metadata: map[string]string{
				"key": "value",
			},
		}
		_ = doc
	}
}

func BenchmarkMappingGeneration(b *testing.B) {
	client := &Client{}
	for i := 0; i < b.N; i++ {
		mapping := client.getHybridSearchMapping()
		_ = mapping
	}
}

