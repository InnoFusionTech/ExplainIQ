# ExplainIQ Architecture Diagram

## Service Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          ExplainIQ Platform                                  │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                              EXTERNAL CLIENTS                                │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    │ HTTP Requests
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            FRONTEND LAYER                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                               │
│  ┌──────────────────────┐         ┌──────────────────────┐                  │
│  │  Go Frontend         │         │  Next.js Frontend    │                  │
│  │  Port: 8085         │         │  Port: 3000         │                  │
│  │                      │         │                      │                  │
│  │  - Static Files     │         │  - React UI         │                  │
│  │  - HTML Templates   │         │  - API Routes       │                  │
│  │  - Basic Routes     │         │  - PDF Generation   │                  │
│  └──────────────────────┘         └──────────────────────┘                  │
│           │                               │                                   │
│           └───────────────┬───────────────┘                                   │
│                           │                                                   │
│                           │ HTTP API Calls                                     │
│                           ▼                                                   │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    │
┌─────────────────────────────────────────────────────────────────────────────┐
│                           ORCHESTRATION LAYER                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                               │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                    Orchestrator Service                               │ │
│  │                    Port: 8080                                         │ │
│  │                                                                         │ │
│  │  ┌─────────────────────────────────────────────────────────────────┐ │ │
│  │  │  Features:                                                       │ │ │
│  │  │  • Session Management                                            │ │ │
│  │  │  • Pipeline Orchestration                                        │ │ │
│  │  │  • SSE (Server-Sent Events)                                      │ │ │
│  │  │  • Authentication (JWT)                                          │ │ │
│  │  │  • Quota Management                                              │ │ │
│  │  │  • Rate Limiting                                                 │ │ │
│  │  │  • Cost Tracking                                                 │ │ │
│  │  └─────────────────────────────────────────────────────────────────┘ │ │
│  │                                                                         │ │
│  │  ┌─────────────────────────────────────────────────────────────────┐ │ │
│  │  │  Pipeline Workflow:                                              │ │ │
│  │  │                                                                   │ │ │
│  │  │  1. Summarizer → Creates outline, identifies misconceptions     │ │ │
│  │  │  2. Explainer → Generates lesson content                        │ │ │
│  │  │  3. Critic → Reviews and suggests improvements                  │ │ │
│  │  │  4. Visualizer → Creates visualizations                         │ │ │
│  │  └─────────────────────────────────────────────────────────────────┘ │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                           │                                                   │
│                           │ Coordinates                                       │
│                           ▼                                                   │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
            ┌───────────────────────┼───────────────────────┐
            │                       │                       │
            │                       │                       │
┌───────────▼──────────┐   ┌─────────▼─────────┐   ┌──────────▼──────────┐
│   AGENT LAYER        │   │   AGENT LAYER      │   │   AGENT LAYER      │
├──────────────────────┤   ├───────────────────┤   ├────────────────────┤
│                      │   │                   │   │                    │
│  Summarizer          │   │  Explainer        │   │  Critic            │
│  Port: 8081          │   │  Port: 8082       │   │  Port: 8083        │
│                      │   │                   │   │                    │
│  • Topic Analysis    │   │  • Lesson Gen     │   │  • Quality Review  │
│  • Outline Creation  │   │  • Explanations   │   │  • Issue Detection │
│  • Misconceptions    │   │  • Step-by-step   │   │  • Patch Plans     │
│                      │   │                   │   │                    │
└──────────────────────┘   └───────────────────┘   └────────────────────┘
            │                       │                       │
            │                       │                       │
            └───────────────────────┼───────────────────────┘
                                    │
                                    │
                        ┌───────────▼───────────┐
                        │   AGENT LAYER         │
                        ├───────────────────────┤
                        │                       │
                        │  Visualizer           │
                        │  Port: 8084           │
                        │                       │
                        │  • Diagrams           │
                        │  • Visual Aids        │
                        │  • Infographics       │
                        │                       │
                        └───────────────────────┘
                                    │
┌─────────────────────────────────────────────────────────────────────────────┐
│                           EXTERNAL SERVICES                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                               │
│  ┌──────────────────────┐         ┌──────────────────────┐                  │
│  │  Google Gemini AI    │         │  Elasticsearch       │                  │
│  │  (Optional)          │         │  (Optional)          │                  │
│  │                      │         │                      │                  │
│  │  • LLM API           │         │  • Context Retrieval │                  │
│  │  • Text Generation   │         │  • Document Search   │                  │
│  │  • Embeddings        │         │  • Knowledge Base    │                  │
│  └──────────────────────┘         └──────────────────────┘                  │
│                                                                               │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Service Communication Flow

