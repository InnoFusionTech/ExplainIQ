package storage

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Session represents a learning session stored in Firestore
type Session struct {
	ID        string                 `firestore:"id"`
	Topic     string                 `firestore:"topic"`
	Status    string                 `firestore:"status"`
	CreatedAt time.Time              `firestore:"created_at"`
	UpdatedAt time.Time              `firestore:"updated_at"`
	Steps     []SessionStep          `firestore:"steps"`
	Final     map[string]interface{} `firestore:"final,omitempty"`
	Metadata  map[string]interface{} `firestore:"metadata,omitempty"`
}

// SessionStep represents a step in the session workflow
type SessionStep struct {
	ID        string                 `firestore:"id"`
	Name      string                 `firestore:"name"`
	Status    string                 `firestore:"status"`
	Payload   map[string]interface{} `firestore:"payload,omitempty"`
	Duration  int                    `firestore:"duration_ms"`
	StartedAt time.Time              `firestore:"started_at"`
	UpdatedAt time.Time              `firestore:"updated_at"`
	Error     string                 `firestore:"error,omitempty"`
}

// FirestoreClient represents a Firestore storage client
type FirestoreClient struct {
	client     *firestore.Client
	collection string
	logger     *logrus.Logger
}

// NewFirestoreClient creates a new Firestore client
func NewFirestoreClient(ctx context.Context, projectID, collection string) (*FirestoreClient, error) {
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create Firestore client: %w", err)
	}

	return &FirestoreClient{
		client:     client,
		collection: collection,
		logger:     logrus.New(),
	}, nil
}

// NewFirestoreClientWithClient creates a new Firestore client with an existing client
func NewFirestoreClientWithClient(client *firestore.Client, collection string) *FirestoreClient {
	return &FirestoreClient{
		client:     client,
		collection: collection,
		logger:     logrus.New(),
	}
}

// CreateSession creates a new session in Firestore
func (f *FirestoreClient) CreateSession(ctx context.Context, topic string) (*Session, error) {
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

	// Store session in Firestore
	docRef := f.client.Collection(f.collection).Doc(sessionID)
	_, err := docRef.Set(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to create session in Firestore: %w", err)
	}

	f.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"topic":      topic,
		"collection": f.collection,
	}).Info("Session created in Firestore")

	return session, nil
}

// SaveStep saves a step to the session in Firestore
func (f *FirestoreClient) SaveStep(ctx context.Context, sessionID, step string, payload interface{}, durationMs int) error {
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

	// Get session document reference
	docRef := f.client.Collection(f.collection).Doc(sessionID)

	// Use transaction to ensure atomicity
	err := f.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// Get current session
		doc, err := tx.Get(docRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return fmt.Errorf("session %s not found", sessionID)
			}
			return fmt.Errorf("failed to get session: %w", err)
		}

		var session Session
		if err := doc.DataTo(&session); err != nil {
			return fmt.Errorf("failed to parse session data: %w", err)
		}

		// Add step to session
		session.Steps = append(session.Steps, stepData)
		session.UpdatedAt = now

		// Update session in Firestore
		return tx.Set(docRef, session)
	})

	if err != nil {
		return fmt.Errorf("failed to save step: %w", err)
	}

	f.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"step":       step,
		"duration_ms": durationMs,
	}).Info("Step saved to Firestore")

	return nil
}

// SaveFinal saves the final result to the session in Firestore
func (f *FirestoreClient) SaveFinal(ctx context.Context, sessionID string, final interface{}) error {
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

	// Get session document reference
	docRef := f.client.Collection(f.collection).Doc(sessionID)

	// Use transaction to ensure atomicity
	err := f.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// Get current session
		doc, err := tx.Get(docRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return fmt.Errorf("session %s not found", sessionID)
			}
			return fmt.Errorf("failed to get session: %w", err)
		}

		var session Session
		if err := doc.DataTo(&session); err != nil {
			return fmt.Errorf("failed to parse session data: %w", err)
		}

		// Update session with final result
		session.Final = finalData
		session.Status = "completed"
		session.UpdatedAt = now

		// Update session in Firestore
		return tx.Set(docRef, session)
	})

	if err != nil {
		return fmt.Errorf("failed to save final result: %w", err)
	}

	f.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"collection": f.collection,
	}).Info("Final result saved to Firestore")

	return nil
}

