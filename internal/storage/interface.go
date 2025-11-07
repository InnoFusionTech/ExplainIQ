package storage

import (
	"context"
)

// Client defines the interface for storage operations
type Client interface {
	// CreateSession creates a new session
	CreateSession(ctx context.Context, topic string) (*Session, error)

	// SaveStep saves a step to the session
	SaveStep(ctx context.Context, sessionID, step string, payload interface{}, durationMs int) error

	// SaveFinal saves the final result to the session
	SaveFinal(ctx context.Context, sessionID string, final interface{}) error

	// GetSession retrieves a session by ID
	GetSession(ctx context.Context, sessionID string) (*Session, error)

	// ListSessions retrieves all sessions with a limit
	ListSessions(ctx context.Context, limit int) ([]*Session, error)

	// UpdateSessionStatus updates the status of a session
	UpdateSessionStatus(ctx context.Context, sessionID, status string) error

	// DeleteSession deletes a session
	DeleteSession(ctx context.Context, sessionID string) error

	// Close closes the client connection
	Close() error

	// Health checks the health of the storage
	Health(ctx context.Context) error

	// GetSessionStats returns statistics about sessions
	GetSessionStats(ctx context.Context) (map[string]interface{}, error)
}

// Storage defines the interface for key-value storage operations
type Storage interface {
	// Get retrieves a value by key
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value by key
	Set(ctx context.Context, key string, value []byte) error

	// Delete removes a value by key
	Delete(ctx context.Context, key string) error

	// Close closes the storage connection
	Close() error
}

// MockClientInterface extends the Client interface with mock-specific methods
type MockClientInterface interface {
	Client

	// Clear clears all sessions from the mock storage
	Clear()

	// GetSessionCount returns the number of sessions in mock storage
	GetSessionCount() int

	// GetAllSessions returns all sessions (for testing purposes)
	GetAllSessions() map[string]*Session
}
