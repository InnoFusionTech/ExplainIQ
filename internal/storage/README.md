# Storage Package

The storage package provides a unified interface for persisting learning sessions and their associated data. It includes both Firestore and in-memory mock implementations for development and testing.

## Features

- **Firestore Integration**: Production-ready Firestore client for Google Cloud
- **Mock Implementation**: In-memory storage for testing and development
- **Unified Interface**: Common interface for both implementations
- **Session Management**: Create, retrieve, update, and delete sessions
- **Step Tracking**: Save and track workflow steps with payloads
- **Final Results**: Store and retrieve final session results
- **Concurrent Safe**: Thread-safe operations for both implementations
- **Health Monitoring**: Health checks and statistics
- **Comprehensive Testing**: Full test coverage with benchmarks

## Data Structures

### Session
```go
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
```

### SessionStep
```go
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
```

## Interface

### Client Interface
```go
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
```

## Usage

### Firestore Client

```go
package main

import (
    "context"
    "log"
    
    "github.com/creduntvitam/explainiq/internal/storage"
)

func main() {
    ctx := context.Background()

    // Create Firestore client
    client, err := storage.NewFirestoreClient(ctx, "your-project-id", "sessions")
    if err != nil {
        log.Fatalf("Failed to create Firestore client: %v", err)
    }
    defer client.Close()

    // Create a session
    session, err := client.CreateSession(ctx, "machine learning basics")
    if err != nil {
        log.Fatalf("Failed to create session: %v", err)
    }

    // Save a step
    payload := map[string]interface{}{
        "output": "Analyzed topic: machine learning basics",
        "keywords": []string{"machine learning", "algorithms", "data"},
    }
    err = client.SaveStep(ctx, session.ID, "analyze-topic", payload, 2000)
    if err != nil {
        log.Printf("Failed to save step: %v", err)
    }

    // Save final result
    finalResult := map[string]interface{}{
        "lesson": "Complete lesson on machine learning basics",
        "images": map[string]string{
            "intro_image": "path/to/intro.jpg",
            "concept_diagram": "path/to/diagram.png",
        },
        "summary": "This lesson covers the fundamentals of machine learning",
    }
    err = client.SaveFinal(ctx, session.ID, finalResult)
    if err != nil {
        log.Printf("Failed to save final result: %v", err)
    }

    // Retrieve the session
    retrievedSession, err := client.GetSession(ctx, session.ID)
    if err != nil {
        log.Printf("Failed to retrieve session: %v", err)
    } else {
        log.Printf("Retrieved session with %d steps", len(retrievedSession.Steps))
    }
}
```

### Mock Client (for testing)

```go
package main

import (
    "context"
    "log"
    
    "github.com/creduntvitam/explainiq/internal/storage"
)

func main() {
    ctx := context.Background()

    // Create mock client
    client := storage.NewMockClient()

    // Create a session
    session, err := client.CreateSession(ctx, "test topic")
    if err != nil {
        log.Fatalf("Failed to create session: %v", err)
    }

    // Save a step
    payload := map[string]interface{}{
        "output": "Test step completed",
        "data":   "test data",
    }
    err = client.SaveStep(ctx, session.ID, "test-step", payload, 1000)
    if err != nil {
        log.Printf("Failed to save step: %v", err)
    }

    // Save final result
    finalResult := map[string]interface{}{
        "result": "Test completed successfully",
    }
    err = client.SaveFinal(ctx, session.ID, finalResult)
    if err != nil {
        log.Printf("Failed to save final result: %v", err)
    }

    // Get session count
    log.Printf("Total sessions: %d", client.GetSessionCount())

    // Clear storage
    client.Clear()
}
```

### Using the Interface

