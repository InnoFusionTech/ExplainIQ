# Orchestrator API Usage Examples

## Starting the Server

```bash
go run ./cmd/orchestrator
```

The server will start on `http://localhost:8080`

## API Endpoints

### 1. Create a Session

**POST** `/api/sessions`

```bash
curl -X POST http://localhost:8080/api/sessions \
  -H "Content-Type: application/json" \
  -d '{"topic": "machine learning"}'
```

Response:
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000"
}
```

### 2. Run a Session (SSE Stream)

**POST** `/api/sessions/{id}/run`

```bash
curl -X POST http://localhost:8080/api/sessions/123e4567-e89b-12d3-a456-426614174000/run
```

This will stream Server-Sent Events:

```
data: {"type":"connected","session_id":"123e4567-e89b-12d3-a456-426614174000"}

data: {"type":"step-start","session_id":"123e4567-e89b-12d3-a456-426614174000","step_id":"step-1","data":{"step_name":"analyze-topic","step_index":0},"timestamp":"2023-10-14T23:58:28Z"}

data: {"type":"step-delta","session_id":"123e4567-e89b-12d3-a456-426614174000","step_id":"step-1","data":{"delta":"Processing analyze-topic... 20% complete","progress":20},"timestamp":"2023-10-14T23:58:28Z"}

data: {"type":"step-delta","session_id":"123e4567-e89b-12d3-a456-426614174000","step_id":"step-1","data":{"delta":"Processing analyze-topic... 40% complete","progress":40},"timestamp":"2023-10-14T23:58:29Z"}

data: {"type":"step-complete","session_id":"123e4567-e89b-12d3-a456-426614174000","step_id":"step-1","data":{"step_name":"analyze-topic","output":"Completed analyze-topic for topic: machine learning","duration":2500},"timestamp":"2023-10-14T23:58:30Z"}

data: {"type":"final","session_id":"123e4567-e89b-12d3-a456-426614174000","data":{"result":{"lesson":"Complete lesson on machine learning","images":{"image1":"path/to/image1.jpg","image2":"path/to/image2.jpg"},"summary":"Summary of machine learning lesson","duration":12500000000,"completed_at":"2023-10-14T23:58:35Z"}},"timestamp":"2023-10-14T23:58:35Z"}
```

### 3. Get Session Result

**GET** `/api/sessions/{id}/result`

```bash
curl http://localhost:8080/api/sessions/123e4567-e89b-12d3-a456-426614174000/result
```

Response:
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

### 4. Health Check

**GET** `/health`

```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "healthy",
  "timestamp": "2023-10-14T23:58:35Z",
  "sessions": 1
}
```

## JavaScript Client Example

```javascript
// Create a session
async function createSession(topic) {
  const response = await fetch('/api/sessions', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ topic }),
  });
  
  const data = await response.json();
  return data.id;
}

// Run a session with SSE
function runSession(sessionId) {
  const eventSource = new EventSource(`/api/sessions/${sessionId}/run`);
  
  eventSource.onmessage = function(event) {
    const data = JSON.parse(event.data);
    
    switch (data.type) {
      case 'connected':
        console.log('Connected to session:', data.session_id);
        break;
        
      case 'step-start':
        console.log('Step started:', data.data.step_name);
        break;
        
      case 'step-delta':
        console.log('Progress:', data.data.delta);
        break;
        
      case 'step-complete':
        console.log('Step completed:', data.data.step_name);
        break;
        
      case 'final':
        console.log('Session completed:', data.data.result);
        eventSource.close();
        break;
    }
  };
  
  eventSource.onerror = function(event) {
    console.error('SSE error:', event);
    eventSource.close();
  };
}

// Get session result
async function getSessionResult(sessionId) {
  const response = await fetch(`/api/sessions/${sessionId}/result`);
  const result = await response.json();
  return result;
}

// Usage
async function main() {
  const sessionId = await createSession('machine learning');
  runSession(sessionId);
}
```

## Python Client Example

```python
import requests
import json
import sseclient

def create_session(topic):
    response = requests.post('http://localhost:8080/api/sessions', 
                           json={'topic': topic})
    return response.json()['id']

def run_session(session_id):
    response = requests.post(f'http://localhost:8080/api/sessions/{session_id}/run',
                           stream=True)
    
    client = sseclient.SSEClient(response)
    
    for event in client.events():
        data = json.loads(event.data)
        
        if data['type'] == 'connected':
            print(f"Connected to session: {data['session_id']}")
        elif data['type'] == 'step-start':
            print(f"Step started: {data['data']['step_name']}")
        elif data['type'] == 'step-delta':
            print(f"Progress: {data['data']['delta']}")
        elif data['type'] == 'step-complete':
            print(f"Step completed: {data['data']['step_name']}")
        elif data['type'] == 'final':
            print(f"Session completed: {data['data']['result']}")
            break

def get_session_result(session_id):
    response = requests.get(f'http://localhost:8080/api/sessions/{session_id}/result')
    return response.json()

# Usage
if __name__ == '__main__':
    session_id = create_session('machine learning')
    run_session(session_id)
    result = get_session_result(session_id)
    print(f"Final result: {result}")
```

## Workflow Steps

The orchestrator executes the following steps for each session:

1. **analyze-topic**: Analyze the input topic
2. **generate-outline**: Create a lesson outline
3. **create-content**: Generate the lesson content
4. **generate-images**: Create relevant images
5. **finalize-lesson**: Finalize and package the lesson

Each step includes progress updates via SSE events.

## Error Handling

- **404**: Session not found
- **400**: Invalid request or session not completed
- **409**: Session already running

## CORS Support

The server includes CORS headers for development:
- `Access-Control-Allow-Origin: *`
- `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`
- `Access-Control-Allow-Headers: Accept, Authorization, Content-Type, X-CSRF-Token`




