package llm

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
)

// ExampleEmbeddingUsage demonstrates how to use the embedding functionality
func ExampleEmbeddingUsage() {
	ctx := context.Background()

	// Set up environment variables (in production, use proper authentication)
	os.Setenv("EXPLAINIQ_PROJECT_ID", "your-project-id")
	os.Setenv("EXPLAINIQ_REGION", "us-central1")
	os.Setenv("GOOGLE_ACCESS_TOKEN", "your-access-token")

	// Create embedding client
	client := NewEmbeddingClient("your-project-id", "us-central1")

	// Configure client settings
	client.SetBatchSize(5)                // Maximum batch size for Vertex AI
	client.SetMaxRetries(3)               // Retry failed requests
	client.SetRetryDelay(1 * time.Second) // Initial retry delay
	client.SetMaxTokens(3072)             // Maximum tokens per text

	// Prepare texts for embedding
	texts := []string{
		"Machine learning is a subset of artificial intelligence that focuses on algorithms that can learn from data.",
		"Deep learning uses neural networks with multiple layers to model and understand complex patterns in data.",
		"Natural language processing combines computational linguistics with machine learning to help computers understand human language.",
		"Computer vision enables machines to interpret and understand visual information from the world.",
		"Reinforcement learning is a type of machine learning where agents learn to make decisions through trial and error.",
	}

	// Generate embeddings
	embeddings, err := client.Embed(ctx, texts)
	if err != nil {
		log.Fatalf("Failed to generate embeddings: %v", err)
	}

	// Process results
	fmt.Printf("Generated %d embeddings:\n", len(embeddings))
	for i, embedding := range embeddings {
		fmt.Printf("Text %d: %d dimensions, first 5 values: [%.3f, %.3f, %.3f, %.3f, %.3f]\n",
			i+1, len(embedding), embedding[0], embedding[1], embedding[2], embedding[3], embedding[4])
	}
}

// ExampleEmbeddingWithConvenienceFunction demonstrates using the convenience function
func ExampleEmbeddingWithConvenienceFunction() {
	ctx := context.Background()

	// Set up environment variables
	os.Setenv("EXPLAINIQ_PROJECT_ID", "your-project-id")
	os.Setenv("EXPLAINIQ_REGION", "us-central1")
	os.Setenv("GOOGLE_ACCESS_TOKEN", "your-access-token")

	// Use the convenience function
	texts := []string{
		"Artificial intelligence is transforming industries worldwide.",
		"Machine learning algorithms can identify patterns in large datasets.",
	}

	embeddings, err := Embed(ctx, texts)
	if err != nil {
		log.Fatalf("Failed to generate embeddings: %v", err)
	}

	fmt.Printf("Generated %d embeddings using convenience function\n", len(embeddings))
}

// ExampleEmbeddingBatching demonstrates how batching works
func ExampleEmbeddingBatching() {
	ctx := context.Background()

	// Create client with small batch size for demonstration
	client := NewEmbeddingClient("your-project-id", "us-central1")
	client.SetBatchSize(2) // Process 2 texts at a time

	// Large number of texts to demonstrate batching
	texts := make([]string, 10)
	for i := 0; i < 10; i++ {
		texts[i] = fmt.Sprintf("This is document number %d with some content about machine learning.", i+1)
	}

	fmt.Printf("Processing %d texts in batches of %d\n", len(texts), client.batchSize)

	// The client will automatically batch the requests
	embeddings, err := client.Embed(ctx, texts)
	if err != nil {
		log.Fatalf("Failed to generate embeddings: %v", err)
	}

	fmt.Printf("Successfully generated %d embeddings\n", len(embeddings))
}

