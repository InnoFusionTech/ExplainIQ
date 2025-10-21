package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// MockClient represents an in-memory mock implementation of the storage client
type MockClient struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	logger   *logrus.Logger
}

// NewMockClient creates a new mock storage client
func NewMockClient() *MockClient {
	return &MockClient{
		sessions: make(map[string]*Session),
		logger:   logrus.New(),
	}
}

// CreateSession creates a new session in memory
func (m *MockClient) CreateSession(ctx context.Context, topic string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sessionID := uuid.New().String()
	now := time.Now()

	session := &Session{
		ID:        sessionID,
		Topic:     topic,
		Status:    "created",
		CreatedAt: now,
		UpdatedAt: now,
		Steps:     make([]SessionStep, 0),
		Metadata:  make(map[string]interface{}),
	}

	m.sessions[sessionID] = session

	m.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"topic":      topic,
	}).Info("Session created in mock storage")

	return session, nil
}

// SaveStep saves a step to the session in memory
func (m *MockClient) SaveStep(ctx context.Context, sessionID, step string, payload interface{}, durationMs int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	now := time.Now()

	// Create step data
	stepData := SessionStep{
		ID:        uuid.New().String(),
		Name:      step,
		Status:    "completed",
		Duration:  durationMs,
		StartedAt: now.Add(-time.Duration(durationMs) * time.Millisecond),
		UpdatedAt: now,
	}

	// Convert payload to map if it's not already
	if payload != nil {
		stepData.Payload = make(map[string]interface{})
		if payloadMap, ok := payload.(map[string]interface{}); ok {
			stepData.Payload = payloadMap
		} else {
			stepData.Payload["data"] = payload
		}
	}

	// Add step to session
	session.Steps = append(session.Steps, stepData)
	session.UpdatedAt = now

	m.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"step":       step,
		"duration_ms": durationMs,
	}).Info("Step saved to mock storage")

	return nil
}

// SaveFinal saves the final result to the session in memory
func (m *MockClient) SaveFinal(ctx context.Context, sessionID string, final interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	now := time.Now()

	// Convert final to map if it's not already
	finalData := make(map[string]interface{})
	if final != nil {
		if finalMap, ok := final.(map[string]interface{}); ok {
			finalData = finalMap
		} else {
			finalData["result"] = final
		}
	}

	// Update session with final result
	session.Final = finalData
	session.Status = "completed"
	session.UpdatedAt = now

	m.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
	}).Info("Final result saved to mock storage")

	return nil
}

// GetSession retrieves a session from memory
func (m *MockClient) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	// Return a copy to avoid race conditions
	sessionCopy := *session
	sessionCopy.Steps = make([]SessionStep, len(session.Steps))
	copy(sessionCopy.Steps, session.Steps)

	if session.Final != nil {
		sessionCopy.Final = make(map[string]interface{})
		for k, v := range session.Final {
			sessionCopy.Final[k] = v
		}
	}

	if session.Metadata != nil {
		sessionCopy.Metadata = make(map[string]interface{})
		for k, v := range session.Metadata {
			sessionCopy.Metadata[k] = v
		}
	}

	m.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"status":     session.Status,
		"steps_count": len(session.Steps),
	}).Debug("Session retrieved from mock storage")

	return &sessionCopy, nil
}

// ListSessions retrieves all sessions from memory
func (m *MockClient) ListSessions(ctx context.Context, limit int) ([]*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sessions []*Session
	count := 0

	// Simple implementation - in a real scenario, you'd want to sort by created_at
	for _, session := range m.sessions {
		if count >= limit {
			break
		}
		sessions = append(sessions, session)
		count++
	}

	m.logger.WithFields(logrus.Fields{
		"count": len(sessions),
	}).Debug("Sessions listed from mock storage")

	return sessions, nil
}

// UpdateSessionStatus updates the status of a session
func (m *MockClient) UpdateSessionStatus(ctx context.Context, sessionID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.Status = status
	session.UpdatedAt = time.Now()

	m.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"status":     status,
	}).Info("Session status updated in mock storage")

	return nil
}

// DeleteSession deletes a session from memory
func (m *MockClient) DeleteSession(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	delete(m.sessions, sessionID)

	m.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
	}).Info("Session deleted from mock storage")

	return nil
}

// Close closes the mock client (no-op for in-memory)
func (m *MockClient) Close() error {
	return nil
}

// Health checks the health of the mock storage (always healthy)
func (m *MockClient) Health(ctx context.Context) error {
	return nil
}

// GetSessionStats returns statistics about sessions
func (m *MockClient) GetSessionStats(ctx context.Context) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalCount := len(m.sessions)
	statusCounts := make(map[string]int)

	for _, session := range m.sessions {
		statusCounts[session.Status]++
	}

	stats := map[string]interface{}{
		"total_sessions": totalCount,
		"status_counts":  statusCounts,
		"storage_type":   "mock",
		"timestamp":      time.Now(),
	}

	return stats, nil
}

// Clear clears all sessions from the mock storage (useful for testing)
func (m *MockClient) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessions = make(map[string]*Session)
	m.logger.Info("Mock storage cleared")
}

// GetSessionCount returns the number of sessions in mock storage
func (m *MockClient) GetSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.sessions)
}

// GetAllSessions returns all sessions (for testing purposes)
func (m *MockClient) GetAllSessions() map[string]*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid race conditions
	sessions := make(map[string]*Session)
	for id, session := range m.sessions {
		sessionCopy := *session
		sessionCopy.Steps = make([]SessionStep, len(session.Steps))
		copy(sessionCopy.Steps, session.Steps)

		if session.Final != nil {
			sessionCopy.Final = make(map[string]interface{})
			for k, v := range session.Final {
				sessionCopy.Final[k] = v
			}
		}

		if session.Metadata != nil {
			sessionCopy.Metadata = make(map[string]interface{})
			for k, v := range session.Metadata {
				sessionCopy.Metadata[k] = v
			}
		}

		sessions[id] = &sessionCopy
	}

	return sessions
}




