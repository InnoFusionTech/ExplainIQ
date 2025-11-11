package google

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/InnoFusionTech/ExplainIQ/internal/adk"
	authclient "github.com/InnoFusionTech/ExplainIQ/internal/auth"
	"github.com/sirupsen/logrus"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/remoteagent"
	"google.golang.org/adk/session"
)

// Client represents a Google ADK-compatible client for communicating with agents
// This follows Google ADK patterns for agent-to-agent communication
// It can use either HTTP REST (legacy) or A2A protocol (Google ADK)
type Client struct {
	baseURL     string
	httpClient  *http.Client
	logger      *logrus.Logger
	authClient  *authclient.Client
	useA2A      bool
	remoteAgent agent.Agent
}

// NewClient creates a new Google ADK-compatible client
// It will use A2A protocol if the baseURL points to an A2A server (has /.well-known/agent-card)
// Otherwise, it will use HTTP REST for backward compatibility
func NewClient(baseURL string) *Client {
	// Check if this is an A2A server by checking if it's HTTPS or has a specific pattern
	useA2A := strings.HasPrefix(baseURL, "https://") || strings.Contains(baseURL, ".a.run.app")

	client := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logrus.New(),
		useA2A: useA2A,
	}

	// If using A2A, create remote agent
	if useA2A {
		agentCardURL := baseURL
		if !strings.HasSuffix(agentCardURL, "/") {
			agentCardURL += "/"
		}
		agentCardURL += ".well-known/agent-card"

		remoteAgent, err := remoteagent.New(remoteagent.A2AConfig{
			Name:            "Remote Agent Client",
			AgentCardSource: agentCardURL,
		})
		if err == nil {
			client.remoteAgent = remoteAgent
		} else {
			// Fall back to HTTP if A2A fails
			client.useA2A = false
			client.logger.WithError(err).Warn("Failed to create remote agent, falling back to HTTP")
		}
	}

	return client
}

// WithAuthClient sets an authentication client for service-to-service authentication
func (c *Client) WithAuthClient(authClient *authclient.Client) *Client {
	c.authClient = authClient
	return c
}

// WithTimeout sets the HTTP client timeout
func (c *Client) WithTimeout(timeout time.Duration) *Client {
	c.httpClient.Timeout = timeout
	return c
}

// WithLogger sets a custom logger
func (c *Client) WithLogger(logger *logrus.Logger) *Client {
	c.logger = logger
	return c
}

// ExecuteTask executes a task on a remote agent following Google ADK patterns
// It uses A2A protocol if available, otherwise falls back to HTTP REST
func (c *Client) ExecuteTask(ctx context.Context, req *adk.TaskRequest) (*adk.TaskResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Use A2A protocol if detected (A2A servers use JSON-RPC on /invoke endpoint)
	// Agent services only have /invoke endpoints, not /task, so we must use JSON-RPC
	// Otherwise fall back to HTTP REST (for non-A2A servers)
	if c.useA2A {
		// A2A servers use JSON-RPC format on /invoke endpoint
		// Use JSON-RPC even if remoteAgent is nil (it may fail to create but endpoint still exists)
		return c.executeTaskJSONRPC(ctx, req)
	}

	// Fall back to HTTP REST (for non-A2A servers with /task endpoint)
	return c.executeTaskHTTP(ctx, req)
}

