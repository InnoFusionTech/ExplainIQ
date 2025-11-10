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
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/remoteagent"
	"google.golang.org/adk/session"
	"github.com/sirupsen/logrus"
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

	// Use A2A protocol if available
	if c.useA2A && c.remoteAgent != nil {
		return c.executeTaskA2A(ctx, req)
	}

	// Fall back to HTTP REST
	return c.executeTaskHTTP(ctx, req)
}

// executeTaskA2A executes a task using A2A protocol
// Note: This is a simplified implementation. For full A2A support, we should use
// the A2A client library directly. For now, we fall back to HTTP REST.
func (c *Client) executeTaskA2A(ctx context.Context, req *adk.TaskRequest) (*adk.TaskResponse, error) {
	// For now, fall back to HTTP REST for A2A as well
	// Full A2A implementation would require more complex session management
	c.logger.WithFields(logrus.Fields{
		"session_id": req.SessionID,
		"step":       req.Step,
	}).Debug("A2A protocol not fully implemented, falling back to HTTP REST")
	
	return c.executeTaskHTTP(ctx, req)
}

// executeTaskHTTP executes a task using HTTP REST (legacy)
func (c *Client) executeTaskHTTP(ctx context.Context, req *adk.TaskRequest) (*adk.TaskResponse, error) {
	// Marshal request
	requestBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
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

