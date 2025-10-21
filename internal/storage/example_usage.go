package storage

import (
	"context"
	"fmt"
	"log"
)

// ExampleFirestoreUsage demonstrates how to use the Firestore client
func ExampleFirestoreUsage() {
	ctx := context.Background()

	// Create Firestore client
	client, err := NewFirestoreClient(ctx, "your-project-id", "sessions")
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	// Create a session
	session, err := client.CreateSession(ctx, "machine learning basics")
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	fmt.Printf("Created session: %s\n", session.ID)

	// Save some steps
	steps := []struct {
		name     string
		payload  map[string]interface{}
		duration int
	}{
		{
			name: "analyze-topic",
			payload: map[string]interface{}{
				"output": "Analyzed topic: machine learning basics",
				"keywords": []string{"machine learning", "algorithms", "data"},
			},
			duration: 2000,
		},
		{
			name: "generate-outline",
			payload: map[string]interface{}{
				"output": "Generated lesson outline",
				"sections": []string{"Introduction", "Core Concepts", "Examples", "Summary"},
			},
			duration: 1500,
		},
		{
			name: "create-content",
			payload: map[string]interface{}{
				"output": "Created lesson content",
				"word_count": 1500,
				"difficulty": "beginner",
			},
			duration: 5000,
		},
	}

	for _, step := range steps {
		err := client.SaveStep(ctx, session.ID, step.name, step.payload, step.duration)
		if err != nil {
			log.Printf("Failed to save step %s: %v", step.name, err)
		} else {
			fmt.Printf("Saved step: %s\n", step.name)
		}
	}

	// Save final result
	finalResult := map[string]interface{}{
		"lesson": "Complete lesson on machine learning basics",
		"images": map[string]string{
			"intro_image": "path/to/intro.jpg",
			"concept_diagram": "path/to/diagram.png",
		},
		"summary": "This lesson covers the fundamentals of machine learning",
		"duration_minutes": 30,
		"difficulty": "beginner",
	}

	err = client.SaveFinal(ctx, session.ID, finalResult)
	if err != nil {
		log.Printf("Failed to save final result: %v", err)
	} else {
		fmt.Printf("Saved final result for session: %s\n", session.ID)
	}

	// Retrieve the session
	retrievedSession, err := client.GetSession(ctx, session.ID)
	if err != nil {
		log.Printf("Failed to retrieve session: %v", err)
	} else {
		fmt.Printf("Retrieved session with %d steps\n", len(retrievedSession.Steps))
		fmt.Printf("Session status: %s\n", retrievedSession.Status)
	}

	// Get session statistics
	stats, err := client.GetSessionStats(ctx)
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
	} else {
		fmt.Printf("Session stats: %+v\n", stats)
	}
}

// ExampleMockUsage demonstrates how to use the mock client for testing
func ExampleMockUsage() {
	ctx := context.Background()

	// Create mock client
	client := NewMockClient()

	// Create a session
	session, err := client.CreateSession(ctx, "test topic")
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	fmt.Printf("Created mock session: %s\n", session.ID)

	// Save a step
	payload := map[string]interface{}{
		"output": "Test step completed",
		"data":   "test data",
	}
	err = client.SaveStep(ctx, session.ID, "test-step", payload, 1000)
	if err != nil {
		log.Printf("Failed to save step: %v", err)
	} else {
		fmt.Printf("Saved step to mock session\n")
	}

	// Save final result
	finalResult := map[string]interface{}{
		"result": "Test completed successfully",
	}
	err = client.SaveFinal(ctx, session.ID, finalResult)
	if err != nil {
		log.Printf("Failed to save final result: %v", err)
	} else {
		fmt.Printf("Saved final result to mock session\n")
	}

	// Retrieve session
	retrievedSession, err := client.GetSession(ctx, session.ID)
	if err != nil {
		log.Printf("Failed to retrieve session: %v", err)
	} else {
		fmt.Printf("Retrieved mock session with status: %s\n", retrievedSession.Status)
	}

	// Get session count
	fmt.Printf("Total sessions in mock storage: %d\n", client.GetSessionCount())

	// Clear mock storage
	client.Clear()
	fmt.Printf("Cleared mock storage\n")
}