// ExampleEmbeddingErrorHandling demonstrates error handling
func ExampleEmbeddingErrorHandling() {
	ctx := context.Background()

	client := NewEmbeddingClient("your-project-id", "us-central1")

	// Test various error conditions
	testCases := []struct {
		name          string
		texts         []string
		expectedError bool
	}{
		{
			name:          "Empty texts",
			texts:         []string{},
			expectedError: true,
		},
		{
			name:          "Nil texts",
			texts:         nil,
			expectedError: true,
		},
		{
			name:          "Empty string",
			texts:         []string{""},
			expectedError: true,
		},
		{
			name:          "Valid texts",
			texts:         []string{"Valid text 1", "Valid text 2"},
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		fmt.Printf("Testing %s...\n", tc.name)

		_, err := client.Embed(ctx, tc.texts)
		hasError := err != nil

		if hasError != tc.expectedError {
			fmt.Printf("  Unexpected result: expected error=%v, got error=%v\n", tc.expectedError, hasError)
		} else {
			fmt.Printf("  ✓ Expected result\n")
		}
	}
}

// ExampleEmbeddingConfiguration demonstrates client configuration
func ExampleEmbeddingConfiguration() {
	client := NewEmbeddingClient("your-project-id", "us-central1")

	// Get current configuration
	info := client.GetModelInfo()
	fmt.Println("Current configuration:")
	for key, value := range info {
		fmt.Printf("  %s: %v\n", key, value)
	}

	// Modify configuration
	fmt.Println("\nModifying configuration...")
	client.SetBatchSize(3)
	client.SetMaxRetries(5)
	client.SetRetryDelay(2 * time.Second)
	client.SetMaxTokens(2048)

	// Get updated configuration
	updatedInfo := client.GetModelInfo()
	fmt.Println("\nUpdated configuration:")
	for key, value := range updatedInfo {
		fmt.Printf("  %s: %v\n", key, value)
	}
}

// ExampleEmbeddingWithRetry demonstrates retry behavior
func ExampleEmbeddingWithRetry() {
	ctx := context.Background()

	client := NewEmbeddingClient("your-project-id", "us-central1")

	// Configure retry settings
	client.SetMaxRetries(3)
	client.SetRetryDelay(1 * time.Second)

	texts := []string{
		"This text will be processed with retry logic in case of temporary failures.",
	}

	fmt.Printf("Processing with retry configuration: max_retries=%d, retry_delay=%v\n",
		client.maxRetries, client.retryDelay)

	// The client will automatically retry on retryable errors
	embeddings, err := client.Embed(ctx, texts)
	if err != nil {
		log.Printf("Failed after retries: %v", err)
		return
	}

	fmt.Printf("Successfully generated embedding with %d dimensions\n", len(embeddings[0]))
}

// ExampleEmbeddingRequestResponse demonstrates request/response structures
func ExampleEmbeddingRequestResponse() {
	// Create a sample request
	request := EmbeddingRequest{
		Instances: []EmbeddingInstance{
			{
				Content:  "Sample text for embedding",
				TaskType: "RETRIEVAL_DOCUMENT",
				Title:    "Sample Document",
			},
		},
		Parameters: EmbeddingParameters{
			OutputDimensionality: 768,
		},
	}

	fmt.Println("Sample request structure:")
	fmt.Printf("  Instances: %d\n", len(request.Instances))
	fmt.Printf("  Task Type: %s\n", request.Instances[0].TaskType)
	fmt.Printf("  Output Dimensions: %d\n", request.Parameters.OutputDimensionality)

	// Create a sample response
	response := EmbeddingResponse{
		Predictions: []EmbeddingPrediction{
			{
				Embeddings: EmbeddingValue{
					Values: []float32{0.1, 0.2, 0.3, 0.4, 0.5}, // Sample embedding
				},
				Stats: EmbeddingStats{
					TokenCount: 5,
				},
			},
		},
		Metadata: EmbeddingMetadata{
			Model: "text-embedding-004",
		},
	}

	fmt.Println("\nSample response structure:")
	fmt.Printf("  Predictions: %d\n", len(response.Predictions))
	fmt.Printf("  Embedding Dimensions: %d\n", len(response.Predictions[0].Embeddings.Values))
	fmt.Printf("  Token Count: %d\n", response.Predictions[0].Stats.TokenCount)
	fmt.Printf("  Model: %s\n", response.Metadata.Model)
}




