package constants

import "time"

// Service names
const (
	ServiceOrchestrator  = "orchestrator"
	ServiceSummarizer    = "agent-summarizer"
	ServiceExplainer     = "agent-explainer"
	ServiceCritic        = "agent-critic"
	ServiceVisualizer    = "agent-visualizer"
	ServiceFrontend      = "frontend"
)

// Session statuses
const (
	SessionStatusCreated   = "created"
	SessionStatusRunning   = "running"
	SessionStatusCompleted = "completed"
	SessionStatusFailed    = "failed"
	SessionStatusCancelled = "cancelled"
)

// Step statuses
const (
	StepStatusPending   = "pending"
	StepStatusRunning   = "running"
	StepStatusCompleted = "completed"
	StepStatusFailed    = "failed"
	StepStatusCancelled = "cancelled"
)

// SSE Event types
const (
	EventTypeConnected      = "connected"
	EventTypeStepStart      = "step-start"
	EventTypeStepComplete  = "step-complete"
	EventTypeStepRetry     = "step-retry"
	EventTypeStepDelta     = "step-delta"
	EventTypePipelineCompleted = "pipeline-completed"
	EventTypePipelineFailed    = "pipeline-failed"
	EventTypeFinal           = "final"
)

// HTTP endpoints
const (
	EndpointHealth     = "/health"
	EndpointHealthz    = "/healthz"
	EndpointTask       = "/task"
	EndpointSessions   = "/api/sessions"
	EndpointSessionRun = "/api/sessions/{id}/run"
	EndpointSessionResult = "/api/sessions/{id}/result"
)

// Default ports
const (
	DefaultPortOrchestrator = "8080"
	DefaultPortSummarizer   = "8081"
	DefaultPortExplainer    = "8082"
	DefaultPortCritic       = "8083"
	DefaultPortVisualizer   = "8084"
	DefaultPortFrontend     = "8085"
)

// Default service URLs
const (
	DefaultURLOrchestrator = "http://localhost:8080"
	DefaultURLSummarizer   = "http://localhost:8081"
	DefaultURLExplainer    = "http://localhost:8082"
	DefaultURLCritic       = "http://localhost:8083"
	DefaultURLVisualizer   = "http://localhost:8084"
	DefaultURLFrontend     = "http://localhost:8085"
)

// Docker service URLs
const (
	DockerURLOrchestrator = "http://orchestrator:8080"
	DockerURLSummarizer   = "http://agent-summarizer:8081"
	DockerURLExplainer    = "http://agent-explainer:8082"
	DockerURLCritic       = "http://agent-critic:8083"
	DockerURLVisualizer   = "http://agent-visualizer:8084"
	DockerURLFrontend     = "http://frontend:8085"
)

// Pipeline configuration defaults
const (
	DefaultMaxRetries    = 3
	DefaultRetryDelay    = 2 * time.Second
	DefaultStepTimeout   = 5 * time.Minute
	DefaultContextTopK   = 5
	DefaultElasticIndex  = "lessons"
)

// Time durations (converted to constants)
const (
	DefaultShutdownTimeout = 30 * time.Second
	DefaultReadTimeout      = 15 * time.Second
	DefaultWriteTimeout     = 5 * time.Minute  // Increased for long-running tasks like critic
	DefaultIdleTimeout      = 60 * time.Second
)

