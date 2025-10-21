package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/explainiq/agent/internal/auth"
	"github.com/explainiq/agent/internal/cost_tracker"
	"github.com/explainiq/agent/internal/quota"
	"github.com/explainiq/agent/internal/rate_limiter"
	"github.com/explainiq/agent/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Session represents a learning session
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

// SessionResult represents the final result of a session
type SessionResult struct {
	Lesson      string            `json:"lesson"`
	Images      map[string]string `json:"images"`
	Summary     string            `json:"summary"`
	Duration    time.Duration     `json:"duration"`
	CompletedAt time.Time         `json:"completed_at"`
}

// SessionStep represents a step in the session workflow
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

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"session_id"`
	StepID    string                 `json:"step_id,omitempty"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// CreateSessionRequest represents the request to create a new session
type CreateSessionRequest struct {
	Topic string `json:"topic"`
}

// CreateSessionResponse represents the response for creating a session
type CreateSessionResponse struct {
	ID string `json:"id"`
}

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"session_id"`
	StepID    string                 `json:"step_id,omitempty"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// Orchestrator manages learning sessions
type Orchestrator struct {
	sessions     map[string]*Session
	mu           sync.RWMutex
	logger       *logrus.Logger
	clients      map[string][]chan SSEEvent
	clientsMu    sync.RWMutex
	pipeline     *Pipeline
	authClient   *auth.Client
	quotaManager *quota.QuotaManager
}

// NewOrchestrator creates a new orchestrator instance
func NewOrchestrator() *Orchestrator {
	// Create pipeline with default config
	config := DefaultPipelineConfig()
	pipeline, err := NewPipeline(config)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create pipeline")
	}

	// Create auth client
	serviceURL := os.Getenv("SERVICE_URL")
	if serviceURL == "" {
		serviceURL = "http://localhost:8080"
	}
	authClient := auth.NewClient(serviceURL)

	// Create storage client for cost tracking
	storageClient, err := storage.NewFirestoreClient(context.Background(), os.Getenv("GCP_PROJECT_ID"))
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create storage client")
	}

	// Create rate limiter (10 requests per second, burst of 20)
	rateLimiter := rate_limiter.NewLimiter(10.0, 20)

	// Create cost tracker
	costTracker := cost_tracker.NewCostTracker(storageClient)

	// Create quota manager
	quotaManager := quota.NewQuotaManager(rateLimiter, costTracker)

	return &Orchestrator{
		sessions:     make(map[string]*Session),
		logger:       logrus.New(),
		clients:      make(map[string][]chan SSEEvent),
		pipeline:     pipeline,
		authClient:   authClient,
		quotaManager: quotaManager,
	}
}

// CreateSession creates a new learning session
func (o *Orchestrator) CreateSession(topic string) *Session {
	o.mu.Lock()
	defer o.mu.Unlock()

	sessionID := uuid.New().String()
	session := &Session{
		ID:        sessionID,
		Topic:     topic,
		Status:    "created",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Steps:     make([]SessionStep, 0),
		Metadata:  make(map[string]interface{}),
	}

	o.sessions[sessionID] = session
	o.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"topic":      topic,
	}).Info("Session created")

	return session
}

// GetSession retrieves a session by ID
func (o *Orchestrator) GetSession(id string) (*Session, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	session, exists := o.sessions[id]
	return session, exists
}

// UpdateSession updates a session
func (o *Orchestrator) UpdateSession(session *Session) {
	o.mu.Lock()
	defer o.mu.Unlock()

	session.UpdatedAt = time.Now()
	o.sessions[session.ID] = session
}

// AddClient adds a client to receive SSE events for a session
func (o *Orchestrator) AddClient(sessionID string, client chan SSEEvent) {
	o.clientsMu.Lock()
	defer o.clientsMu.Unlock()

	o.clients[sessionID] = append(o.clients[sessionID], client)
}