```go
package main

import (
    "context"
    "log"
    
    "github.com/creduntvitam/explainiq/internal/storage"
)

func processSession(client storage.Client, ctx context.Context, topic string) error {
    // Create session
    session, err := client.CreateSession(ctx, topic)
    if err != nil {
        return err
    }

    // Save steps
    steps := []struct {
        name     string
        payload  map[string]interface{}
        duration int
    }{
        {"analyze-topic", map[string]interface{}{"output": "Analyzed topic"}, 2000},
        {"generate-outline", map[string]interface{}{"output": "Generated outline"}, 1500},
        {"create-content", map[string]interface{}{"output": "Created content"}, 5000},
    }

    for _, step := range steps {
        err := client.SaveStep(ctx, session.ID, step.name, step.payload, step.duration)
        if err != nil {
            return err
        }
    }

    // Save final result
    finalResult := map[string]interface{}{
        "lesson": "Complete lesson",
        "summary": "Lesson summary",
    }
    return client.SaveFinal(ctx, session.ID, finalResult)
}

func main() {
    ctx := context.Background()

    // Choose implementation based on environment
    var client storage.Client
    if useMock {
        client = storage.NewMockClient()
    } else {
        firestoreClient, err := storage.NewFirestoreClient(ctx, "project-id", "sessions")
        if err != nil {
            log.Fatal(err)
        }
        client = firestoreClient
        defer firestoreClient.Close()
    }

    // Use the client through the interface
    err := processSession(client, ctx, "machine learning")
    if err != nil {
        log.Fatal(err)
    }
}
```

## Configuration

### Firestore Setup

1. **Enable Firestore API** in your Google Cloud project
2. **Set up authentication** using one of:
   - Service account key file
   - Application Default Credentials (ADC)
   - Workload Identity (for GKE)
3. **Configure permissions** for Firestore access

### Environment Variables

```bash
# For Firestore
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"
export GOOGLE_CLOUD_PROJECT="your-project-id"

# Or use Application Default Credentials
gcloud auth application-default login
```

## Error Handling

The storage package provides comprehensive error handling:

```go
// Check for specific error types
session, err := client.GetSession(ctx, sessionID)
if err != nil {
    if strings.Contains(err.Error(), "not found") {
        // Handle session not found
        log.Printf("Session %s not found", sessionID)
    } else {
        // Handle other errors
        log.Printf("Failed to get session: %v", err)
    }
}
```

## Testing

### Running Tests

```bash
# Run all tests
go test ./internal/storage

# Run with verbose output
go test -v ./internal/storage

# Run specific test
go test -run TestMockClient ./internal/storage

# Run benchmarks
go test -bench=. ./internal/storage
```

### Test Coverage

```bash
# Generate coverage report
go test -cover ./internal/storage

# Generate HTML coverage report
go test -coverprofile=coverage.out ./internal/storage
go tool cover -html=coverage.out
```

## Performance

### Benchmarks

The package includes comprehensive benchmarks:

- `BenchmarkMockClientCreateSession`: Session creation performance
- `BenchmarkMockClientGetSession`: Session retrieval performance
- `BenchmarkMockClientSaveStep`: Step saving performance

### Optimization Tips

1. **Use transactions** for atomic operations (Firestore)
2. **Batch operations** when possible
3. **Use appropriate indexes** for queries
4. **Monitor costs** in production (Firestore charges per operation)

## Security Considerations

### Firestore Security Rules

```javascript
// Example Firestore security rules
rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
    match /sessions/{sessionId} {
      allow read, write: if request.auth != null;
    }
  }
}
```

### Data Validation

- Input validation is performed on all operations
- UUIDs are used for session IDs to prevent enumeration
- Timestamps are automatically managed
- Payload data is stored as-is (validate at application level)

## Monitoring

### Health Checks

```go
// Check storage health
err := client.Health(ctx)
if err != nil {
    log.Printf("Storage health check failed: %v", err)
}
```

### Statistics

```go
// Get session statistics
stats, err := client.GetSessionStats(ctx)
if err != nil {
    log.Printf("Failed to get stats: %v", err)
} else {
    log.Printf("Total sessions: %v", stats["total_sessions"])
    log.Printf("Status counts: %v", stats["status_counts"])
}
```

## Dependencies

- **cloud.google.com/go/firestore**: Google Cloud Firestore client
- **github.com/google/uuid**: UUID generation
- **github.com/sirupsen/logrus**: Structured logging
- **github.com/stretchr/testify**: Testing framework

## Future Enhancements

- **Caching Layer**: Redis integration for improved performance
- **Backup/Restore**: Automated backup and restore functionality
- **Metrics**: Prometheus metrics integration
- **Encryption**: Field-level encryption for sensitive data
- **Audit Logging**: Comprehensive audit trail
- **Multi-region**: Cross-region replication support




