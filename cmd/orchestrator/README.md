# Orchestrator Service

The Orchestrator service manages learning sessions with real-time progress updates via Server-Sent Events (SSE). It provides a RESTful API for creating sessions, streaming execution progress, and retrieving final results.

## Features

- **Session Management**: Create and manage learning sessions
- **Real-time Updates**: Server-Sent Events for live progress streaming
- **Workflow Execution**: Multi-step learning content generation
- **CORS Support**: Development-friendly CORS configuration
- **Graceful Shutdown**: Proper signal handling and cleanup
- **Health Monitoring**: Heartbeat endpoint for service health
- **Chi Router**: Fast, lightweight HTTP router
- **Structured Logging**: Comprehensive logging with logrus

## API Endpoints

### POST /api/sessions
Creates a new learning session.

**Request:**
```json
{
  "topic": "machine learning"
}
```

**Response:**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000"
}
```

### POST /api/sessions/{id}/run
Starts session execution and streams progress via SSE.

**Response:** Server-Sent Events stream with the following event types:
- `connected`: Initial connection confirmation
- `step-start`: A workflow step has started
- `step-delta`: Progress update within a step
- `step-complete`: A workflow step has completed
- `final`: Session execution completed

**Example SSE Events:**
```
data: {"type":"connected","session_id":"123e4567-e89b-12d3-a456-426614174000"}

data: {"type":"step-start","session_id":"123e4567-e89b-12d3-a456-426614174000","step_id":"step-1","data":{"step_name":"analyze-topic","step_index":0},"timestamp":"2023-10-14T23:58:28Z"}

data: {"type":"step-delta","session_id":"123e4567-e89b-12d3-a456-426614174000","step_id":"step-1","data":{"delta":"Processing analyze-topic... 20% complete","progress":20},"timestamp":"2023-10-14T23:58:28Z"}

data: {"type":"step-complete","session_id":"123e4567-e89b-12d3-a456-426614174000","step_id":"step-1","data":{"step_name":"analyze-topic","output":"Completed analyze-topic for topic: machine learning","duration":2500},"timestamp":"2023-10-14T23:58:30Z"}

data: {"type":"final","session_id":"123e4567-e89b-12d3-a456-426614174000","data":{"result":{"lesson":"Complete lesson on machine learning","images":{"image1":"path/to/image1.jpg","image2":"path/to/image2.jpg"},"summary":"Summary of machine learning lesson","duration":12500000000,"completed_at":"2023-10-14T23:58:35Z"}},"timestamp":"2023-10-14T23:58:35Z"}
```

### GET /api/sessions/{id}/result
Retrieves the final result of a completed session.

**Response:**
```json
{
  "lesson": "Complete lesson on machine learning",
  "images": {
    "image1": "path/to/image1.jpg",
    "image2": "path/to/image2.jpg"
  },
  "summary": "Summary of machine learning lesson",
  "duration": 12500000000,
  "completed_at": "2023-10-14T23:58:35Z"
}
```

### GET /health
Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2023-10-14T23:58:35Z",
  "sessions": 1
}
```

## Workflow Steps

The orchestrator executes the following steps for each session:

1. **analyze-topic**: Analyze the input topic and requirements
2. **generate-outline**: Create a structured lesson outline
3. **create-content**: Generate the main lesson content
4. **generate-images**: Create relevant images and visual aids
5. **finalize-lesson**: Finalize and package the complete lesson

Each step includes:
- Progress updates via `step-delta` events
- Completion confirmation via `step-complete` events
- Error handling and reporting
- Duration tracking

## Data Structures

### Session
```go
type Session struct {
    ID        string                 `json:"id"`
    Topic     string                 `json:"topic"`
    Status    string                 `json:"status"`
    CreatedAt time.Time              `json:"created_at"`
    UpdatedAt time.Time              `json:"updated_at"`
    Result    *SessionResult         `json:"result,omitempty"`
    Steps     []SessionStep          `json:"steps,omitempty"`
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
}
```

### SessionResult
```go
type SessionResult struct {
    Lesson      string            `json:"lesson"`
    Images      map[string]string `json:"images"`
    Summary     string            `json:"summary"`
    Duration    time.Duration     `json:"duration"`
    CompletedAt time.Time         `json:"completed_at"`
}
```

### SessionStep
```go
type SessionStep struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Status      string                 `json:"status"`
    StartedAt   *time.Time             `json:"started_at,omitempty"`
    CompletedAt *time.Time             `json:"completed_at,omitempty"`
    Duration    *time.Duration         `json:"duration,omitempty"`
    Output      string                 `json:"output,omitempty"`
    Error       string                 `json:"error,omitempty"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
