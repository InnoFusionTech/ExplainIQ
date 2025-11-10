package google

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/server/adka2a"
	"google.golang.org/adk/session"
)

// A2AServer represents a Google ADK A2A server
type A2AServer struct {
	agent     agent.Agent
	port      string
	baseURL   *url.URL
	listener  net.Listener
	server    *http.Server
	logger    interface {
		Infof(format string, args ...interface{})
		Errorf(format string, args ...interface{})
		Fatalf(format string, args ...interface{})
	}
}

// NewA2AServer creates a new A2A server
func NewA2AServer(agent agent.Agent, port string, logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}) (*A2AServer, error) {
	if port == "" {
		port = "8080"
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to port %s: %w", port, err)
	}

	baseURL := &url.URL{
		Scheme: "http",
		Host:   listener.Addr().String(),
	}

	// If running on Cloud Run, use HTTPS and get URL from environment
	if os.Getenv("K_SERVICE") != "" {
		baseURL.Scheme = "https"
		// Try to get the service URL from environment
		if serviceURL := os.Getenv("SERVICE_URL"); serviceURL != "" {
			parsedURL, err := url.Parse(serviceURL)
			if err == nil {
				baseURL = parsedURL
			}
		}
	}

	return &A2AServer{
		agent:    agent,
		port:     port,
		baseURL:  baseURL,
		listener: listener,
		logger:   logger,
	}, nil
}

// Start starts the A2A server
func (s *A2AServer) Start() error {
	// Create AgentCard
	agentCard := &a2a.AgentCard{
		Name:               s.agent.Name(),
		Skills:             adka2a.BuildAgentSkills(s.agent),
		PreferredTransport: a2a.TransportProtocolJSONRPC,
		URL:                s.baseURL.JoinPath("/invoke").String(),
		Capabilities:      a2a.AgentCapabilities{Streaming: true},
	}

	// Create executor
	executor := adka2a.NewExecutor(adka2a.ExecutorConfig{
		RunnerConfig: runner.Config{
			AppName:        s.agent.Name(),
			Agent:          s.agent,
			SessionService: session.InMemoryService(),
		},
	})

	// Create HTTP mux
	mux := http.NewServeMux()
	mux.Handle(a2asrv.WellKnownAgentCardPath, a2asrv.NewStaticAgentCardHandler(agentCard))

	requestHandler := a2asrv.NewHandler(executor)
	mux.Handle("/invoke", a2asrv.NewJSONRPCHandler(requestHandler))

	// Health check endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	s.logger.Infof("A2A server started on %s", s.baseURL.String())
	s.logger.Infof("AgentCard available at %s", s.baseURL.JoinPath(a2asrv.WellKnownAgentCardPath).String())

	s.server = &http.Server{
		Handler: mux,
	}

	return s.server.Serve(s.listener)
}

// Stop stops the A2A server
func (s *A2AServer) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// GetAgentCardURL returns the URL where the AgentCard is available
func (s *A2AServer) GetAgentCardURL() string {
	return s.baseURL.JoinPath(a2asrv.WellKnownAgentCardPath).String()
}

// GetBaseURL returns the base URL of the server
func (s *A2AServer) GetBaseURL() string {
	return s.baseURL.String()
}


