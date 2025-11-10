package google

import (
	"encoding/json"
	"fmt"
	"iter"

	"github.com/InnoFusionTech/ExplainIQ/internal/adk"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// TaskProcessorAdapter adapts a TaskProcessor to work with Google ADK
type TaskProcessorAdapter struct {
	processor adk.TaskProcessor
	logger    interface {
		WithFields(fields map[string]interface{}) interface {
			Info(args ...interface{})
			Error(args ...interface{})
		}
	}
}

// NewTaskProcessorAdapter creates a new adapter
func NewTaskProcessorAdapter(processor adk.TaskProcessor, logger interface {
	WithFields(fields map[string]interface{}) interface {
		Info(args ...interface{})
		Error(args ...interface{})
	}
}) *TaskProcessorAdapter {
	return &TaskProcessorAdapter{
		processor: processor,
		logger:    logger,
	}
}

// CreateAgent creates a Google ADK agent from a TaskProcessor
func CreateAgent(name, description string, processor adk.TaskProcessor, logger interface {
	WithFields(fields map[string]interface{}) interface {
		Info(args ...interface{})
		Error(args ...interface{})
	}
}) (agent.Agent, error) {
	adapter := NewTaskProcessorAdapter(processor, logger)
	return agent.New(agent.Config{
		Name:        name,
		Description: description,
		Run: func(ic agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return adapter.run(ic)
		},
	})
}

func (a *TaskProcessorAdapter) run(ic agent.InvocationContext) iter.Seq2[*session.Event, error] {
	return func(yield func(*session.Event, error) bool) {
		// Get the user content from the invocation context
		userContent := ic.UserContent()
		if userContent == nil {
			// Try to get from session events
			events := ic.Session().Events()
			if events.Len() == 0 {
				event := NewErrorEvent(ic, fmt.Errorf("no user message found"))
				yield(event, nil)
				return
			}
			
			// Get the last event
			lastEvent := events.At(events.Len() - 1)
			if lastEvent.LLMResponse.Content == nil {
				event := NewErrorEvent(ic, fmt.Errorf("last event has no content"))
				yield(event, nil)
				return
			}
			userContent = lastEvent.LLMResponse.Content
		}

		// Convert user content to TaskRequest
		req, err := contentToTaskRequest(userContent, ic)
		if err != nil {
			event := NewErrorEvent(ic, fmt.Errorf("failed to parse task request: %w", err))
			yield(event, nil)
			return
		}

		// Process the task
		// InvocationContext embeds context.Context, so we can use ic directly
		response, err := a.processor.ProcessTask(ic, req)
		if err != nil {
			if a.logger != nil {
				a.logger.WithFields(map[string]interface{}{
					"session_id": req.SessionID,
					"error":      err,
				}).Error("Task processing failed")
			}
			event := NewErrorEvent(ic, fmt.Errorf("task processing failed: %w", err))
			yield(event, nil)
			return
		}

		// Convert TaskResponse to session event
		event := responseToEvent(ic, response)
		yield(event, nil)
	}
}

// contentToTaskRequest converts genai.Content to a TaskRequest
func contentToTaskRequest(content *genai.Content, ic agent.InvocationContext) (adk.TaskRequest, error) {
	// Extract session ID from context
	sessionID := ic.Session().ID()

	// Parse the message content
	var inputs map[string]string
	var step string
	var topic string

	// Extract text from content parts
	for _, part := range content.Parts {
		if part.Text != "" {
			// Try to parse as JSON TaskRequest
			var taskReq struct {
				SessionID string            `json:"session_id"`
				Step      string            `json:"step"`
				Topic     string            `json:"topic"`
				Inputs    map[string]string `json:"inputs"`
			}
			if err := json.Unmarshal([]byte(part.Text), &taskReq); err == nil {
				// Successfully parsed as JSON TaskRequest
				if taskReq.SessionID != "" {
					sessionID = taskReq.SessionID
				}
				return adk.TaskRequest{
					SessionID: sessionID,
					Step:      taskReq.Step,
					Topic:     taskReq.Topic,
					Inputs:    taskReq.Inputs,
				}, nil
			}
			// If not JSON, treat as topic
			topic = part.Text
			inputs = map[string]string{
				"topic": part.Text,
			}
		}
	}

	if inputs == nil {
		inputs = make(map[string]string)
	}

	// Try to extract step from session state
	if step == "" {
		step = "default"
		state := ic.Session().State()
		if stepVal, err := state.Get("step"); err == nil {
			if stepStr, ok := stepVal.(string); ok {
				step = stepStr
			}
		}
	}

	return adk.TaskRequest{
		SessionID: sessionID,
		Step:      step,
		Topic:     topic,
		Inputs:    inputs,
	}, nil
}

// responseToEvent converts a TaskResponse to a session event
func responseToEvent(ic agent.InvocationContext, response adk.TaskResponse) *session.Event {
	event := session.NewEvent(ic.InvocationID())
	
	// Convert artifacts to JSON
	parts := make([]*genai.Part, 0)
	
	if len(response.Artifacts) > 0 {
		artifactsJSON, err := json.Marshal(response.Artifacts)
		if err == nil {
			parts = append(parts, &genai.Part{
				Text: string(artifactsJSON),
			})
		}
	}

	// Add delta if present
	if response.Delta != "" {
		parts = append(parts, &genai.Part{
			Text: response.Delta,
		})
	}

	// Add metrics if present
	if len(response.Metrics) > 0 {
		metricsJSON, err := json.Marshal(response.Metrics)
		if err == nil {
			parts = append(parts, &genai.Part{
				Text: string(metricsJSON),
			})
		}
	}

	// Set the LLM response
	event.LLMResponse = model.LLMResponse{
		Content: &genai.Content{
			Role:  genai.RoleModel,
			Parts: parts,
		},
	}

	return event
}

// NewErrorEvent creates a new error event
func NewErrorEvent(ic agent.InvocationContext, err error) *session.Event {
	event := session.NewEvent(ic.InvocationID())
	event.LLMResponse = model.LLMResponse{
		ErrorCode:    "TASK_ERROR",
		ErrorMessage: err.Error(),
		Content: &genai.Content{
			Role: genai.RoleModel,
			Parts: []*genai.Part{
				{
					Text: fmt.Sprintf(`{"error": "%s"}`, err.Error()),
				},
			},
		},
	}
	return event
}

// TaskRequestToMessage converts a TaskRequest to genai.Part slice
func TaskRequestToMessage(req adk.TaskRequest) ([]*genai.Part, error) {
	// Marshal TaskRequest to JSON
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task request: %w", err)
	}

	return []*genai.Part{
		{
			Text: string(reqJSON),
		},
	}, nil
}

// MessageToTaskRequest converts genai.Part slice to a TaskRequest
func MessageToTaskRequest(parts []*genai.Part, sessionID string) (adk.TaskRequest, error) {
	if len(parts) == 0 {
		return adk.TaskRequest{}, fmt.Errorf("no message parts found")
	}

	// Try to parse as JSON TaskRequest
	var req adk.TaskRequest
	if err := json.Unmarshal([]byte(parts[0].Text), &req); err == nil {
		if req.SessionID == "" {
			req.SessionID = sessionID
		}
		return req, nil
	}

	// If not JSON, treat as topic
	return adk.TaskRequest{
		SessionID: sessionID,
		Step:      "default",
		Topic:     parts[0].Text,
		Inputs: map[string]string{
			"topic": parts[0].Text,
		},
	}, nil
}
