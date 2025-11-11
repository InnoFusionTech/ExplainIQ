package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/InnoFusionTech/ExplainIQ/internal/auth"
	"github.com/InnoFusionTech/ExplainIQ/internal/brainprint"
	"github.com/InnoFusionTech/ExplainIQ/internal/cost_tracker"
	"github.com/InnoFusionTech/ExplainIQ/internal/quota"
	"github.com/InnoFusionTech/ExplainIQ/internal/rate_limiter"
	"github.com/InnoFusionTech/ExplainIQ/internal/storage"
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
	Images      map[string]string `json:"images,omitempty"`
	Summary     string            `json:"summary,omitempty"`
	Duration    time.Duration     `json:"duration,omitempty"`
	CompletedAt time.Time         `json:"completed_at,omitempty"`
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
	Topic           string `json:"topic"`
	ExplanationType string `json:"explanation_type,omitempty"` // standard, visualization, simple, analogy
}

// CreateSessionResponse represents the response for creating a session
type CreateSessionResponse struct {
	ID string `json:"id"`
}

// SavedLesson represents a saved lesson for later viewing
type SavedLesson struct {
	ID          string                 `json:"id"`
	SessionID   string                 `json:"session_id"`
	UserID      string                 `json:"user_id"`
	Topic       string                 `json:"topic"`
	Title       string                 `json:"title"`
	ExplanationType string             `json:"explanation_type"`
	Result      *SessionResult         `json:"result,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// Orchestrator manages learning sessions
type Orchestrator struct {
	sessions      map[string]*Session
	savedLessons  map[string]*SavedLesson // userID -> []SavedLesson (stored by userID)
	mu            sync.RWMutex
	logger        *logrus.Logger
	clients       map[string][]chan SSEEvent
	clientsMu     sync.RWMutex
	pipeline      *Pipeline
	authClient    *auth.Client
	quotaManager  *quota.QuotaManager
	brainprintSvc *brainprint.Service
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

	// Create storage client for cost tracking (optional)
	var storageClient storage.Storage
	var costTracker *cost_tracker.CostTracker

	gcpProjectID := os.Getenv("GCP_PROJECT_ID")
	if gcpProjectID != "" {
		client, err := storage.NewFirestoreClient(context.Background(), gcpProjectID, "sessions")
		if err != nil {
			logrus.WithError(err).Warn("Failed to create Firestore client, continuing without cost tracking")
			storageClient = nil
			costTracker = nil
		} else {
			storageClient = client
			costTracker = cost_tracker.NewCostTracker(storageClient)
		}
	} else {
		logrus.Warn("GCP_PROJECT_ID not set, continuing without cost tracking")
		storageClient = nil
		costTracker = nil
	}

	// Create rate limiter (10 requests per second, burst of 20)
	rateLimiter := rate_limiter.NewLimiter(10.0, 20)

	// Create quota manager
	var quotaManager *quota.QuotaManager
	if costTracker != nil {
		quotaManager = quota.NewQuotaManager(rateLimiter, costTracker)
	} else {
		// Create quota manager without cost tracking
		quotaManager = quota.NewQuotaManager(rateLimiter, nil)
	}

	// Create BrainPrint service (using in-memory storage for now)
	// Cast storage client to brainprint.StorageInterface if available
	var brainprintStorage brainprint.StorageInterface
	if storageClient != nil {
		brainprintStorage = storageClient
	}
	brainprintSvc := brainprint.NewService(brainprintStorage)

	return &Orchestrator{
		sessions:      make(map[string]*Session),
		savedLessons:  make(map[string]*SavedLesson),
		logger:        logrus.New(),
		clients:       make(map[string][]chan SSEEvent),
		pipeline:      pipeline,
		authClient:    authClient,
		quotaManager:  quotaManager,
		brainprintSvc: brainprintSvc,
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
		o.logger.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to decode create session request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Topic == "" {
		o.logger.Warn("Create session request missing topic")
		http.Error(w, "Topic is required", http.StatusBadRequest)
		return
	}

	// Set default explanation type if not provided
	explanationType := req.ExplanationType
	if explanationType == "" {
		explanationType = "standard"
	}

	// Create session with error handling
	session := o.CreateSession(req.Topic)
	if session == nil {
		o.logger.Error("Failed to create session - CreateSession returned nil")
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}
	
	// Store explanation type in session metadata
	session.Metadata["explanation_type"] = explanationType
	response := CreateSessionResponse{ID: session.ID}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		o.logger.WithFields(logrus.Fields{
			"session_id": session.ID,
			"error":      err,
		}).Error("Failed to encode create session response")
	}
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

	// Send initial event (formatted to match SSEEvent structure)
	initialEvent := SSEEvent{
		Type:      "connected",
		SessionID: sessionID,
		Data: map[string]interface{}{
			"session_id": sessionID,
			"timestamp":  time.Now().Format(time.RFC3339),
		},
		Timestamp: time.Now(),
	}
	initialData, _ := json.Marshal(initialEvent)
	fmt.Fprintf(w, "data: %s\n\n", string(initialData))
	flusher.Flush()

		// Stream events
		for {
			select {
			case event := <-client:
				data, _ := json.Marshal(event)
				fmt.Fprintf(w, "data: %s\n\n", string(data))
				flusher.Flush()

				// Close connection on final events
				if event.Type == "final" || event.Type == "session_complete" || event.Type == "session_error" {
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

// getBrainPrintHandler handles GET /api/brainprint/:userID
func (o *Orchestrator) getBrainPrintHandler(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		o.logger.Warn("BrainPrint request missing userID")
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	profile, err := o.brainprintSvc.GetBrainPrint(ctx, userID)
	if err != nil {
		o.logger.WithFields(logrus.Fields{
			"user_id": userID,
			"error":   err,
		}).Error("Failed to get BrainPrint")
		// Return empty profile instead of error to allow frontend to display empty state
		profile = brainprint.NewUserLearningProfile(userID)
	}

	// Get a random tip
	tip := o.getRandomTip()

	// Format response with tip
	response := map[string]interface{}{
		"userID":         profile.UserID,
		"totalSessions": profile.TotalSessions,
		"byType":         profile.ByType,
		"recommendedType": profile.RecommendedType,
		"dominantStyle":   profile.RecommendedType,
		"usage":          profile.ByType,
		"tip":            tip,
		"lastUpdated":    profile.LastUpdated,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		o.logger.WithFields(logrus.Fields{
			"user_id": userID,
			"error":   err,
		}).Error("Failed to encode BrainPrint response")
	}
}

// sessionCompleteHandler handles POST /api/session/complete
func (o *Orchestrator) sessionCompleteHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID       string `json:"session_id"`
		UserID          string `json:"user_id"`
		ExplanationType string `json:"explanation_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		http.Error(w, "Session ID is required", http.StatusBadRequest)
		return
	}

	// Get user ID from request or use session ID
	userID := req.UserID
	if userID == "" {
		userID = req.SessionID
	}

	// Get explanation type from request or default to standard
	explanationType := req.ExplanationType
	if explanationType == "" {
		explanationType = "standard"
	}

	ctx := r.Context()

	// Track session completion
	if err := o.brainprintSvc.TrackSession(ctx, userID, explanationType, true); err != nil {
		o.logger.WithFields(logrus.Fields{
			"session_id": req.SessionID,
			"user_id":    userID,
			"error":      err,
		}).Error("Failed to track session completion")
		http.Error(w, "Failed to track session", http.StatusInternalServerError)
		return
	}

	// Get updated BrainPrint
	profile, err := o.brainprintSvc.GetBrainPrint(ctx, userID)
	if err != nil {
		o.logger.WithFields(logrus.Fields{
			"user_id": userID,
			"error":   err,
		}).Error("Failed to get updated BrainPrint")
		http.Error(w, "Failed to retrieve BrainPrint", http.StatusInternalServerError)
		return
	}

	// Get a random tip
	tip := o.getRandomTip()

	response := map[string]interface{}{
		"success":       true,
		"session_id":    req.SessionID,
		"user_id":       userID,
		"dominantStyle": profile.RecommendedType,
		"totalSessions": profile.TotalSessions,
		"usage":         profile.ByType,
		"tip":           tip,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getTipsHandler handles GET /api/tips
func (o *Orchestrator) getTipsHandler(w http.ResponseWriter, r *http.Request) {
	tip := o.getRandomTip()
	
	response := map[string]interface{}{
		"tip": tip,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getRandomTip returns a random learning tip
func (o *Orchestrator) getRandomTip() string {
	tips := []string{
		"Try visualization next to boost retention by 30%!",
		"Analogy explanations help connect new ideas to familiar concepts.",
		"Simple explanations are great for complex topics - try them!",
		"Mix different explanation types to discover your learning style.",
		"Visual learners benefit most from diagram-based explanations.",
		"Practice with toy examples to reinforce core mechanisms.",
		"Real-life applications make abstract concepts concrete.",
		"Memory hooks help you remember key concepts longer.",
		"Best practices save time and prevent common mistakes.",
		"Try different explanation types to find what works best for you!",
		"Visualization mode creates interactive diagrams for better understanding.",
		"Standard explanations provide comprehensive coverage of topics.",
		"Simple explanations break down complex ideas into digestible parts.",
		"Analogy explanations use familiar concepts to explain new ones.",
	}

	// Use current time to pseudo-randomly select a tip
	index := int(time.Now().Unix()) % len(tips)
	return tips[index]
}

// saveLessonHandler handles POST /api/saved
func (o *Orchestrator) saveLessonHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID       string          `json:"session_id"`
		UserID          string          `json:"user_id"`
		Title           string          `json:"title,omitempty"`
		ExplanationType string          `json:"explanation_type,omitempty"`
		Result          *SessionResult  `json:"result,omitempty"`
	}

	w.Header().Set("Content-Type", "application/json")

	// Read body for logging and decoding
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Invalid request body",
			"message": "Failed to read request body",
		})
		return
	}

	// Log the request body for debugging
	o.logger.WithFields(logrus.Fields{
		"body": string(bodyBytes),
	}).Debug("Save lesson request body")

	// Restore body for decoding
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Decode JSON
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		o.logger.WithFields(logrus.Fields{
			"error": err.Error(),
			"body":  string(bodyBytes),
		}).Error("Failed to decode save lesson request")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
		return
	}

	if req.SessionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Session ID is required",
			"message": "Session ID is required",
		})
		return
	}

	// Get user ID from request or use session ID
	userID := req.UserID
	if userID == "" {
		userID = req.SessionID
	}

	// Get session to get topic
	session, exists := o.GetSession(req.SessionID)
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Session not found",
			"message": "Session not found",
		})
		return
	}

	// Generate title if not provided
	title := req.Title
	if title == "" {
		title = session.Topic
	}

	// Get explanation type from request or session metadata, default to "standard"
	explanationType := req.ExplanationType
	if explanationType == "" {
		if expType, ok := session.Metadata["explanation_type"].(string); ok && expType != "" {
			explanationType = expType
		} else {
			explanationType = "standard"
		}
	}

	// Use session result if not provided in request
	result := req.Result
	if result == nil {
		// Use result from session if available
		if session.Result != nil {
			result = session.Result
		} else {
			// Create a minimal result
			result = &SessionResult{
				Lesson:      "",
				Images:      make(map[string]string),
				Summary:     "",
				Duration:    0,
				CompletedAt: time.Now(),
			}
		}
	}

	// Ensure result has required fields
	// If result.Lesson is empty, try to get it from session.Result
	if result.Lesson == "" {
		if session.Result != nil && session.Result.Lesson != "" {
			result.Lesson = session.Result.Lesson
		} else {
			// Log warning if lesson is still empty
			o.logger.WithFields(logrus.Fields{
				"session_id": req.SessionID,
				"has_result": req.Result != nil,
				"has_session_result": session.Result != nil,
			}).Warn("Lesson content is empty, saving with empty lesson")
		}
	}
	if result.Images == nil {
		result.Images = make(map[string]string)
	}
	if session.Result != nil && len(result.Images) == 0 && len(session.Result.Images) > 0 {
		result.Images = session.Result.Images
	}
	if result.Summary == "" && session.Result != nil {
		result.Summary = session.Result.Summary
	}
	if result.CompletedAt.IsZero() {
		result.CompletedAt = time.Now()
	}

	// Log what we're saving for debugging
	o.logger.WithFields(logrus.Fields{
		"session_id":    req.SessionID,
		"lesson_length":  len(result.Lesson),
		"has_images":    len(result.Images) > 0,
		"has_summary":   result.Summary != "",
		"title":         title,
	}).Info("Saving lesson with data")

	// Create saved lesson
	savedID := uuid.New().String()
	savedLesson := &SavedLesson{
		ID:              savedID,
		SessionID:        req.SessionID,
		UserID:          userID,
		Topic:           session.Topic,
		Title:           title,
		ExplanationType: explanationType,
		Result:          result,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Store saved lesson
	o.mu.Lock()
	o.savedLessons[savedID] = savedLesson
	o.mu.Unlock()

	o.logger.WithFields(logrus.Fields{
		"saved_id":   savedID,
		"session_id": req.SessionID,
		"user_id":    userID,
		"topic":      session.Topic,
	}).Info("Lesson saved")

	response := map[string]interface{}{
		"success": true,
		"id":      savedID,
		"message": "Lesson saved successfully",
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// getSavedLessonsHandler handles GET /api/saved/{userID}
func (o *Orchestrator) getSavedLessonsHandler(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	w.Header().Set("Content-Type", "application/json")
	
	if userID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "User ID is required",
			"message": "User ID is required",
		})
		return
	}

	o.mu.RLock()
	savedLessons := make([]*SavedLesson, 0)
	for _, lesson := range o.savedLessons {
		if lesson.UserID == userID {
			savedLessons = append(savedLessons, lesson)
		}
	}
	o.mu.RUnlock()

	// Sort by created date (newest first)
	for i := 0; i < len(savedLessons)-1; i++ {
		for j := i + 1; j < len(savedLessons); j++ {
			if savedLessons[i].CreatedAt.Before(savedLessons[j].CreatedAt) {
				savedLessons[i], savedLessons[j] = savedLessons[j], savedLessons[i]
			}
		}
	}

	response := map[string]interface{}{
		"lessons": savedLessons,
		"count":   len(savedLessons),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getSavedLessonHandler handles GET /api/saved/{userID}/{id}
func (o *Orchestrator) getSavedLessonHandler(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	savedID := chi.URLParam(r, "id")
	w.Header().Set("Content-Type", "application/json")

	if userID == "" || savedID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "User ID and Saved ID are required",
			"message": "User ID and Saved ID are required",
		})
		return
	}

	o.mu.RLock()
	savedLesson, exists := o.savedLessons[savedID]
	o.mu.RUnlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Saved lesson not found",
			"message": "Saved lesson not found",
		})
		return
	}

	if savedLesson.UserID != userID {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Unauthorized",
			"message": "Unauthorized",
		})
		return
	}

	json.NewEncoder(w).Encode(savedLesson)
}

