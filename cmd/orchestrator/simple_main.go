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
	sessions  map[string]*Session
	mu        sync.RWMutex
	logger    *logrus.Logger
	clients   map[string][]chan SSEEvent
	clientsMu sync.RWMutex
}

// NewOrchestrator creates a new orchestrator instance
func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		sessions: make(map[string]*Session),
		logger:   logrus.New(),
		clients:  make(map[string][]chan SSEEvent),
	}
}

// CreateSession creates a new learning session
func (o *Orchestrator) CreateSession(topic string) *Session {
	o.mu.Lock()
	defer o.mu.Unlock()

	session := &Session{
		ID:        uuid.New().String(),
		Topic:     topic,
		Status:    "created",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Steps:     []SessionStep{},
		Metadata:  make(map[string]interface{}),
	}

	o.sessions[session.ID] = session
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

// RunSession executes a session workflow
func (o *Orchestrator) RunSession(sessionID string) {
	ctx := context.Background()

	// Get session
	session, exists := o.GetSession(sessionID)
	if !exists {
		o.logger.WithField("session_id", sessionID).Error("Session not found")
		return
	}

	// Update session status
	session.Status = "running"
	o.UpdateSession(session)

	// Broadcast session start
	o.BroadcastEvent(sessionID, SSEEvent{
		Type:      "session-start",
		SessionID: sessionID,
		Data: map[string]interface{}{
			"topic": session.Topic,
		},
		Timestamp: time.Now(),
	})

	// Simulate some work
	time.Sleep(2 * time.Second)

	// Update session status
	session.Status = "completed"
	session.Result = &SessionResult{
		Lesson:      "This is a sample lesson about " + session.Topic,
		Images:      map[string]string{"diagram1": "https://example.com/diagram1.png"},
		Summary:     "Summary of " + session.Topic,
		Duration:    2 * time.Second,
		CompletedAt: time.Now(),
	}
	o.UpdateSession(session)

	// Broadcast session completion
	o.BroadcastEvent(sessionID, SSEEvent{
		Type:      "session-complete",
		SessionID: sessionID,
		Data: map[string]interface{}{
			"result": session.Result,
		},
		Timestamp: time.Now(),
	})
}

// HTTP Handlers
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
	json.NewEncoder(w).Encode(response)
}

func (o *Orchestrator) runSessionHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		http.Error(w, "Session ID is required", http.StatusBadRequest)
		return
	}

	// Check if session exists
	_, exists := o.GetSession(sessionID)
	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Start session in background
	go o.RunSession(sessionID)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

func (o *Orchestrator) getSessionResultHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		http.Error(w, "Session ID is required", http.StatusBadRequest)
		return
	}

	session, exists := o.GetSession(sessionID)
	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func (o *Orchestrator) runSessionSSEHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		http.Error(w, "Session ID is required", http.StatusBadRequest)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create client channel
	clientChan := make(chan SSEEvent, 10)

	// Add client
	o.clientsMu.Lock()
	o.clients[sessionID] = append(o.clients[sessionID], clientChan)
	o.clientsMu.Unlock()

	// Remove client when done
	defer func() {
		o.clientsMu.Lock()
		clients := o.clients[sessionID]
		for i, client := range clients {
			if client == clientChan {
				o.clients[sessionID] = append(clients[:i], clients[i+1:]...)
				break
			}
		}
		o.clientsMu.Unlock()
		close(clientChan)
	}()

	// Send events to client
	for {
		select {
		case event := <-clientChan:
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (o *Orchestrator) heartbeatHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "healthy",
		"service":   "orchestrator",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

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
			r.Post("/", o.createSessionHandler)
			r.Post("/{id}/run", o.runSessionHandler)
			r.Get("/{id}/result", o.getSessionResultHandler)
			r.Get("/{id}/run", o.runSessionSSEHandler)
		})
	})

	return r
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



