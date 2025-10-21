package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMockClient tests the mock client implementation
func TestMockClient(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	t.Run("CreateSession", func(t *testing.T) {
		session, err := client.CreateSession(ctx, "test topic")
		require.NoError(t, err)
		require.NotNil(t, session)

		assert.NotEmpty(t, session.ID)
		assert.Equal(t, "test topic", session.Topic)
		assert.Equal(t, "created", session.Status)
		assert.False(t, session.CreatedAt.IsZero())
		assert.False(t, session.UpdatedAt.IsZero())
		assert.NotNil(t, session.Steps)
		assert.NotNil(t, session.Metadata)

		// Verify session is stored
		assert.Equal(t, 1, client.GetSessionCount())
	})

	t.Run("GetSession", func(t *testing.T) {
		// Create a session first
		createdSession, err := client.CreateSession(ctx, "get test")
		require.NoError(t, err)

		// Retrieve the session
		retrievedSession, err := client.GetSession(ctx, createdSession.ID)
		require.NoError(t, err)
		require.NotNil(t, retrievedSession)

		assert.Equal(t, createdSession.ID, retrievedSession.ID)
		assert.Equal(t, createdSession.Topic, retrievedSession.Topic)
		assert.Equal(t, createdSession.Status, retrievedSession.Status)
	})

	t.Run("GetSessionNotFound", func(t *testing.T) {
		_, err := client.GetSession(ctx, "non-existent-id")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("SaveStep", func(t *testing.T) {
		// Create a session first
		session, err := client.CreateSession(ctx, "step test")
		require.NoError(t, err)

		// Save a step
		payload := map[string]interface{}{
			"output": "Step completed successfully",
			"data":   "test data",
		}
		err = client.SaveStep(ctx, session.ID, "test-step", payload, 1500)
		require.NoError(t, err)

		// Retrieve session and verify step was saved
		retrievedSession, err := client.GetSession(ctx, session.ID)
		require.NoError(t, err)

		assert.Len(t, retrievedSession.Steps, 1)
		step := retrievedSession.Steps[0]
		assert.Equal(t, "test-step", step.Name)
		assert.Equal(t, "completed", step.Status)
		assert.Equal(t, 1500, step.Duration)
		assert.Equal(t, payload, step.Payload)
		assert.False(t, step.StartedAt.IsZero())
		assert.False(t, step.UpdatedAt.IsZero())
	})

	t.Run("SaveStepWithNonMapPayload", func(t *testing.T) {
		// Create a session first
		session, err := client.CreateSession(ctx, "non-map payload test")
		require.NoError(t, err)

		// Save a step with non-map payload
		err = client.SaveStep(ctx, session.ID, "test-step", "simple string payload", 1000)
		require.NoError(t, err)

		// Retrieve session and verify step was saved
		retrievedSession, err := client.GetSession(ctx, session.ID)
		require.NoError(t, err)

		assert.Len(t, retrievedSession.Steps, 1)
		step := retrievedSession.Steps[0]
		assert.Equal(t, "test-step", step.Name)
		assert.Equal(t, "simple string payload", step.Payload["data"])
	})

	t.Run("SaveStepSessionNotFound", func(t *testing.T) {
		err := client.SaveStep(ctx, "non-existent-id", "test-step", nil, 1000)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("SaveFinal", func(t *testing.T) {
		// Create a session first
		session, err := client.CreateSession(ctx, "final test")
		require.NoError(t, err)

		// Save final result
		finalResult := map[string]interface{}{
			"lesson": "Complete lesson content",
			"images": map[string]string{
				"image1": "path/to/image1.jpg",
				"image2": "path/to/image2.jpg",
			},
			"summary": "Lesson summary",
		}
		err = client.SaveFinal(ctx, session.ID, finalResult)
		require.NoError(t, err)

		// Retrieve session and verify final result was saved
		retrievedSession, err := client.GetSession(ctx, session.ID)
		require.NoError(t, err)

		assert.Equal(t, "completed", retrievedSession.Status)
		assert.Equal(t, finalResult, retrievedSession.Final)
	})

	t.Run("SaveFinalWithNonMapResult", func(t *testing.T) {
		// Create a session first
		session, err := client.CreateSession(ctx, "non-map final test")
		require.NoError(t, err)

		// Save final result with non-map data
		err = client.SaveFinal(ctx, session.ID, "simple final result")
		require.NoError(t, err)

		// Retrieve session and verify final result was saved
		retrievedSession, err := client.GetSession(ctx, session.ID)
		require.NoError(t, err)

		assert.Equal(t, "completed", retrievedSession.Status)
		assert.Equal(t, "simple final result", retrievedSession.Final["result"])
	})

	t.Run("SaveFinalSessionNotFound", func(t *testing.T) {
		err := client.SaveFinal(ctx, "non-existent-id", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("UpdateSessionStatus", func(t *testing.T) {
		// Create a session first
		session, err := client.CreateSession(ctx, "status test")
		require.NoError(t, err)

		// Update status
		err = client.UpdateSessionStatus(ctx, session.ID, "running")
		require.NoError(t, err)

		// Retrieve session and verify status was updated
		retrievedSession, err := client.GetSession(ctx, session.ID)
		require.NoError(t, err)

		assert.Equal(t, "running", retrievedSession.Status)
	})

	t.Run("UpdateSessionStatusNotFound", func(t *testing.T) {
		err := client.UpdateSessionStatus(ctx, "non-existent-id", "running")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("DeleteSession", func(t *testing.T) {
		// Create a session first
		session, err := client.CreateSession(ctx, "delete test")
		require.NoError(t, err)

		// Verify session exists
		_, err = client.GetSession(ctx, session.ID)
		require.NoError(t, err)

		// Delete session
		err = client.DeleteSession(ctx, session.ID)
		require.NoError(t, err)

		// Verify session no longer exists
		_, err = client.GetSession(ctx, session.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("DeleteSessionNotFound", func(t *testing.T) {
		err := client.DeleteSession(ctx, "non-existent-id")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("ListSessions", func(t *testing.T) {
		// Clear existing sessions
		client.Clear()

		// Create multiple sessions
		_, err := client.CreateSession(ctx, "session 1")
		require.NoError(t, err)
		_, err = client.CreateSession(ctx, "session 2")
		require.NoError(t, err)
		_, err = client.CreateSession(ctx, "session 3")
		require.NoError(t, err)

		// List sessions
		sessions, err := client.ListSessions(ctx, 10)
		require.NoError(t, err)

		assert.Len(t, sessions, 3)
	})

	t.Run("ListSessionsWithLimit", func(t *testing.T) {
		// Clear existing sessions
		client.Clear()

		// Create multiple sessions
		for i := 0; i < 5; i++ {
			_, err := client.CreateSession(ctx, fmt.Sprintf("session %d", i))
			require.NoError(t, err)
		}

		// List sessions with limit
		sessions, err := client.ListSessions(ctx, 3)
		require.NoError(t, err)

		assert.Len(t, sessions, 3)
	})

	t.Run("GetSessionStats", func(t *testing.T) {
		// Clear existing sessions
		client.Clear()

		// Create sessions with different statuses
		session1, err := client.CreateSession(ctx, "session 1")
		require.NoError(t, err)
		client.UpdateSessionStatus(ctx, session1.ID, "running")

		session2, err := client.CreateSession(ctx, "session 2")
		require.NoError(t, err)
		client.SaveFinal(ctx, session2.ID, map[string]interface{}{"result": "completed"})

		// Get stats
		stats, err := client.GetSessionStats(ctx)
		require.NoError(t, err)

		assert.Equal(t, 2, stats["total_sessions"])
		assert.Equal(t, "mock", stats["storage_type"])
		assert.NotNil(t, stats["status_counts"])
		assert.NotNil(t, stats["timestamp"])
	})

	t.Run("Health", func(t *testing.T) {
		err := client.Health(ctx)
		require.NoError(t, err)
	})

	t.Run("Close", func(t *testing.T) {
		err := client.Close()
		require.NoError(t, err)
	})

	t.Run("Clear", func(t *testing.T) {
		// Clear any existing sessions first
		client.Clear()

		// Create some sessions
		_, err := client.CreateSession(ctx, "test 1")
		require.NoError(t, err)
		_, err = client.CreateSession(ctx, "test 2")
		require.NoError(t, err)

		assert.Equal(t, 2, client.GetSessionCount())

		// Clear all sessions
		client.Clear()

		assert.Equal(t, 0, client.GetSessionCount())
	})

	t.Run("GetAllSessions", func(t *testing.T) {
		// Clear existing sessions
		client.Clear()

		// Create some sessions
		session1, err := client.CreateSession(ctx, "test 1")
		require.NoError(t, err)
		session2, err := client.CreateSession(ctx, "test 2")
		require.NoError(t, err)

		// Get all sessions
		allSessions := client.GetAllSessions()

		assert.Len(t, allSessions, 2)
		assert.Contains(t, allSessions, session1.ID)
		assert.Contains(t, allSessions, session2.ID)
	})
}

// TestMockClientConcurrency tests concurrent access to the mock client
func TestMockClientConcurrency(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	// Test concurrent session creation
	t.Run("ConcurrentSessionCreation", func(t *testing.T) {
		const numGoroutines = 10
		done := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(i int) {
				_, err := client.CreateSession(ctx, "concurrent test")
				done <- err
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			err := <-done
			require.NoError(t, err)
		}

		assert.Equal(t, numGoroutines, client.GetSessionCount())
	})

	// Test concurrent step saving
	t.Run("ConcurrentStepSaving", func(t *testing.T) {
		// Create a session
		session, err := client.CreateSession(ctx, "concurrent steps")
		require.NoError(t, err)

		const numSteps = 5
		done := make(chan error, numSteps)

		for i := 0; i < numSteps; i++ {
			go func(i int) {
				payload := map[string]interface{}{
					"step_number": i,
					"data":        "concurrent step data",
				}
				err := client.SaveStep(ctx, session.ID, "step-"+string(rune(i)), payload, 1000)
				done <- err
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numSteps; i++ {
			err := <-done
			require.NoError(t, err)
		}

		// Verify all steps were saved
		retrievedSession, err := client.GetSession(ctx, session.ID)
		require.NoError(t, err)
		assert.Len(t, retrievedSession.Steps, numSteps)
	})
}

// TestSessionDataIntegrity tests that session data is properly copied to avoid race conditions
func TestSessionDataIntegrity(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	// Create a session
	session, err := client.CreateSession(ctx, "integrity test")
	require.NoError(t, err)

	// Add some steps
	err = client.SaveStep(ctx, session.ID, "step1", map[string]interface{}{"data": "step1"}, 1000)
	require.NoError(t, err)
	err = client.SaveStep(ctx, session.ID, "step2", map[string]interface{}{"data": "step2"}, 2000)
	require.NoError(t, err)

	// Get session multiple times and verify data integrity
	session1, err := client.GetSession(ctx, session.ID)
	require.NoError(t, err)
	session2, err := client.GetSession(ctx, session.ID)
	require.NoError(t, err)

	// Modify one copy
	session1.Steps[0].Name = "modified"

	// Verify the other copy is unchanged
	assert.Equal(t, "step1", session2.Steps[0].Name)
	assert.Equal(t, "step2", session2.Steps[1].Name)
}

// Benchmark tests
func BenchmarkMockClientCreateSession(b *testing.B) {
	client := NewMockClient()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.CreateSession(ctx, "benchmark test")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMockClientGetSession(b *testing.B) {
	client := NewMockClient()
	ctx := context.Background()

	// Create a session first
	session, err := client.CreateSession(ctx, "benchmark test")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.GetSession(ctx, session.ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMockClientSaveStep(b *testing.B) {
	client := NewMockClient()
	ctx := context.Background()

	// Create a session first
	session, err := client.CreateSession(ctx, "benchmark test")
	if err != nil {
		b.Fatal(err)
	}

	payload := map[string]interface{}{
		"data": "benchmark step data",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := client.SaveStep(ctx, session.ID, "benchmark-step", payload, 1000)
		if err != nil {
			b.Fatal(err)
		}
	}
}