// GetSession retrieves a session from Firestore
func (f *FirestoreClient) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	docRef := f.client.Collection(f.collection).Doc(sessionID)
	doc, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("session %s not found", sessionID)
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session Session
	if err := doc.DataTo(&session); err != nil {
		return nil, fmt.Errorf("failed to parse session data: %w", err)
	}

	f.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"status":     session.Status,
		"steps_count": len(session.Steps),
	}).Debug("Session retrieved from Firestore")

	return &session, nil
}

// ListSessions retrieves all sessions from Firestore
func (f *FirestoreClient) ListSessions(ctx context.Context, limit int) ([]*Session, error) {
	query := f.client.Collection(f.collection).OrderBy("created_at", firestore.Desc).Limit(limit)
	iter := query.Documents(ctx)
	defer iter.Stop()

	var sessions []*Session
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate sessions: %w", err)
		}

		var session Session
		if err := doc.DataTo(&session); err != nil {
			return nil, fmt.Errorf("failed to parse session data: %w", err)
		}

		sessions = append(sessions, &session)
	}

	f.logger.WithFields(logrus.Fields{
		"count":      len(sessions),
		"collection": f.collection,
	}).Debug("Sessions listed from Firestore")

	return sessions, nil
}

// UpdateSessionStatus updates the status of a session
func (f *FirestoreClient) UpdateSessionStatus(ctx context.Context, sessionID, status string) error {
	docRef := f.client.Collection(f.collection).Doc(sessionID)
	_, err := docRef.Update(ctx, []firestore.Update{
		{Path: "status", Value: status},
		{Path: "updated_at", Value: time.Now()},
	})
	if err != nil {
		return fmt.Errorf("failed to update session status: %w", err)
	}

	f.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"status":     status,
	}).Info("Session status updated in Firestore")

	return nil
}

// DeleteSession deletes a session from Firestore
func (f *FirestoreClient) DeleteSession(ctx context.Context, sessionID string) error {
	docRef := f.client.Collection(f.collection).Doc(sessionID)
	_, err := docRef.Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	f.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"collection": f.collection,
	}).Info("Session deleted from Firestore")

	return nil
}

// Close closes the Firestore client
func (f *FirestoreClient) Close() error {
	if f.client != nil {
		return f.client.Close()
	}
	return nil
}

// Health checks the health of the Firestore connection
func (f *FirestoreClient) Health(ctx context.Context) error {
	// Try to read from the collection to check connectivity
	iter := f.client.Collection(f.collection).Limit(1).Documents(ctx)
	defer iter.Stop()

	_, err := iter.Next()
	if err != nil && err != iterator.Done {
		return fmt.Errorf("Firestore health check failed: %w", err)
	}

	return nil
}

// GetSessionStats returns statistics about sessions
func (f *FirestoreClient) GetSessionStats(ctx context.Context) (map[string]interface{}, error) {
	// Count total sessions
	totalIter := f.client.Collection(f.collection).Documents(ctx)
	defer totalIter.Stop()

	totalCount := 0
	statusCounts := make(map[string]int)

	for {
		doc, err := totalIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to count sessions: %w", err)
		}

		totalCount++
		var session Session
		if err := doc.DataTo(&session); err == nil {
			statusCounts[session.Status]++
		}
	}

	stats := map[string]interface{}{
		"total_sessions": totalCount,
		"status_counts":  statusCounts,
		"collection":     f.collection,
		"timestamp":      time.Now(),
	}

	return stats, nil
}