```
┌──────────────┐
│   Frontend   │
│  (Port 8085) │
└──────┬───────┘
       │
       │ 1. POST /api/sessions
       │    { "topic": "..." }
       │
       ▼
┌─────────────────┐
│  Orchestrator   │
│  (Port 8080)    │
└──────┬──────────┘
       │
       │ 2. Creates Session
       │
       │ 3. Pipeline Execution:
       │
       ├───► Summarizer (8081)
       │    • Input: topic
       │    • Output: outline, misconceptions
       │
       ├───► Explainer (8082)
       │    • Input: outline, misconceptions
       │    • Output: lesson content
       │
       ├───► Critic (8083)
       │    • Input: lesson content
       │    • Output: critique, patch plan
       │
       └───► Visualizer (8084)
            • Input: lesson content
            • Output: visualizations
       │
       │ 4. Aggregates Results
       │
       ▼
┌──────────────┐
│   Frontend   │
│  (Port 8085) │
└──────────────┘
       │
       │ 5. SSE Stream / Result
       │
       ▼
```

## Service Dependencies

```
Orchestrator
├── Depends on:
│   ├── agent-summarizer (8081)
│   ├── agent-explainer (8082)
│   ├── agent-critic (8083)
│   └── agent-visualizer (8084)
│
├── Optional Dependencies:
│   ├── Elasticsearch (for context retrieval)
│   └── Google Cloud Metadata (for authentication)
│
└── Internal Dependencies:
    ├── auth (JWT authentication)
    ├── quota (usage limits)
    ├── rate_limiter (rate limiting)
    ├── cost_tracker (cost tracking)
    └── storage (session storage)

Agent Services (Summarizer, Explainer, Critic, Visualizer)
├── Depends on:
│   └── Google Gemini API (via internal/llm)
│
└── Internal Dependencies:
    ├── adk (Agent Development Kit)
    ├── auth (service authentication)
    └── llm (LLM client)

Frontend
└── Depends on:
    └── Orchestrator (8080)
```

## Network Topology

```
┌──────────────────────────────────────────────────────────┐
│              Docker Network: explainiq-network            │
│                   Driver: bridge                          │
└──────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
        ▼                   ▼                   ▼
┌───────────────┐   ┌───────────────┐   ┌───────────────┐
│ Orchestrator  │   │   Agents      │   │   Frontend   │
│   :8080       │   │  :8081-8084   │   │   :8085      │
└───────────────┘   └───────────────┘   └───────────────┘
```

## Port Allocation

| Service | Port | Health Endpoint | Description |
|---------|------|-----------------|-------------|
| Orchestrator | 8080 | `/health` | Main API coordinator |
| Agent Summarizer | 8081 | `/health` | AI summarization service |
| Agent Explainer | 8082 | `/health` | AI explanation service |
| Agent Critic | 8083 | `/healthz` | AI critique service |
| Agent Visualizer | 8084 | `/health` | AI visualization service |
| Frontend (Go) | 8085 | `/healthz` | Go-based web frontend |
| Frontend (Next.js) | 3000 | `/` | React/Next.js frontend |

## Data Flow

### Request Flow
1. **Client** → Frontend (HTTP request with topic)
2. **Frontend** → Orchestrator (POST /api/sessions)
3. **Orchestrator** → Creates session, starts pipeline
4. **Orchestrator** → Agent Summarizer (POST /task)
5. **Agent Summarizer** → Returns outline & misconceptions
6. **Orchestrator** → Agent Explainer (POST /task)
7. **Agent Explainer** → Returns lesson content
8. **Orchestrator** → Agent Critic (POST /task)
9. **Agent Critic** → Returns critique & patch plan
10. **Orchestrator** → Agent Visualizer (POST /task)
11. **Agent Visualizer** → Returns visualizations
12. **Orchestrator** → Aggregates results
13. **Orchestrator** → Frontend (SSE stream / Result)
14. **Frontend** → Client (Response / SSE)

### Authentication Flow
1. **Orchestrator** → Google Metadata Server (Get ID token)
2. **Orchestrator** → Agent Service (Request with Bearer token)
3. **Agent Service** → Google Public Keys (Verify JWT)
4. **Agent Service** → Validates token & processes request

## Component Details

### Orchestrator
- **Technology**: Go, Gin framework, Chi router
- **Key Features**:
  - Session lifecycle management
  - Pipeline orchestration with retry logic
  - Server-Sent Events (SSE) for real-time updates
  - Authentication & authorization
  - Quota management
  - Rate limiting
  - Cost tracking

### Agent Services
- **Technology**: Go, Gin framework
- **Common Features**:
  - ADK (Agent Development Kit) integration
  - JWT authentication
  - Health check endpoints
  - Task processing via `/task` endpoint

### Frontend Services
- **Go Frontend**: Simple static file server with templates
- **Next.js Frontend**: React-based UI with PDF generation

### Internal Packages
- **adk**: Agent Development Kit for service communication
- **auth**: Google JWT authentication
- **llm**: Gemini AI client wrapper
- **storage**: Storage abstraction (Firestore)
- **quota**: Usage quota management
- **rate_limiter**: Rate limiting middleware
- **cost_tracker**: Cost tracking utilities
- **elastic**: Elasticsearch client (optional)












