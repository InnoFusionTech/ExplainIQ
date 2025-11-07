package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/InnoFusionTech/ExplainIQ/internal/adk"
	"github.com/InnoFusionTech/ExplainIQ/internal/auth"
	"github.com/InnoFusionTech/ExplainIQ/internal/elastic"
	"github.com/InnoFusionTech/ExplainIQ/internal/llm"
	"github.com/sirupsen/logrus"
)

// ContextDoc represents a document retrieved from context search
type ContextDoc struct {
	Doc     elastic.Doc
	Score   float64
	Snippet string
}

// RetrieverSearchResult is an alias for the retriever's SearchResult type
type RetrieverSearchResult = elastic.SearchResult

// PipelineConfig represents configuration for the pipeline
type PipelineConfig struct {
	MaxRetries     int               `json:"max_retries"`
	RetryDelay     time.Duration     `json:"retry_delay"`
	StepTimeout    time.Duration     `json:"step_timeout"`
	ContextTopK    int               `json:"context_top_k"`
	ElasticIndex   string            `json:"elastic_index"`
	AgentBaseURLs  map[string]string `json:"agent_base_urls"`
	ElasticBaseURL string            `json:"elastic_base_url"`
	ElasticAPIKey  string            `json:"elastic_api_key"`
	LLMProjectID   string            `json:"llm_project_id"`
	LLMLocation    string            `json:"llm_location"`
}

// DefaultPipelineConfig returns the default pipeline configuration
func DefaultPipelineConfig() PipelineConfig {
	// Get environment variables for service URLs
	elasticURL := os.Getenv("ELASTIC_URL")
	if elasticURL == "" {
		elasticURL = "http://elasticsearch:9200"
	}
	
	return PipelineConfig{
		MaxRetries:   3,
		RetryDelay:   2 * time.Second,
		StepTimeout:  5 * time.Minute,
		ContextTopK:  5,
		ElasticIndex: "lessons",
		AgentBaseURLs: map[string]string{
			"summarizer": "http://agent-summarizer:8081",
			"explainer":  "http://agent-explainer:8082",
			"visualizer": "http://agent-visualizer:8084",
			"critic":     "http://agent-critic:8083",
		},
		ElasticBaseURL: elasticURL,
		ElasticAPIKey:  "",
		LLMProjectID:   "explainiq-project",
		LLMLocation:    "us-central1",
	}
}

// PipelineStep represents a step in the pipeline
type PipelineStep struct {
	Name            string            `json:"name"`
	Agent           string            `json:"agent"`
	Inputs          map[string]string `json:"inputs"`
	RequiresContext bool              `json:"requires_context"`
	Retryable       bool              `json:"retryable"`
}

// PipelineResult represents the result of pipeline execution
type PipelineResult struct {
	SessionID   string                 `json:"session_id"`
	Status      string                 `json:"status"`
	Steps       []PipelineStepResult   `json:"steps"`
	FinalResult map[string]interface{} `json:"final_result"`
	Error       string                 `json:"error,omitempty"`
	Duration    time.Duration          `json:"duration"`
	CompletedAt time.Time              `json:"completed_at"`
}