// executeTaskJSONRPC executes a task using JSON-RPC protocol (A2A)
func (c *Client) executeTaskJSONRPC(ctx context.Context, req *adk.TaskRequest) (*adk.TaskResponse, error) {
	// Marshal request
	requestBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create JSON-RPC request
	// The A2A JSON-RPC handler expects method "message/send" (per A2A spec §7)
	// Params should be MessageSendParams with a Message object
	// Message should have role and parts fields
	jsonRPCRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "message/send", // A2A JSON-RPC method name per spec
		"params": map[string]interface{}{
			"message": map[string]interface{}{
				"role": "user",
				"parts": []map[string]interface{}{
					{
						"kind": "text",
						"text": string(requestBody),
					},
				},
			},
		},
		"id": 1,
	}

	jsonRPCBody, err := json.Marshal(jsonRPCRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON-RPC request: %w", err)
	}

	// Create HTTP request to /invoke endpoint
	url := c.baseURL
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	url += "invoke"

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonRPCBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "ExplainIQ-Google-ADK-Client/1.0")
	httpReq.Header.Set("X-ADK-Version", "1.0")

	// Add authentication if available (for Cloud Run service-to-service auth)
	if c.authClient != nil && strings.HasPrefix(c.baseURL, "https://") {
		token, err := c.authClient.GetIDToken(ctx, c.baseURL)
		if err != nil {
			c.logger.WithError(err).Warn("Failed to get ID token, proceeding without authentication")
		} else {
			httpReq.Header.Set("Authorization", "Bearer "+token)
		}
	}

	// Execute request
	c.logger.WithFields(logrus.Fields{
		"url":        url,
		"session_id": req.SessionID,
		"step":       req.Step,
	}).Debug("Executing task via JSON-RPC (A2A)")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		var errorResp struct {
			Error   string `json:"error"`
			Details string `json:"details"`
		}
		if err := json.Unmarshal(responseBody, &errorResp); err == nil {
			return nil, fmt.Errorf("agent error: %s - %s", errorResp.Error, errorResp.Details)
		}
		return nil, fmt.Errorf("agent error: HTTP %d - %s", resp.StatusCode, string(responseBody))
	}

	// Parse JSON-RPC response
	var jsonRPCResponse struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      int             `json:"id"`
		Result  json.RawMessage `json:"result,omitempty"`
		Error   *struct {
			Code    int         `json:"code"`
			Message string      `json:"message"`
			Data    interface{} `json:"data,omitempty"` // Can be string, object, or null
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(responseBody, &jsonRPCResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON-RPC response: %w", err)
	}

	if jsonRPCResponse.Error != nil {
		errorMsg := jsonRPCResponse.Error.Message
		if jsonRPCResponse.Error.Data != nil {
			// Try to format the error data
			if dataBytes, err := json.Marshal(jsonRPCResponse.Error.Data); err == nil {
				errorMsg += " - " + string(dataBytes)
			} else {
				errorMsg += fmt.Sprintf(" - %v", jsonRPCResponse.Error.Data)
			}
		}
		return nil, fmt.Errorf("agent error: %s (code: %d)", errorMsg, jsonRPCResponse.Error.Code)
	}

	// Parse the result - A2A protocol returns SendMessageResult which can be Task or Message
	// The result is a single event (Task or Message), not an array
	// Log the raw result for debugging
	resultPreview := string(jsonRPCResponse.Result)
	if len(resultPreview) > 1000 {
		resultPreview = resultPreview[:1000] + "..."
	}
	c.logger.WithFields(logrus.Fields{
		"session_id":     req.SessionID,
		"result_length":  len(jsonRPCResponse.Result),
		"result_preview": resultPreview,
		"full_result":    string(jsonRPCResponse.Result), // Log full result to see structure
	}).Info("Parsing A2A JSON-RPC response")

	// Try to parse as a single event (Task or Message)
	var singleEvent map[string]interface{}
	if err := json.Unmarshal(jsonRPCResponse.Result, &singleEvent); err != nil {
		// If not a single event, try as array
		var events []map[string]interface{}
		if err2 := json.Unmarshal(jsonRPCResponse.Result, &events); err2 == nil {
			if len(events) > 0 {
				singleEvent = events[len(events)-1] // Use last event
			} else {
				return nil, fmt.Errorf("empty events array in A2A response")
			}
		} else {
			// Try to parse as direct TaskResponse (fallback)
			var taskResponse adk.TaskResponse
			if err3 := json.Unmarshal(jsonRPCResponse.Result, &taskResponse); err3 == nil {
				return &taskResponse, nil
			}
			// Log the actual response for debugging
			c.logger.WithFields(logrus.Fields{
				"session_id": req.SessionID,
				"raw_result": string(jsonRPCResponse.Result),
			}).Error("Failed to parse A2A response")
			return nil, fmt.Errorf("failed to unmarshal A2A response: %w (also tried array and TaskResponse). Raw result: %s", err, string(jsonRPCResponse.Result))
		}
	}

	// Use singleEvent for parsing
	events := []map[string]interface{}{singleEvent}

	// Extract TaskResponse from A2A events
	// A2A returns Task or Message events with "kind" field
	// Task has History[]Message, Message has Parts[]Part
	var taskResponse adk.TaskResponse
	taskResponse.Artifacts = make(map[string]string)
	taskResponse.Metrics = make(map[string]interface{})

	// Log the event structure for debugging
	c.logger.WithFields(logrus.Fields{
		"session_id": req.SessionID,
		"event_keys": func() []string {
			keys := make([]string, 0, len(singleEvent))
			for k := range singleEvent {
				keys = append(keys, k)
			}
			return keys
		}(),
		"has_parts": func() bool {
			_, ok := singleEvent["parts"]
			return ok
		}(),
		"has_history": func() bool {
			_, ok := singleEvent["history"]
			return ok
		}(),
		"has_llm_response": func() bool {
			_, ok := singleEvent["llm_response"]
			return ok
		}(),
	}).Info("Parsing A2A event structure")

	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]

		// Try to extract from Message parts (A2A Message has Parts field)
		if parts, ok := event["parts"].([]interface{}); ok {
			c.logger.WithFields(logrus.Fields{
				"session_id":  req.SessionID,
				"parts_count": len(parts),
			}).Info("Found parts in event")

			for partIdx, part := range parts {
				if partMap, ok := part.(map[string]interface{}); ok {
					if text, ok := partMap["text"].(string); ok {
						textPreview := text
						if len(textPreview) > 500 {
							textPreview = textPreview[:500] + "..."
						}
						c.logger.WithFields(logrus.Fields{
							"session_id":   req.SessionID,
							"part_index":   partIdx,
							"text_length":  len(text),
							"text_preview": textPreview,
						}).Info("Processing part text")

						// Try to parse as TaskResponse JSON (artifacts are sent as JSON)
						var tempResp adk.TaskResponse
						if err := json.Unmarshal([]byte(text), &tempResp); err == nil {
							c.logger.WithFields(logrus.Fields{
								"session_id":     req.SessionID,
								"part_index":     partIdx,
								"artifact_count": len(tempResp.Artifacts),
								"artifact_keys": func() []string {
									keys := make([]string, 0, len(tempResp.Artifacts))
									for k := range tempResp.Artifacts {
										keys = append(keys, k)
									}
									return keys
								}(),
							}).Info("Successfully parsed TaskResponse from part")

							// Merge artifacts
							for k, v := range tempResp.Artifacts {
								taskResponse.Artifacts[k] = v
							}
							// Merge metrics
							for k, v := range tempResp.Metrics {
								taskResponse.Metrics[k] = v
							}
							if tempResp.Delta != "" {
								taskResponse.Delta = tempResp.Delta
							}
							if tempResp.Next != "" {
								taskResponse.Next = tempResp.Next
							}
						} else {
							// If not TaskResponse, might be just artifacts JSON
							var artifacts map[string]string
							if err2 := json.Unmarshal([]byte(text), &artifacts); err2 == nil {
								c.logger.WithFields(logrus.Fields{
									"session_id":     req.SessionID,
									"part_index":     partIdx,
									"artifact_count": len(artifacts),
									"artifact_keys": func() []string {
										keys := make([]string, 0, len(artifacts))
										for k := range artifacts {
											keys = append(keys, k)
										}
										return keys
									}(),
								}).Info("Successfully parsed artifacts map from part")

								for k, v := range artifacts {
									taskResponse.Artifacts[k] = v
								}
							} else {
								c.logger.WithFields(logrus.Fields{
									"session_id":   req.SessionID,
									"part_index":   partIdx,
									"parse_error":  err.Error(),
									"text_preview": textPreview,
								}).Warn("Failed to parse part as TaskResponse or artifacts map")
							}
						}
					}
				}
			}
		}

		// Try to extract from Task History (Task has History[]Message)
		if history, ok := event["history"].([]interface{}); ok {
			for _, msg := range history {
				if msgMap, ok := msg.(map[string]interface{}); ok {
					if parts, ok := msgMap["parts"].([]interface{}); ok {
						for _, part := range parts {
							if partMap, ok := part.(map[string]interface{}); ok {
								if text, ok := partMap["text"].(string); ok {
									// Try to parse as TaskResponse JSON
									var tempResp adk.TaskResponse
									if err := json.Unmarshal([]byte(text), &tempResp); err == nil {
										for k, v := range tempResp.Artifacts {
											taskResponse.Artifacts[k] = v
										}
										for k, v := range tempResp.Metrics {
											taskResponse.Metrics[k] = v
										}
										if tempResp.Delta != "" {
											taskResponse.Delta = tempResp.Delta
										}
										if tempResp.Next != "" {
											taskResponse.Next = tempResp.Next
										}
									} else {
										var artifacts map[string]string
										if err2 := json.Unmarshal([]byte(text), &artifacts); err2 == nil {
											for k, v := range artifacts {
												taskResponse.Artifacts[k] = v
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}

		// Also check for LLMResponse format (fallback for session.Event format)
		if llmResp, ok := event["llm_response"].(map[string]interface{}); ok {
			if content, ok := llmResp["content"].(map[string]interface{}); ok {
				if parts, ok := content["parts"].([]interface{}); ok {
					for _, part := range parts {
						if partMap, ok := part.(map[string]interface{}); ok {
							if text, ok := partMap["text"].(string); ok {
								var tempResp adk.TaskResponse
								if err := json.Unmarshal([]byte(text), &tempResp); err == nil {
									for k, v := range tempResp.Artifacts {
										taskResponse.Artifacts[k] = v
									}
									for k, v := range tempResp.Metrics {
										taskResponse.Metrics[k] = v
									}
									if tempResp.Delta != "" {
										taskResponse.Delta = tempResp.Delta
									}
									if tempResp.Next != "" {
										taskResponse.Next = tempResp.Next
									}
								} else {
									var artifacts map[string]string
									if err2 := json.Unmarshal([]byte(text), &artifacts); err2 == nil {
										for k, v := range artifacts {
											taskResponse.Artifacts[k] = v
										}
									}
								}
							}
						}
					}
				}
			}
		}

		// If we found artifacts, we're done
		if len(taskResponse.Artifacts) > 0 {
			break
		}
	}

	c.logger.WithFields(logrus.Fields{
		"session_id": req.SessionID,
		"artifacts":  len(taskResponse.Artifacts),
		"artifact_keys": func() []string {
			keys := make([]string, 0, len(taskResponse.Artifacts))
			for k := range taskResponse.Artifacts {
				keys = append(keys, k)
			}
			return keys
		}(),
		"response_size": len(responseBody),
	}).Info("Task executed successfully via JSON-RPC (A2A)")

	return &taskResponse, nil
}

// executeTaskHTTP executes a task using HTTP REST (legacy)
func (c *Client) executeTaskHTTP(ctx context.Context, req *adk.TaskRequest) (*adk.TaskResponse, error) {
	// Marshal request
	requestBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	// HTTP REST uses /task endpoint (not /invoke, which is for JSON-RPC)
	url := c.baseURL + "/task"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "ExplainIQ-Google-ADK-Client/1.0")
	httpReq.Header.Set("X-ADK-Version", "1.0")

	// Add authentication if available (for Cloud Run service-to-service auth)
	if c.authClient != nil && strings.HasPrefix(c.baseURL, "https://") {
		token, err := c.authClient.GetIDToken(ctx, c.baseURL)
		if err != nil {
			c.logger.WithError(err).Warn("Failed to get ID token, proceeding without authentication")
		} else {
			httpReq.Header.Set("Authorization", "Bearer "+token)
		}
	}

	// Execute request
	c.logger.WithFields(logrus.Fields{
		"url":        url,
		"session_id": req.SessionID,
		"step":       req.Step,
	}).Debug("Executing task via HTTP REST client")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		var errorResp struct {
			Error   string `json:"error"`
			Details string `json:"details"`
		}
		if err := json.Unmarshal(responseBody, &errorResp); err == nil {
			return nil, fmt.Errorf("agent error: %s - %s", errorResp.Error, errorResp.Details)
		}
		return nil, fmt.Errorf("agent error: HTTP %d - %s", resp.StatusCode, string(responseBody))
	}

	// Parse response
	var taskResponse adk.TaskResponse
	if err := json.Unmarshal(responseBody, &taskResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"session_id":    req.SessionID,
		"artifacts":     len(taskResponse.Artifacts),
		"response_size": len(responseBody),
	}).Debug("Task executed successfully via HTTP REST client")

	return &taskResponse, nil
}

// Health checks the health of the remote agent
func (c *Client) Health(ctx context.Context) error {
	// For A2A, we can check the AgentCard endpoint
	if c.useA2A {
		agentCardURL := c.baseURL
		if !strings.HasSuffix(agentCardURL, "/") {
			agentCardURL += "/"
		}
		agentCardURL += ".well-known/agent-card"

		req, err := http.NewRequestWithContext(ctx, "GET", agentCardURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create health check request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("health check failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("health check failed with status %d", resp.StatusCode)
		}

		return nil
	}

	// For HTTP REST, use /healthz endpoint
	url := c.baseURL + "/healthz"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	// Add authentication if available (for Cloud Run service-to-service auth)
	if c.authClient != nil && strings.HasPrefix(c.baseURL, "https://") {
		token, err := c.authClient.GetIDToken(ctx, c.baseURL)
		if err != nil {
			c.logger.WithError(err).Warn("Failed to get ID token for health check, proceeding without authentication")
		} else {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	return nil
}

// invocationContext implements agent.InvocationContext
type invocationContext struct {
	ctx     context.Context
	session session.Session
}

func (ic *invocationContext) Context() context.Context {
	return ic.ctx
}

func (ic *invocationContext) Session() session.Session {
	return ic.session
}