// ExampleWithInterface demonstrates using the storage interface
func ExampleWithInterface() {
	ctx := context.Background()

	// Use interface to work with either implementation
	var client Client

	// Choose implementation based on environment
	useMock := true // In real code, this would be based on configuration

	if useMock {
		client = NewMockClient()
	} else {
		firestoreClient, err := NewFirestoreClient(ctx, "your-project-id", "sessions")
		if err != nil {
			log.Fatalf("Failed to create Firestore client: %v", err)
		}
		client = firestoreClient
		defer firestoreClient.Close()
	}

	// Use the client through the interface
	session, err := client.CreateSession(ctx, "interface test")
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	fmt.Printf("Created session using interface: %s\n", session.ID)

	// Save a step
	err = client.SaveStep(ctx, session.ID, "interface-step", map[string]interface{}{
		"output": "Step completed via interface",
	}, 1500)
	if err != nil {
		log.Printf("Failed to save step: %v", err)
	}

	// Save final result
	err = client.SaveFinal(ctx, session.ID, map[string]interface{}{
		"result": "Interface test completed",
	})
	if err != nil {
		log.Printf("Failed to save final result: %v", err)
	}

	// Get session
	retrievedSession, err := client.GetSession(ctx, session.ID)
	if err != nil {
		log.Printf("Failed to retrieve session: %v", err)
	} else {
		fmt.Printf("Retrieved session via interface: %s\n", retrievedSession.Status)
	}
}

// ExampleErrorHandling demonstrates proper error handling
func ExampleErrorHandling() {
	ctx := context.Background()
	client := NewMockClient()

	// Try to get a non-existent session
	_, err := client.GetSession(ctx, "non-existent-id")
	if err != nil {
		fmt.Printf("Expected error for non-existent session: %v\n", err)
	}

	// Try to save a step to a non-existent session
	err = client.SaveStep(ctx, "non-existent-id", "test-step", nil, 1000)
	if err != nil {
		fmt.Printf("Expected error for non-existent session: %v\n", err)
	}

	// Try to save final result to a non-existent session
	err = client.SaveFinal(ctx, "non-existent-id", nil)
	if err != nil {
		fmt.Printf("Expected error for non-existent session: %v\n", err)
	}

	// Try to update status of a non-existent session
	err = client.UpdateSessionStatus(ctx, "non-existent-id", "running")
	if err != nil {
		fmt.Printf("Expected error for non-existent session: %v\n", err)
	}

	// Try to delete a non-existent session
	err = client.DeleteSession(ctx, "non-existent-id")
	if err != nil {
		fmt.Printf("Expected error for non-existent session: %v\n", err)
	}
}

// ExampleConcurrentUsage demonstrates concurrent usage patterns
func ExampleConcurrentUsage() {
	ctx := context.Background()
	client := NewMockClient()

	// Create a session
	session, err := client.CreateSession(ctx, "concurrent test")
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	// Simulate concurrent step saving
	done := make(chan error, 3)

	go func() {
		err := client.SaveStep(ctx, session.ID, "step1", map[string]interface{}{
			"output": "Step 1 completed",
		}, 1000)
		done <- err
	}()

	go func() {
		err := client.SaveStep(ctx, session.ID, "step2", map[string]interface{}{
			"output": "Step 2 completed",
		}, 2000)
		done <- err
	}()

	go func() {
		err := client.SaveStep(ctx, session.ID, "step3", map[string]interface{}{
			"output": "Step 3 completed",
		}, 1500)
		done <- err
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		err := <-done
		if err != nil {
			log.Printf("Step %d failed: %v", i+1, err)
		}
	}

	// Retrieve session and verify all steps were saved
	retrievedSession, err := client.GetSession(ctx, session.ID)
	if err != nil {
		log.Printf("Failed to retrieve session: %v", err)
	} else {
		fmt.Printf("Session has %d steps\n", len(retrievedSession.Steps))
	}
}

// ExampleHealthCheck demonstrates health checking
func ExampleHealthCheck() {
	ctx := context.Background()
	client := NewMockClient()

	// Check health
	err := client.Health(ctx)
	if err != nil {
		log.Printf("Health check failed: %v", err)
	} else {
		fmt.Println("Storage is healthy")
	}

	// Get statistics
	stats, err := client.GetSessionStats(ctx)
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
	} else {
		fmt.Printf("Storage stats: %+v\n", stats)
	}
}