```

### SSEEvent
```go
type SSEEvent struct {
    Type      string                 `json:"type"`
    SessionID string                 `json:"session_id"`
    StepID    string                 `json:"step_id,omitempty"`
    Data      map[string]interface{} `json:"data"`
    Timestamp time.Time              `json:"timestamp"`
}
```

## Running the Service

### Development
```bash
go run ./cmd/orchestrator
```

### Production
```bash
go build -o orchestrator ./cmd/orchestrator
./orchestrator
```

The service will start on `http://localhost:8080` by default.

## Configuration

The service uses the following default configuration:
- **Port**: 8080
- **Timeout**: 60 seconds per request
- **CORS**: Enabled for all origins (development)
- **Logging**: Structured logging with logrus

## Error Handling

- **404**: Session not found
- **400**: Invalid request or session not completed
- **409**: Session already running
- **500**: Internal server error

## CORS Configuration

The service includes CORS middleware for development:
- `Access-Control-Allow-Origin: *`
- `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`
- `Access-Control-Allow-Headers: Accept, Authorization, Content-Type, X-CSRF-Token`
- `Access-Control-Expose-Headers: Link`

## Graceful Shutdown

The service handles SIGINT and SIGTERM signals for graceful shutdown:
- Stops accepting new requests
- Waits for ongoing requests to complete (30-second timeout)
- Closes all connections
- Exits cleanly

## Testing

Run the test suite:
```bash
go test ./cmd/orchestrator
```

Run specific tests:
```bash
go test ./cmd/orchestrator -run TestCreateSession
```

Run with verbose output:
```bash
go test -v ./cmd/orchestrator
```

## Client Examples

### JavaScript/TypeScript
```javascript
// Create session
const response = await fetch('/api/sessions', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ topic: 'machine learning' })
});
const { id } = await response.json();

// Stream progress
const eventSource = new EventSource(`/api/sessions/${id}/run`);
eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Event:', data.type, data.data);
  
  if (data.type === 'final') {
    eventSource.close();
  }
};
```

### Python
```python
import requests
import json

# Create session
response = requests.post('http://localhost:8080/api/sessions', 
                        json={'topic': 'machine learning'})
session_id = response.json()['id']

# Stream progress
response = requests.post(f'http://localhost:8080/api/sessions/{session_id}/run',
                        stream=True)

for line in response.iter_lines():
    if line.startswith(b'data: '):
        data = json.loads(line[6:])
        print(f"Event: {data['type']}")
        
        if data['type'] == 'final':
            break
```

### cURL
```bash
# Create session
curl -X POST http://localhost:8080/api/sessions \
  -H "Content-Type: application/json" \
  -d '{"topic": "machine learning"}'

# Stream progress
curl -X POST http://localhost:8080/api/sessions/{id}/run

# Get result
curl http://localhost:8080/api/sessions/{id}/result
```

## Dependencies

- **chi/v5**: HTTP router
- **chi/cors**: CORS middleware
- **logrus**: Structured logging
- **uuid**: UUID generation

## Architecture

The orchestrator follows a clean architecture pattern:

1. **HTTP Layer**: Chi router with middleware
2. **Business Logic**: Session management and workflow execution
3. **Data Layer**: In-memory session storage
4. **Event System**: SSE broadcasting for real-time updates

## Performance Considerations

- **In-memory Storage**: Sessions are stored in memory (not persistent)
- **Concurrent Execution**: Multiple sessions can run simultaneously
- **SSE Optimization**: Efficient event broadcasting to multiple clients
- **Connection Management**: Automatic cleanup of disconnected clients

## Security Considerations

- **CORS**: Currently open for development (restrict in production)
- **Input Validation**: Basic request validation
- **Error Handling**: No sensitive information in error responses
- **Rate Limiting**: Not implemented (consider for production)

## Monitoring

- **Health Endpoint**: `/health` for service monitoring
- **Structured Logging**: JSON-formatted logs with correlation IDs
- **Metrics**: Session count and status tracking
- **Error Tracking**: Comprehensive error logging

## Future Enhancements

- **Persistence**: Database storage for sessions
- **Authentication**: API key or JWT authentication
- **Rate Limiting**: Request throttling
- **Metrics**: Prometheus metrics endpoint
- **Configuration**: Environment-based configuration
- **Clustering**: Multi-instance support
- **Queue System**: Background job processing
- **Webhook Support**: External notifications




