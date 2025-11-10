# Google ADK Migration Guide

## Overview

This document describes the migration from our custom ADK implementation to Google's official Agent Development Kit (ADK).

## Architecture Changes

### Current Implementation
- **Protocol**: HTTP REST API
- **Request/Response**: Custom `TaskRequest`/`TaskResponse` structs
- **Communication**: Direct HTTP calls between orchestrator and agent services
- **Server**: Custom Gin-based HTTP server

### Google ADK Implementation
- **Protocol**: A2A (Agent-to-Agent) protocol
- **Request/Response**: A2A `Message`/`Event` types
- **Communication**: Agent-to-Agent protocol with AgentCards
- **Server**: Google ADK's A2A server (`adka2a`)

## Migration Steps

### 1. Update Dependencies
- Add `google.golang.org/adk` as a direct dependency
- Add `github.com/a2aproject/a2a-go` for A2A protocol support

### 2. Create AgentCards
Each agent service needs an AgentCard that describes:
- Agent name and description
- Capabilities (streaming, etc.)
- Preferred transport protocol
- URL endpoint

### 3. Migrate Agent Services
- Replace custom HTTP server with Google ADK's A2A server
- Use `adka2a.NewExecutor` to create A2A executors
- Use `a2asrv.NewHandler` to handle A2A requests
- Adapt TaskRequest/TaskResponse to A2A Message/Event format

### 4. Update Orchestrator
- Replace custom ADK client with Google ADK's `remoteagent`
- Use `remoteagent.New` to create remote agent clients
- Update pipeline to use A2A protocol instead of HTTP REST

### 5. Update Pipeline
- Replace HTTP-based agent calls with A2A remote agent calls
- Adapt event handling to work with A2A events
- Update error handling for A2A protocol

## Key Differences

### Request Format
**Current (HTTP REST)**:
```go
type TaskRequest struct {
    SessionID string
    Step      string
    Topic     string
    Inputs    map[string]string
}
```

**Google ADK (A2A)**:
```go
// Uses a2a.Message with Parts
message := a2a.NewMessage(a2a.MessageRoleUser, parts...)
```

### Response Format
**Current (HTTP REST)**:
```go
type TaskResponse struct {
    Delta     string
    Artifacts map[string]string
    Next      string
    Metrics   map[string]interface{}
}
```

**Google ADK (A2A)**:
```go
// Uses session.Event iterator
func Run(InvocationContext) iter.Seq2[*session.Event, error]
```

### Server Setup
**Current**:
```go
router.POST("/task", taskHandler)
```

**Google ADK**:
```go
executor := adka2a.NewExecutor(adka2a.ExecutorConfig{...})
handler := a2asrv.NewHandler(executor)
mux.Handle("/invoke", a2asrv.NewJSONRPCHandler(handler))
```

### Client Setup
**Current**:
```go
client := adkgoogle.NewClient(baseURL)
response, err := client.ExecuteTask(ctx, req)
```

**Google ADK**:
```go
remoteAgent, err := remoteagent.New(remoteagent.A2AConfig{
    Name:            "Agent Name",
    AgentCardSource: agentURL,
})
```

## Migration Checklist

- [ ] Update go.mod files with Google ADK dependencies
- [ ] Create AgentCards for each agent service
- [ ] Migrate agent-summarizer to use Google ADK A2A server
- [ ] Migrate agent-explainer to use Google ADK A2A server
- [ ] Migrate agent-critic to use Google ADK A2A server
- [ ] Migrate agent-visualizer to use Google ADK A2A server
- [ ] Update orchestrator to use Google ADK remote agent client
- [ ] Update pipeline to use A2A protocol
- [ ] Test all agent services
- [ ] Test orchestrator pipeline
- [ ] Update deployment configurations
- [ ] Update documentation

## Benefits of Migration

1. **Standard Protocol**: A2A is a standard protocol for agent-to-agent communication
2. **Better Tooling**: Google ADK provides better tooling and utilities
3. **Streaming Support**: Built-in support for streaming responses
4. **Agent Cards**: Standardized way to describe agent capabilities
5. **Future-Proof**: Aligns with Google's agent development roadmap

## Challenges

1. **Architecture Change**: Significant change from HTTP REST to A2A protocol
2. **Event Model**: Need to adapt to event-based communication
3. **Learning Curve**: Team needs to learn A2A protocol and Google ADK APIs
4. **Testing**: Need to update all tests for new protocol

