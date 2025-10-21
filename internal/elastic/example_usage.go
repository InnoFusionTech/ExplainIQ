package elastic

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ExampleUsage demonstrates how to use the Elasticsearch client
func ExampleUsage() {
	ctx := context.Background()

	// Create client
	client, err := NewClient(ctx, "https://your-elasticsearch-cluster.com:9200", "your-api-key")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Create index with hybrid search mapping
	indexName := "explainiq-docs"
	err = client.CreateIndex(ctx, indexName, nil) // Uses default hybrid mapping
	if err != nil {
		log.Fatalf("Failed to create index: %v", err)
	}

	// Prepare documents for indexing
	docs := []Doc{
		{
			ID:        "doc-1",
			Topic:     "Machine Learning",
			Section:   "Introduction",
			Text:      "Machine learning is a subset of artificial intelligence that focuses on algorithms that can learn from data.",
			Embedding: generateMockEmbedding(1536), // In real usage, use actual embeddings
			Metadata: map[string]string{
				"author": "John Doe",
				"source": "ML Textbook",
				"year":   "2023",
			},
			CreatedAt: time.Now().Format(time.RFC3339),
			UpdatedAt: time.Now().Format(time.RFC3339),
		},
		{
			ID:        "doc-2",
			Topic:     "Deep Learning",
			Section:   "Neural Networks",
			Text:      "Deep learning uses neural networks with multiple layers to model and understand complex patterns in data.",
			Embedding: generateMockEmbedding(1536),
			Metadata: map[string]string{
				"author": "Jane Smith",
				"source": "DL Research Paper",
				"year":   "2023",
			},
			CreatedAt: time.Now().Format(time.RFC3339),
			UpdatedAt: time.Now().Format(time.RFC3339),
		},
	}

	// Upsert documents using bulk API
	err = client.UpsertDocs(ctx, indexName, docs)
	if err != nil {
		log.Fatalf("Failed to upsert documents: %v", err)
	}

	// Perform hybrid search
	query := "machine learning algorithms"
	queryEmbedding := generateMockEmbedding(1536) // In real usage, generate from query text

	results, err := client.HybridSearch(ctx, indexName, query, queryEmbedding, 10)
	if err != nil {
		log.Fatalf("Failed to perform hybrid search: %v", err)
	}

	// Process results
	fmt.Printf("Found %d results:\n", results.Total)
	for i, hit := range results.Hits {
		fmt.Printf("%d. ID: %s, Score: %.3f\n", i+1, hit.ID, hit.Score)
		if source, ok := hit.Source["text"].(string); ok {
			fmt.Printf("   Text: %s\n", source)
		}
	}
}

// ExampleHybridSearchMapping shows how to create a custom mapping
func ExampleHybridSearchMapping() {
	ctx := context.Background()

	client, err := NewClient(ctx, "https://your-elasticsearch-cluster.com:9200", "your-api-key")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Custom mapping for different embedding dimensions
	customMapping := map[string]interface{}{
		"settings": map[string]interface{}{
			"number_of_shards":   2,
			"number_of_replicas": 1,
			"analysis": map[string]interface{}{
				"analyzer": map[string]interface{}{
					"custom_analyzer": map[string]interface{}{
						"type":      "custom",
						"tokenizer": "standard",
						"filter":    []string{"lowercase", "stop", "snowball", "synonym"},
					},
				},
			},
		},
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type": "keyword",
				},
				"topic": map[string]interface{}{
					"type":     "text",
					"analyzer": "custom_analyzer",
					"fields": map[string]interface{}{
						"keyword": map[string]interface{}{
							"type": "keyword",
						},
					},
				},
				"text": map[string]interface{}{
					"type":     "text",
					"analyzer": "custom_analyzer",
				},
				"embedding": map[string]interface{}{
					"type":       "dense_vector",
					"dims":       768, // Different embedding model dimensions
					"index":      true,
					"similarity": "dot_product",
				},
				"metadata": map[string]interface{}{
					"type": "object",
				},
			},
		},
	}

	// Create index with custom mapping
	err = client.CreateIndex(ctx, "custom-index", customMapping)
	if err != nil {
		log.Fatalf("Failed to create custom index: %v", err)
	}
}

// ExampleBulkOperations shows how to perform bulk operations
func ExampleBulkOperations() {
	ctx := context.Background()

	client, err := NewClient(ctx, "https://your-elasticsearch-cluster.com:9200", "your-api-key")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Create index
	indexName := "bulk-test"
	err = client.CreateIndex(ctx, indexName, nil)
	if err != nil {
		log.Fatalf("Failed to create index: %v", err)
	}

	// Prepare large batch of documents
	var docs []Doc
	for i := 0; i < 1000; i++ {
		docs = append(docs, Doc{
			ID:        fmt.Sprintf("bulk-doc-%d", i),
			Topic:     fmt.Sprintf("Topic %d", i%10),
			Section:   fmt.Sprintf("Section %d", i%5),
			Text:      fmt.Sprintf("This is document number %d with some content.", i),
			Embedding: generateMockEmbedding(1536),
			Metadata: map[string]string{
				"batch": "bulk-test",
				"index": fmt.Sprintf("%d", i),
			},
			CreatedAt: time.Now().Format(time.RFC3339),
		})
	}

	// Upsert in bulk
	err = client.UpsertDocs(ctx, indexName, docs)
	if err != nil {
		log.Fatalf("Failed to bulk upsert: %v", err)
	}

	fmt.Printf("Successfully indexed %d documents\n", len(docs))
}

// ExampleIndexManagement shows how to manage indices
func ExampleIndexManagement() {
	ctx := context.Background()

	client, err := NewClient(ctx, "https://your-elasticsearch-cluster.com:9200", "your-api-key")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	indexName := "management-test"

	// Check if index exists
	exists, err := client.IndexExists(ctx, indexName)
	if err != nil {
		log.Fatalf("Failed to check index existence: %v", err)
	}

	if exists {
		fmt.Printf("Index %s already exists\n", indexName)

		// Delete existing index
		err = client.DeleteIndex(ctx, indexName)
		if err != nil {
			log.Fatalf("Failed to delete index: %v", err)
		}
		fmt.Printf("Deleted existing index %s\n", indexName)
	}

	// Create new index
	err = client.CreateIndex(ctx, indexName, nil)
	if err != nil {
		log.Fatalf("Failed to create index: %v", err)
	}
	fmt.Printf("Created new index %s\n", indexName)

	// Verify index exists
	exists, err = client.IndexExists(ctx, indexName)
	if err != nil {
		log.Fatalf("Failed to check index existence: %v", err)
	}

	if exists {
		fmt.Printf("Index %s exists and is ready\n", indexName)
	}
}

// generateMockEmbedding creates a mock embedding vector for testing
func generateMockEmbedding(dims int) []float32 {
	embedding := make([]float32, dims)
	for i := range embedding {
		embedding[i] = float32(i) / float32(dims)
	}
	return embedding
}