// deleteSavedLessonHandler handles DELETE /api/saved/{userID}/{id}
func (o *Orchestrator) deleteSavedLessonHandler(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	savedID := chi.URLParam(r, "id")
	w.Header().Set("Content-Type", "application/json")

	if userID == "" || savedID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "User ID and Saved ID are required",
			"message": "User ID and Saved ID are required",
		})
		return
	}

	o.mu.Lock()
	savedLesson, exists := o.savedLessons[savedID]
	if exists && savedLesson.UserID == userID {
		delete(o.savedLessons, savedID)
		o.mu.Unlock()

		o.logger.WithFields(logrus.Fields{
			"saved_id": savedID,
			"user_id":  userID,
		}).Info("Saved lesson deleted")

		response := map[string]interface{}{
			"success": true,
			"message": "Saved lesson deleted successfully",
		}

		json.NewEncoder(w).Encode(response)
		return
	}
	o.mu.Unlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Saved lesson not found",
			"message": "Saved lesson not found",
		})
		return
	}

	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   "Unauthorized",
		"message": "Unauthorized",
	})
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
	r.Get("/healthz", o.heartbeatHandler) // Cloud Run health check endpoint
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
				// TODO: Add service authentication middleware for Chi router
				// r.Use(auth.ServiceAuthMiddleware(o.authClient))
				r.Get("/{id}/result", o.getSessionResultHandler)
			})
		})

		// BrainPrint endpoints (no quota middleware - read-only, lightweight)
		r.Get("/brainprint/{userID}", o.getBrainPrintHandler)
		
		// Session completion endpoint (no quota middleware - called after session completes)
		r.Post("/session/complete", o.sessionCompleteHandler)
		
		// Tips endpoint (no quota middleware - read-only, lightweight)
		r.Get("/tips", o.getTipsHandler)

		// Saved lessons endpoints
		r.Route("/saved", func(r chi.Router) {
			r.Post("/", o.saveLessonHandler)
			r.Get("/{userID}", o.getSavedLessonsHandler)
			r.Get("/{userID}/{id}", o.getSavedLessonHandler)
			r.Delete("/{userID}/{id}", o.deleteSavedLessonHandler)
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
				json.NewEncoder(w).Encode(map[string]interface{}{
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