// PipelineStepResult represents the result of a single pipeline step
type PipelineStepResult struct {
	StepName   string                 `json:"step_name"`
	Status     string                 `json:"status"`
	Output     map[string]string      `json:"output"`
	Error      string                 `json:"error,omitempty"`
	Duration   time.Duration          `json:"duration"`
	RetryCount int                    `json:"retry_count"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// Pipeline represents the orchestrator pipeline
type Pipeline struct {
	config           PipelineConfig
	logger           *logrus.Logger
	elasticClient    *elastic.Client
	elasticRetriever *elastic.Retriever
	embeddingClient  *llm.EmbeddingClient
	adkClients       map[string]*adk.Client
	authClient       *auth.Client
}

// NewPipeline creates a new pipeline instance
func NewPipeline(config PipelineConfig) (*Pipeline, error) {
	logger := logrus.New()

	// Initialize Elasticsearch client (optional)
	var elasticClient *elastic.Client
	var elasticRetriever *elastic.Retriever
	var err error
	
	// Try to connect to Elasticsearch, but don't fail if it's not available
	elasticClient, err = elastic.NewClient(context.Background(), config.ElasticBaseURL, config.ElasticAPIKey)
	if err != nil {
		logger.WithError(err).Warn("Elasticsearch not available, continuing without context retrieval")
		elasticClient = nil
		elasticRetriever = nil
	} else {
		// Initialize embedding client
		embeddingClient := llm.NewEmbeddingClient(config.LLMProjectID, config.LLMLocation)
		// Initialize elastic retriever
		elasticRetriever = elastic.NewRetriever(elasticClient, embeddingClient)
	}

	// Initialize auth client
	authClient := auth.NewClient("http://localhost:8080") // Orchestrator's own URL

	// Initialize ADK clients for each agent
	adkClients := make(map[string]*adk.Client)
	for agentName, baseURL := range config.AgentBaseURLs {
		adkClients[agentName] = adk.NewClient(baseURL,
			adk.WithTimeout(config.StepTimeout),
			adk.WithConfig(adk.TaskConfig{
				Timeout:     config.StepTimeout,
				MaxRetries:  config.MaxRetries,
				RetryDelay:  config.RetryDelay,
				BackoffType: "exponential",
			}),
			adk.WithLogger(logger),
		)
	}

	return &Pipeline{
		config:           config,
		logger:           logger,
		elasticClient:    elasticClient,
		elasticRetriever: elasticRetriever,
		embeddingClient:  nil, // Will be set when needed
		adkClients:       adkClients,
		authClient:       authClient,
	}, nil
}

// runPipeline executes the complete pipeline for a session
func (p *Pipeline) runPipeline(ctx context.Context, sessionID string, orchestrator *Orchestrator) error {
	session, exists := orchestrator.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	p.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"topic":      session.Topic,
	}).Info("Starting pipeline execution")

	// Update session status
	session.Status = "running"
	orchestrator.UpdateSession(session)

	// Define pipeline steps
	steps := []PipelineStep{
		{
			Name:            "summarizer",
			Agent:           "summarizer",
			Inputs:          map[string]string{"topic": session.Topic},
			RequiresContext: true,
			Retryable:       true,
		},
		{
			Name:            "explainer",
			Agent:           "explainer",
			Inputs:          map[string]string{"topic": session.Topic},
			RequiresContext: true,
			Retryable:       true,
		},
		{
			Name:            "visualizer",
			Agent:           "visualizer",
			Inputs:          map[string]string{"topic": session.Topic},
			RequiresContext: false,
			Retryable:       true,
		},
		{
			Name:            "critic",
			Agent:           "critic",
			Inputs:          map[string]string{"topic": session.Topic},
			RequiresContext: false,
			Retryable:       true,
		},
	}

	// Execute pipeline steps
	result := &PipelineResult{
		SessionID:   sessionID,
		Status:      "running",
		Steps:       make([]PipelineStepResult, 0, len(steps)),
		FinalResult: make(map[string]interface{}),
	}

	startTime := time.Now()
	defer func() {
		result.Duration = time.Since(startTime)
		result.CompletedAt = time.Now()
	}()

	// Execute each step
	// Collect outputs from previous steps to pass to subsequent steps
	previousOutputs := make(map[string]map[string]string)
	
	for i, step := range steps {
		// Merge previous step outputs into current step inputs if needed
		step = p.enrichStepInputs(step, previousOutputs)
		
		stepResult := p.executeStep(ctx, sessionID, step, orchestrator, i)
		result.Steps = append(result.Steps, stepResult)
		
		// Store outputs from completed steps for use in subsequent steps
		if stepResult.Status == "completed" {
			previousOutputs[step.Name] = stepResult.Output
			p.logger.WithFields(logrus.Fields{
				"step":           step.Name,
				"output_count":   len(stepResult.Output),
				"has_lesson":     step.Name == "explainer" && stepResult.Output["lesson"] != "",
			}).Info("Stored step outputs for use in subsequent steps")
		} else {
			p.logger.WithFields(logrus.Fields{
				"step":   step.Name,
				"status": stepResult.Status,
			}).Warn("Step did not complete successfully, outputs not stored")
		}

		// Get quota information for SSE metadata
		quotaInfo := make(map[string]interface{})
		if orchestrator.quotaManager != nil {
			if info, err := orchestrator.quotaManager.GetQuotaInfo(ctx, sessionID); err == nil {
				quotaInfo = info
			}
		}

		// Broadcast step completion
		orchestrator.BroadcastEvent(sessionID, SSEEvent{
			Type:      "step-complete",
			SessionID: sessionID,
			StepID:    fmt.Sprintf("step-%d", i+1),
			Data: map[string]interface{}{
				"step_name":   step.Name,
				"status":      stepResult.Status,
				"duration":    stepResult.Duration.Milliseconds(),
				"retry_count": stepResult.RetryCount,
				"quota_info":  quotaInfo,
			},
			Timestamp: time.Now(),
		})

		// Check if step failed and handle accordingly
		if stepResult.Status == "failed" {
			if step.Retryable && stepResult.RetryCount < p.config.MaxRetries {
				p.logger.WithFields(logrus.Fields{
					"session_id":  sessionID,
					"step":        step.Name,
					"retry_count": stepResult.RetryCount,
				}).Warn("Step failed but is retryable, continuing with next step")
				continue
			} else {
				// Non-retryable failure or max retries exceeded
				result.Status = "failed"
				result.Error = fmt.Sprintf("step %s failed: %s", step.Name, stepResult.Error)

				// Update session status
				session.Status = "failed"
				orchestrator.UpdateSession(session)

				// Broadcast final failure event
				orchestrator.BroadcastEvent(sessionID, SSEEvent{
					Type:      "pipeline-failed",
					SessionID: sessionID,
					Data: map[string]interface{}{
						"error":       result.Error,
						"failed_step": step.Name,
					},
					Timestamp: time.Now(),
				})

				return fmt.Errorf("pipeline failed at step %s: %s", step.Name, stepResult.Error)
			}
		}

		// Apply critic patch if this is the critic step
		if step.Name == "critic" && stepResult.Status == "completed" {
			// Get the lesson from explainer output (stored in previousOutputs)
			var lessonJSON string
			if explainerOutput, exists := previousOutputs["explainer"]; exists {
				if lesson, ok := explainerOutput["lesson"]; ok && lesson != "" {
					lessonJSON = lesson
				}
			}
			
			if lessonJSON == "" {
				p.logger.WithFields(logrus.Fields{
					"session_id": sessionID,
				}).Warn("Cannot apply critic patch: lesson not found in explainer output")
			} else {
				if err := p.applyCriticPatch(ctx, sessionID, lessonJSON, stepResult.Output, orchestrator); err != nil {
					p.logger.WithFields(logrus.Fields{
						"session_id": sessionID,
						"error":      err,
					}).Error("Failed to apply critic patch")
					// Continue execution even if critic patch fails
				}
			}
		}
	}

	// Pipeline completed successfully
	result.Status = "completed"

	// Create final result
	finalResult := make(map[string]interface{})
	for _, stepResult := range result.Steps {
		if stepResult.Status == "completed" {
			finalResult[stepResult.StepName] = stepResult.Output
		}
	}
	result.FinalResult = finalResult

	// Update session with final result
	session.Status = "completed"
	session.Result = &SessionResult{
		Lesson:      p.extractLesson(finalResult),
		Images:      p.extractImages(finalResult),
		Summary:     p.extractSummary(finalResult),
		Duration:    result.Duration,
		CompletedAt: result.CompletedAt,
	}
	orchestrator.UpdateSession(session)

	// Track session completion for BrainPrint
	if orchestrator.brainprintSvc != nil {
		// Get user ID from session metadata or use session ID as user ID
		userID := sessionID // Default to session ID
		if userIDFromMeta, ok := session.Metadata["user_id"].(string); ok && userIDFromMeta != "" {
			userID = userIDFromMeta
		}

		// Get explanation type from session metadata
		explanationType := "standard"
		if expType, ok := session.Metadata["explanation_type"].(string); ok && expType != "" {
			explanationType = expType
		}

		// Track successful session
		if err := orchestrator.brainprintSvc.TrackSession(ctx, userID, explanationType, true); err != nil {
			p.logger.WithFields(logrus.Fields{
				"session_id": sessionID,
				"user_id":    userID,
				"error":      err,
			}).Warn("Failed to track session for BrainPrint")
		}
	}

	// Broadcast final success event
	orchestrator.BroadcastEvent(sessionID, SSEEvent{
		Type:      "pipeline-completed",
		SessionID: sessionID,
		Data: map[string]interface{}{
			"result":   finalResult,
			"duration": result.Duration.Milliseconds(),
		},
		Timestamp: time.Now(),
	})

	p.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"duration":   result.Duration,
	}).Info("Pipeline completed successfully")

	return nil
}

// executeStep executes a single pipeline step
func (p *Pipeline) executeStep(ctx context.Context, sessionID string, step PipelineStep, orchestrator *Orchestrator, stepIndex int) PipelineStepResult {
	stepResult := PipelineStepResult{
		StepName:   step.Name,
		Status:     "running",
		Output:     make(map[string]string),
		Metadata:   make(map[string]interface{}),
		RetryCount: 0,
	}

	startTime := time.Now()
	defer func() {
		stepResult.Duration = time.Since(startTime)
	}()

	// Broadcast step start
	orchestrator.BroadcastEvent(sessionID, SSEEvent{
		Type:      "step-start",
		SessionID: sessionID,
		StepID:    fmt.Sprintf("step-%d", stepIndex+1),
		Data: map[string]interface{}{
			"step_name":  step.Name,
			"step_index": stepIndex,
		},
		Timestamp: time.Now(),
	})

	// Get context if required
	var contextDocs []ContextDoc
	if step.RequiresContext {
		var err error
		contextDocs, err = p.getContext(ctx, sessionID, step.Inputs["topic"])
		if err != nil {
			p.logger.WithFields(logrus.Fields{
				"session_id": sessionID,
				"step":       step.Name,
				"error":      err,
			}).Error("Failed to get context")
			// Continue without context rather than failing
		}
	}

	// Prepare inputs with context
	inputs := make(map[string]string)
	for k, v := range step.Inputs {
		inputs[k] = v
	}

	// Add context to inputs
	if len(contextDocs) > 0 {
		contextText := p.formatContext(contextDocs)
		inputs["context"] = contextText
	}

	// Execute step with retry logic
	_, exists := p.adkClients[step.Agent]
	if !exists {
		stepResult.Status = "failed"
		stepResult.Error = fmt.Sprintf("agent %s not found", step.Agent)
		return stepResult
	}

	// Create task request
	taskReq := adk.TaskRequest{
		SessionID: sessionID,
		Step:      step.Name,
		Topic:     step.Inputs["topic"],
		Inputs:    inputs,
	}

	// Execute with retries
	var lastErr error
	for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
		if attempt > 0 {
			stepResult.RetryCount++
			p.logger.WithFields(logrus.Fields{
				"session_id": sessionID,
				"step":       step.Name,
				"attempt":    attempt + 1,
			}).Warn("Retrying step execution")

			// Broadcast retry event
			orchestrator.BroadcastEvent(sessionID, SSEEvent{
				Type:      "step-retry",
				SessionID: sessionID,
				StepID:    fmt.Sprintf("step-%d", stepIndex+1),
				Data: map[string]interface{}{
					"step_name":   step.Name,
					"attempt":     attempt + 1,
					"max_retries": p.config.MaxRetries,
				},
				Timestamp: time.Now(),
			})

			// Wait before retry
			select {
			case <-ctx.Done():
				stepResult.Status = "cancelled"
				stepResult.Error = "context cancelled"
				return stepResult
			case <-time.After(p.config.RetryDelay * time.Duration(attempt)):
			}
		}

		// Try to get ID token for the target agent (optional - for GCP service-to-service auth)
		token, err := p.authClient.GetIDToken(ctx, p.config.AgentBaseURLs[step.Agent])
		var client *adk.Client
		
		if err != nil {
			// Authentication failed (likely not running on GCP) - continue without auth
			p.logger.WithFields(logrus.Fields{
				"session_id": sessionID,
				"step":       step.Name,
				"agent":      step.Agent,
				"error":      err,
			}).Warn("Failed to get ID token for agent, continuing without authentication")
			
			// Use the existing client without authentication
			client = p.adkClients[step.Agent]
		} else {
			// Create authenticated client for this request
			client = adk.NewClient(p.config.AgentBaseURLs[step.Agent],
				adk.WithTimeout(p.config.StepTimeout),
				adk.WithConfig(adk.TaskConfig{
					Timeout:     p.config.StepTimeout,
					MaxRetries:  p.config.MaxRetries,
					RetryDelay:  p.config.RetryDelay,
					BackoffType: "exponential",
				}),
				adk.WithLogger(p.logger),
				adk.WithAuthToken(token),
			)
		}

		// Execute the task - construct full URL using baseURL from client
		// DoTask expects a full URL, so we use the client's baseURL
		fullURL := client.GetBaseURL() + "/task"
		response, err := client.DoTask(ctx, fullURL, taskReq)
		if err == nil {
			// Success
			stepResult.Status = "completed"
			stepResult.Output = response.Artifacts
			stepResult.Metadata["metrics"] = response.Metrics
			stepResult.Metadata["delta"] = response.Delta
			stepResult.Metadata["next"] = response.Next
			return stepResult
		}

		lastErr = err

		// Check if error is retryable
		if taskErr, ok := err.(*adk.TaskError); ok && !taskErr.IsRetryable() {
			// Non-retryable error
			stepResult.Status = "failed"
			stepResult.Error = err.Error()
			return stepResult
		}

		// Broadcast delta updates during execution
		if attempt == 0 {
			orchestrator.BroadcastEvent(sessionID, SSEEvent{
				Type:      "step-delta",
				SessionID: sessionID,
				StepID:    fmt.Sprintf("step-%d", stepIndex+1),
				Data: map[string]interface{}{
					"delta": fmt.Sprintf("Processing %s...", step.Name),
				},
				Timestamp: time.Now(),
			})
		}
	}

	// All retries exhausted
	stepResult.Status = "failed"
	stepResult.Error = fmt.Sprintf("step failed after %d attempts: %v", p.config.MaxRetries+1, lastErr)
	return stepResult
}

// getContext retrieves relevant context using hybrid search
func (p *Pipeline) getContext(ctx context.Context, sessionID, topic string) ([]ContextDoc, error) {
	// Check if Elasticsearch is available
	if p.elasticRetriever == nil {
		p.logger.WithFields(logrus.Fields{
			"session_id": sessionID,
			"topic":      topic,
		}).Info("Elasticsearch not available, skipping context retrieval")
		return []ContextDoc{}, nil
	}

	p.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"topic":      topic,
		"index":      p.config.ElasticIndex,
	}).Info("Retrieving context using hybrid search")

	// Perform hybrid search
	results, err := p.elasticRetriever.HybridSearch(ctx, p.config.ElasticIndex, topic, p.config.ContextTopK)
	if err != nil {
		return nil, fmt.Errorf("hybrid search failed: %w", err)
	}

	// Convert to ContextDoc
	// Note: The retriever returns []SearchResult from the retriever package
	// which has Doc, Score, and Snippet fields
	contextDocs := make([]ContextDoc, 0, len(results))
	for i := range results {
		// Access the retriever's SearchResult fields directly
		// The retriever's SearchResult has Doc, Score, and Snippet fields
		contextDocs = append(contextDocs, ContextDoc{
			Doc:     results[i].Doc,
			Score:   results[i].Score,
			Snippet: results[i].Snippet,
		})
	}

	p.logger.WithFields(logrus.Fields{
		"session_id":    sessionID,
		"results_count": len(contextDocs),
	}).Info("Context retrieved successfully")

	return contextDocs, nil
}

// enrichStepInputs merges outputs from previous steps into the current step's inputs
// This allows steps like visualizer and critic to receive data from previous steps
func (p *Pipeline) enrichStepInputs(step PipelineStep, previousOutputs map[string]map[string]string) PipelineStep {
	// Create a copy of the step to avoid modifying the original
	enrichedStep := step
	
	// Create a copy of inputs
	enrichedInputs := make(map[string]string)
	for k, v := range step.Inputs {
		enrichedInputs[k] = v
	}
	
	// Visualizer needs lesson from explainer
	if step.Name == "visualizer" {
		if explainerOutput, exists := previousOutputs["explainer"]; exists {
			if lesson, ok := explainerOutput["lesson"]; ok && lesson != "" {
				enrichedInputs["lesson"] = lesson
				p.logger.WithFields(logrus.Fields{
					"step":        step.Name,
					"lesson_size": len(lesson),
				}).Info("Added lesson from explainer to visualizer inputs")
			} else {
				p.logger.WithFields(logrus.Fields{
					"step": step.Name,
				}).Warn("Explainer output exists but lesson key is missing or empty")
			}
		} else {
			p.logger.WithFields(logrus.Fields{
				"step":              step.Name,
				"available_outputs": getMapKeys(previousOutputs),
			}).Warn("Explainer output not found in previous outputs")
		}
	}
	
	// Critic needs lesson from explainer
	if step.Name == "critic" {
		if explainerOutput, exists := previousOutputs["explainer"]; exists {
			if lesson, ok := explainerOutput["lesson"]; ok && lesson != "" {
				enrichedInputs["lesson"] = lesson
				p.logger.WithFields(logrus.Fields{
					"step":        step.Name,
					"lesson_size": len(lesson),
				}).Info("Added lesson from explainer to critic inputs")
			} else {
				p.logger.WithFields(logrus.Fields{
					"step": step.Name,
				}).Warn("Explainer output exists but lesson key is missing or empty")
			}
		} else {
			p.logger.WithFields(logrus.Fields{
				"step":              step.Name,
				"available_outputs": getMapKeys(previousOutputs),
			}).Warn("Explainer output not found in previous outputs")
		}
	}
	
	// Explainer can optionally use outline and misconceptions from summarizer
	if step.Name == "explainer" {
		if summarizerOutput, exists := previousOutputs["summarizer"]; exists {
			if outline, ok := summarizerOutput["outline"]; ok && outline != "" {
				enrichedInputs["outline"] = outline
			}
			if misconceptions, ok := summarizerOutput["misconceptions"]; ok && misconceptions != "" {
				enrichedInputs["misconceptions"] = misconceptions
			}
		}
	}
	
	enrichedStep.Inputs = enrichedInputs
	return enrichedStep
}

// getMapKeys returns the keys of a map as a slice of strings
func getMapKeys(m map[string]map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// formatContext formats context documents for agent consumption
func (p *Pipeline) formatContext(docs []ContextDoc) string {
	if len(docs) == 0 {
		return ""
	}

	var contextParts []string
	for i, doc := range docs {
		contextPart := fmt.Sprintf("Document %d (Score: %.3f):\nTopic: %s\nSection: %s\nContent: %s\n",
			i+1, doc.Score, doc.Doc.Topic, doc.Doc.Section, doc.Snippet)
		contextParts = append(contextParts, contextPart)
	}

	return strings.Join(contextParts, "\n---\n\n")
}

// applyCriticPatch applies critic feedback to the lesson JSON
// lessonJSON: The lesson JSON from the explainer step
// criticOutput: The critic's output containing critique and patch_plan
func (p *Pipeline) applyCriticPatch(ctx context.Context, sessionID string, lessonJSON string, criticOutput map[string]string, orchestrator *Orchestrator) error {
	p.logger.WithField("session_id", sessionID).Info("Applying critic patch")

	// Validate lesson JSON is provided
	if lessonJSON == "" {
		return fmt.Errorf("lesson JSON is required to apply critic patch")
	}

	// Parse the lesson JSON
	var lesson map[string]interface{}
	if err := json.Unmarshal([]byte(lessonJSON), &lesson); err != nil {
		return fmt.Errorf("failed to parse lesson JSON: %w", err)
	}

	// Apply patch plan if available
	if patchPlanJSON, exists := criticOutput["patch_plan"]; exists && patchPlanJSON != "" {
		// Use the ApplyPatchPlan function to apply the patch plan
		patchedLesson, err := ApplyPatchPlan(lessonJSON, patchPlanJSON)
		if err != nil {
			p.logger.WithFields(logrus.Fields{
				"session_id": sessionID,
				"error":      err,
			}).Warn("Failed to apply patch plan, using original lesson")
		} else {
			// Use the patched lesson
			lessonJSON = patchedLesson
			// Re-parse the patched lesson
			if err := json.Unmarshal([]byte(lessonJSON), &lesson); err != nil {
				return fmt.Errorf("failed to parse patched lesson JSON: %w", err)
			}
		}
	}
	
	// Add critic metadata
	lesson["critic_reviewed"] = true
	if critiqueJSON, exists := criticOutput["critique"]; exists {
		lesson["critique"] = critiqueJSON
	}

	// Update the lesson in the session
	session, exists := orchestrator.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	// Update session result with patched lesson
	if session.Result == nil {
		session.Result = &SessionResult{}
	}

	// Convert lesson back to JSON string
	updatedLessonJSON, err := json.Marshal(lesson)
	if err != nil {
		return fmt.Errorf("failed to marshal updated lesson: %w", err)
	}

	session.Result.Lesson = string(updatedLessonJSON)
	orchestrator.UpdateSession(session)

	p.logger.WithField("session_id", sessionID).Info("Critic patch applied successfully")
	return nil
}

// extractLesson extracts lesson content from final result
func (p *Pipeline) extractLesson(finalResult map[string]interface{}) string {
	if explainer, exists := finalResult["explainer"]; exists {
		if explainerMap, ok := explainer.(map[string]string); ok {
			if lesson, exists := explainerMap["lesson"]; exists {
				return lesson
			}
		}
	}
	return "No lesson content available"
}

// extractImages extracts image references from final result
func (p *Pipeline) extractImages(finalResult map[string]interface{}) map[string]string {
	images := make(map[string]string)

	if visualizer, exists := finalResult["visualizer"]; exists {
		if visualizerMap, ok := visualizer.(map[string]string); ok {
			for k, v := range visualizerMap {
				if strings.HasPrefix(k, "image_") || strings.Contains(k, "image") {
					images[k] = v
				}
			}
		}
	}

	return images
}

// extractSummary extracts summary content from final result
func (p *Pipeline) extractSummary(finalResult map[string]interface{}) string {
	if summarizer, exists := finalResult["summarizer"]; exists {
		if summarizerMap, ok := summarizer.(map[string]string); ok {
			if summary, exists := summarizerMap["summary"]; exists {
				return summary
			}
		}
	}
	return "No summary available"
}

// GetConfig returns the current pipeline configuration
func (p *Pipeline) GetConfig() PipelineConfig {
	return p.config
}

// SetConfig updates the pipeline configuration
func (p *Pipeline) SetConfig(config PipelineConfig) {
	p.config = config
}

// Health checks the health of all pipeline components
func (p *Pipeline) Health(ctx context.Context) error {
	// Check Elasticsearch health (if available)
	if p.elasticClient != nil {
		if err := p.elasticClient.Health(ctx); err != nil {
			return fmt.Errorf("elasticsearch health check failed: %w", err)
		}
	}

	// Check ADK clients health
	for agentName, client := range p.adkClients {
		if err := client.Health(ctx); err != nil {
			return fmt.Errorf("agent %s health check failed: %w", agentName, err)
		}
	}

	return nil
}

// ApplyPatchPlan applies a patch plan to a lesson JSON deterministically
func ApplyPatchPlan(lessonJSON string, patchPlanJSON string) (string, error) {
	// Parse the original lesson
	var lesson llm.OGLesson
	if err := json.Unmarshal([]byte(lessonJSON), &lesson); err != nil {
		return "", fmt.Errorf("failed to parse lesson JSON: %w", err)
	}

	// Parse the patch plan
	var patchPlan []llm.PatchPlanItem
	if err := json.Unmarshal([]byte(patchPlanJSON), &patchPlan); err != nil {
		return "", fmt.Errorf("failed to parse patch plan JSON: %w", err)
	}

	// Apply each patch in order
	for _, patch := range patchPlan {
		switch patch.Section {
		case "big_picture":
			lesson.BigPicture = patch.ReplacementText
		case "metaphor":
			lesson.Metaphor = patch.ReplacementText
		case "core_mechanism":
			lesson.CoreMechanism = patch.ReplacementText
		case "toy_example_code":
			lesson.ToyExampleCode = patch.ReplacementText
		case "memory_hook":
			lesson.MemoryHook = patch.ReplacementText
		case "real_life":
			lesson.RealLife = patch.ReplacementText
		case "best_practices":
			lesson.BestPractices = patch.ReplacementText
		default:
			return "", fmt.Errorf("unknown section: %s", patch.Section)
		}
	}

	// Marshal the patched lesson back to JSON
	patchedJSON, err := json.Marshal(lesson)
	if err != nil {
		return "", fmt.Errorf("failed to marshal patched lesson: %w", err)
	}

	return string(patchedJSON), nil
}
