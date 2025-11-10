package adk

import (
	"context"
	"fmt"
	"time"
)

// TaskProcessor defines the interface for processing tasks
// This interface is used by agents to process task requests
type TaskProcessor interface {
	ProcessTask(ctx context.Context, req TaskRequest) (TaskResponse, error)
}

// TaskRequest represents a request to execute a task
type TaskRequest struct {
	SessionID string            `json:"session_id"` // Unique session identifier
	Step      string            `json:"step"`       // Current step in the workflow
	Topic     string            `json:"topic"`      // Topic or category of the task
	Inputs    map[string]string `json:"inputs"`     // Input parameters for the task
}

// TaskResponse represents the response from a task execution
type TaskResponse struct {
	Delta     string                 `json:"delta"`     // Incremental output or changes
	Artifacts map[string]string      `json:"artifacts"` // Generated artifacts (files, data, etc.)
	Next      string                 `json:"next"`      // Next step or action to take
	Metrics   map[string]interface{} `json:"metrics"`   // Performance and execution metrics
}

// TaskStatus represents the status of a task execution
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
	TaskStatusRetrying  TaskStatus = "retrying"
)

// TaskError represents an error that occurred during task execution
type TaskError struct {
	Code    string `json:"code"`    // Error code
	Message string `json:"message"` // Error message
	Details string `json:"details"` // Additional error details
}

// TaskMetadata represents metadata about a task execution
type TaskMetadata struct {
	TaskID         string                 `json:"task_id"`         // Unique task identifier
	SessionID      string                 `json:"session_id"`      // Session identifier
	Step           string                 `json:"step"`            // Current step
	Topic          string                 `json:"topic"`           // Task topic
	Status         TaskStatus             `json:"status"`          // Current status
	CreatedAt      time.Time              `json:"created_at"`      // Creation timestamp
	StartedAt      *time.Time             `json:"started_at"`      // Start timestamp
	CompletedAt    *time.Time             `json:"completed_at"`    // Completion timestamp
	Duration       *time.Duration         `json:"duration"`        // Execution duration
	RetryCount     int                    `json:"retry_count"`     // Number of retries
	MaxRetries     int                    `json:"max_retries"`     // Maximum retries allowed
	Error          *TaskError             `json:"error"`           // Error information
	Inputs         map[string]string      `json:"inputs"`          // Input parameters
	Outputs        map[string]string      `json:"outputs"`         // Output parameters
	Artifacts      map[string]string      `json:"artifacts"`       // Generated artifacts
	Metrics        map[string]interface{} `json:"metrics"`         // Execution metrics
	CorrelationID  string                 `json:"correlation_id"`  // Request correlation ID
	IdempotencyKey string                 `json:"idempotency_key"` // Idempotency key
}

// TaskConfig represents configuration for task execution
type TaskConfig struct {
	Timeout     time.Duration `json:"timeout"`      // Task timeout
	MaxRetries  int           `json:"max_retries"`  // Maximum retries
	RetryDelay  time.Duration `json:"retry_delay"`  // Initial retry delay
	BackoffType string        `json:"backoff_type"` // Backoff strategy (exponential, linear, fixed)
}

// DefaultTaskConfig returns the default task configuration
func DefaultTaskConfig() TaskConfig {
	return TaskConfig{
		Timeout:     30 * time.Second,
		MaxRetries:  3,
		RetryDelay:  1 * time.Second,
		BackoffType: "exponential",
	}
}

// Validate validates the task request
func (tr *TaskRequest) Validate() error {
	if tr.SessionID == "" {
		return &TaskError{
			Code:    "INVALID_REQUEST",
			Message: "SessionID is required",
		}
	}

	if tr.Step == "" {
		return &TaskError{
			Code:    "INVALID_REQUEST",
			Message: "Step is required",
		}
	}

	if tr.Topic == "" {
		return &TaskError{
			Code:    "INVALID_REQUEST",
			Message: "Topic is required",
		}
	}

	return nil
}

// IsEmpty checks if the task response is empty
func (tr *TaskResponse) IsEmpty() bool {
	return tr.Delta == "" &&
		len(tr.Artifacts) == 0 &&
		tr.Next == "" &&
		len(tr.Metrics) == 0
}

// AddArtifact adds an artifact to the response
func (tr *TaskResponse) AddArtifact(key, value string) {
	if tr.Artifacts == nil {
		tr.Artifacts = make(map[string]string)
	}
	tr.Artifacts[key] = value
}

// AddMetric adds a metric to the response
func (tr *TaskResponse) AddMetric(key string, value interface{}) {
	if tr.Metrics == nil {
		tr.Metrics = make(map[string]interface{})
	}
	tr.Metrics[key] = value
}

// Error implements the error interface for TaskError
func (te *TaskError) Error() string {
	if te.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", te.Code, te.Message, te.Details)
	}
	return fmt.Sprintf("%s: %s", te.Code, te.Message)
}

// IsRetryable checks if the error is retryable
func (te *TaskError) IsRetryable() bool {
	retryableCodes := []string{
		"TIMEOUT",
		"NETWORK_ERROR",
		"RATE_LIMITED",
		"TEMPORARY_FAILURE",
		"SERVICE_UNAVAILABLE",
		"INTERNAL_ERROR",
		"BAD_GATEWAY",
		"GATEWAY_TIMEOUT",
	}

	for _, code := range retryableCodes {
		if te.Code == code {
			return true
		}
	}

	return false
}

// UpdateStatus updates the task metadata status
func (tm *TaskMetadata) UpdateStatus(status TaskStatus) {
	tm.Status = status
	now := time.Now()

	switch status {
	case TaskStatusRunning:
		if tm.StartedAt == nil {
			tm.StartedAt = &now
		}
	case TaskStatusCompleted, TaskStatusFailed, TaskStatusCancelled:
		if tm.CompletedAt == nil {
			tm.CompletedAt = &now
		}
		if tm.StartedAt != nil {
			duration := now.Sub(*tm.StartedAt)
			tm.Duration = &duration
		}
	}
}

// IncrementRetry increments the retry count
func (tm *TaskMetadata) IncrementRetry() {
	tm.RetryCount++
	tm.Status = TaskStatusRetrying
}

// CanRetry checks if the task can be retried
func (tm *TaskMetadata) CanRetry() bool {
	return tm.RetryCount < tm.MaxRetries &&
		tm.Status != TaskStatusCompleted &&
		tm.Status != TaskStatusCancelled
}

// SetError sets the error information
func (tm *TaskMetadata) SetError(err error) {
	if taskErr, ok := err.(*TaskError); ok {
		tm.Error = taskErr
	} else {
		tm.Error = &TaskError{
			Code:    "UNKNOWN_ERROR",
			Message: err.Error(),
		}
	}
	tm.Status = TaskStatusFailed
}

// AddMetric adds a metric to the task metadata
func (tm *TaskMetadata) AddMetric(key string, value interface{}) {
	if tm.Metrics == nil {
		tm.Metrics = make(map[string]interface{})
	}
	tm.Metrics[key] = value
}