// RemoveClient removes a client from receiving SSE events
func (o *Orchestrator) RemoveClient(sessionID string, client chan SSEEvent) {
	o.clientsMu.Lock()
	defer o.clientsMu.Unlock()

	clients := o.clients[sessionID]
	for i, c := range clients {
		if c == client {
			o.clients[sessionID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}
}

// BroadcastEvent broadcasts an SSE event to all clients for a session
func (o *Orchestrator) BroadcastEvent(sessionID string, event SSEEvent) {
	o.clientsMu.RLock()
	clients := o.clients[sessionID]
	o.clientsMu.RUnlock()

	for _, client := range clients {
		select {
		case client <- event:
		default:
			// Client channel is full, skip
		}
	}
}

// RunSession executes a session workflow using the pipeline
func (o *Orchestrator) RunSession(sessionID string) {
	ctx := context.Background()

	// Execute the pipeline
	if err := o.pipeline.runPipeline(ctx, sessionID, o); err != nil {
		o.logger.WithFields(logrus.Fields{
			"session_id": sessionID,
			"error":      err,
		}).Error("Pipeline execution failed")

		// Update session status to failed
		session, exists := o.GetSession(sessionID)
		if exists {
			session.Status = "failed"
			o.UpdateSession(session)
		}
	}
}

// HTTP Handlers

// createSessionHandler handles POST /api/sessions
func (o *Orchestrator) createSessionHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Topic == "" {
		http.Error(w, "Topic is required", http.StatusBadRequest)
		return
	}

	session := o.CreateSession(req.Topic)
	response := CreateSessionResponse{ID: session.ID}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// runSessionHandler handles POST /api/sessions/{id}/run
func (o *Orchestrator) runSessionHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	session, exists := o.GetSession(sessionID)
	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	if session.Status == "running" {
		http.Error(w, "Session is already running", http.StatusConflict)
		return
	}

	// Set up SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Create client channel
	client := make(chan SSEEvent, 10)
	o.AddClient(sessionID, client)
	defer o.RemoveClient(sessionID, client)

	// Start session execution in goroutine
	go o.RunSession(sessionID)

	// Stream events to client
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Send initial event
	fmt.Fprintf(w, "data: %s\n\n", `{"type":"connected","session_id":"`+sessionID+`"}`)
	flusher.Flush()

	// Stream events
	for {
		select {
		case event := <-client:
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", string(data))
			flusher.Flush()

			// Close connection on final event
			if event.Type == "final" {
				return
			}
		case <-r.Context().Done():
			return
		}
	}
}

// getSessionResultHandler handles GET /api/sessions/{id}/result
func (o *Orchestrator) getSessionResultHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	session, exists := o.GetSession(sessionID)
	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	if session.Status != "completed" {
		http.Error(w, "Session not completed", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session.Result)
}

// heartbeatHandler handles GET /health
func (o *Orchestrator) heartbeatHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"sessions":  len(o.sessions),
	})
}

// setupRoutes sets up the HTTP routes
func (o *Orchestrator) setupRoutes() *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS for development
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Routes
	r.Get("/health", o.heartbeatHandler)
	r.Route("/api", func(r chi.Router) {
		r.Route("/sessions", func(r chi.Router) {
			// Public endpoints (no auth required, but quota limited)
			r.Group(func(r chi.Router) {
				// Add quota middleware for rate limiting and cost tracking
				r.Use(o.quotaMiddleware())
				r.Post("/", o.createSessionHandler)
				r.Post("/{id}/run", o.runSessionHandler)
			})

			// Protected endpoints (auth required)
			r.Group(func(r chi.Router) {
				// Add service authentication middleware
				r.Use(auth.ServiceAuthMiddleware(o.authClient))
				r.Get("/{id}/result", o.getSessionResultHandler)
			})
		})
	})

	return r
}

// quotaMiddleware creates a quota middleware for the orchestrator
func (o *Orchestrator) quotaMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client IP
			ip := r.RemoteAddr
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				ip = xff
			}

			// Check rate limit
			if !o.quotaManager.RateLimiter.Allow(ip) {
				o.logger.WithFields(logrus.Fields{
					"ip":   ip,
					"path": r.URL.Path,
				}).Warn("Rate limit exceeded")

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(gin.H{
					"error":       "Rate limit exceeded",
					"message":     "Too many requests from your IP. Please try again later.",
					"retry_after": 60,
					"quota_type":  "rate_limit",
				})
				return
			}

			// Continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	// Create orchestrator
	orchestrator := NewOrchestrator()

	// Setup routes
	router := orchestrator.setupRoutes()

	// Create server
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		orchestrator.logger.Info("Starting orchestrator server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			orchestrator.logger.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	orchestrator.logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		orchestrator.logger.Fatalf("Server forced to shutdown: %v", err)
	}

	orchestrator.logger.Info("Server exited")
}
