package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// GeminiClientInterface defines the interface for Gemini client
type GeminiClientInterface interface {
	Summarize(ctx context.Context, topic, context string) (*SummarizeResponse, error)
	ExplainWithOG(ctx context.Context, topic, outline, misconceptions, context string) (*OGLesson, error)
	CritiqueLesson(ctx context.Context, lessonJSON string) (*CritiqueResponse, error)
	VisualizeCore(ctx context.Context, lessonJSON, sessionID string) (*VisualizeResponse, error)
	Health(ctx context.Context) error
	SetAPIKey(apiKey string)
	SetModel(model string)
	SetBaseURL(baseURL string)
	GetModelInfo() map[string]interface{}
}

// GeminiClient represents a client for Google Gemini API
type GeminiClient struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	logger     *logrus.Logger
}

// GeminiRequest represents a request to the Gemini API
type GeminiRequest struct {
	Contents         []GeminiContent        `json:"contents"`
	GenerationConfig GeminiGenerationConfig `json:"generationConfig,omitempty"`
}

// GeminiContent represents content in a Gemini request
type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

// GeminiPart represents a part of content
type GeminiPart struct {
	Text string `json:"text"`
}

// GeminiGenerationConfig represents generation configuration
type GeminiGenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
	TopK            int     `json:"topK,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

// GeminiResponse represents a response from the Gemini API
type GeminiResponse struct {
	Candidates    []GeminiCandidate   `json:"candidates"`
	UsageMetadata GeminiUsageMetadata `json:"usageMetadata,omitempty"`
}

// GeminiCandidate represents a candidate response
type GeminiCandidate struct {
	Content      GeminiContent `json:"content"`
	FinishReason string        `json:"finishReason,omitempty"`
}

// GeminiUsageMetadata represents usage metadata
type GeminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// GeminiError represents an error from the Gemini API
type GeminiError struct {
	Error GeminiErrorDetail `json:"error"`
}

// GeminiErrorDetail represents error details
type GeminiErrorDetail struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// SummarizeRequest represents a request to summarize content
type SummarizeRequest struct {
	Topic   string `json:"topic"`
	Context string `json:"context"`
}

// SummarizeResponse represents the response from summarization
type SummarizeResponse struct {
	Outline        []string `json:"outline"`
	Prerequisites  []string `json:"prerequisites"`
	Misconceptions []string `json:"misconceptions"`
	Citations      []string `json:"citations"`
}

// OGLesson represents an OpenGraph-style lesson with structured content
type OGLesson struct {
	BigPicture     string `json:"big_picture"`      // High-level overview and context
	Metaphor       string `json:"metaphor"`         // Analogical explanation to aid understanding
	CoreMechanism  string `json:"core_mechanism"`   // The fundamental how/why it works
	ToyExampleCode string `json:"toy_example_code"` // Simple, runnable code example
	MemoryHook     string `json:"memory_hook"`      // Mnemonic device or memorable phrase
	RealLife       string `json:"real_life"`        // Real-world applications and examples
	BestPractices  string `json:"best_practices"`   // Key do's and don'ts
}

// CritiqueIssue represents an issue found in a lesson
type CritiqueIssue struct {
	Section  string `json:"section"`  // The section of the lesson (e.g., "big_picture", "metaphor")
	Problem  string `json:"problem"`  // Description of the problem
	Severity string `json:"severity"` // Severity level: "low", "medium", "high", "critical"
}

// PatchPlanItem represents a specific change to be made to a lesson
type PatchPlanItem struct {
	Section         string `json:"section"`          // The section to modify
	Change          string `json:"change"`           // Description of the change
	ReplacementText string `json:"replacement_text"` // The new text to replace the problematic content
}

// CritiqueResponse represents the response from lesson critique
type CritiqueResponse struct {
	Issues    []CritiqueIssue `json:"issues"`     // List of issues found
	PatchPlan []PatchPlanItem `json:"patch_plan"` // List of changes to fix the issues
}

// ImageRef represents a reference to a generated image
type ImageRef struct {
	URL     string `json:"url"`      // Signed URL to the image
	AltText string `json:"alt_text"` // Alt text for accessibility
	Caption string `json:"caption"`  // Caption describing the image
}

// VisualizeResponse represents the response from visualization
type VisualizeResponse struct {
	Images   []ImageRef `json:"images"`   // List of generated images
	Captions []string   `json:"captions"` // List of captions for the images
}

// NewGeminiClient creates a new Gemini client
func NewGeminiClient(apiKey string) *GeminiClient {
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}

	return &GeminiClient{
		apiKey:  apiKey,
		baseURL: "https://generativelanguage.googleapis.com/v1beta",
		model:   "gemini-1.5-flash",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger: logrus.New(),
	}
}

// Summarize generates a summary with outline, prerequisites, and misconceptions
func (c *GeminiClient) Summarize(ctx context.Context, topic, context string) (*SummarizeResponse, error) {
	c.logger.WithFields(logrus.Fields{
		"topic":       topic,
		"context_len": len(context),
		"model":       c.model,
	}).Info("Starting summarization")

	// Create the prompt
	prompt := c.createSummarizePrompt(topic, context)

	// Create the request
	request := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{
						Text: prompt,
					},
				},
			},
		},
		GenerationConfig: GeminiGenerationConfig{
			Temperature:     0.3, // Lower temperature for more consistent output
			TopP:            0.8,
			TopK:            40,
			MaxOutputTokens: 2048,
		},
	}

	// Execute the request
	response, err := c.executeRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to execute summarization request: %w", err)
	}

	// Parse the response
	result, err := c.parseSummarizeResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse summarization response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"topic":          topic,
		"outline_count":  len(result.Outline),
		"prereq_count":   len(result.Prerequisites),
		"miscon_count":   len(result.Misconceptions),
		"citation_count": len(result.Citations),
	}).Info("Summarization completed successfully")

	return result, nil
}

// createSummarizePrompt creates the prompt for summarization
func (c *GeminiClient) createSummarizePrompt(topic, context string) string {
	return fmt.Sprintf(`You are an expert educational content summarizer. Analyze the provided context about "%s" and create a comprehensive summary.

Context:
%s

Please provide a JSON response with the following structure:
{
  "outline": ["bullet point 1", "bullet point 2", ...],
  "prerequisites": ["prerequisite 1", "prerequisite 2", ...],
  "misconceptions": ["misconception 1", "misconception 2", ...],
  "citations": ["doc1", "doc2", ...]
}

Requirements:
- Use concise bullet points (no markdown formatting)
- Extract document IDs from the context and include them in citations array
- Identify key learning objectives for the outline
- List essential knowledge needed before learning this topic
- Highlight common misconceptions people have about this topic
- Return ONLY valid JSON, no additional text or explanations
- Keep each bullet point under 100 characters
- Maximum 10 items per array

Topic: %s`, topic, context, topic)
}

// executeRequest executes a request to the Gemini API
func (c *GeminiClient) executeRequest(ctx context.Context, request GeminiRequest) (*GeminiResponse, error) {
	// Marshal request
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.baseURL, c.model, c.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var apiError GeminiError
		if err := json.Unmarshal(responseBody, &apiError); err == nil {
			return nil, fmt.Errorf("API error %d: %s", apiError.Error.Code, apiError.Error.Message)
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Parse response
	var response GeminiResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Validate response
	if len(response.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	return &response, nil
}

// parseSummarizeResponse parses the summarization response
func (c *GeminiClient) parseSummarizeResponse(response *GeminiResponse) (*SummarizeResponse, error) {
	// Extract text from the first candidate
	if len(response.Candidates) == 0 || len(response.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	text := response.Candidates[0].Content.Parts[0].Text

	// Clean up the text (remove any markdown formatting or extra text)
	text = strings.TrimSpace(text)

	// Try to extract JSON from the response
	jsonStart := strings.Index(text, "{")
	jsonEnd := strings.LastIndex(text, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonStart >= jsonEnd {
		return nil, fmt.Errorf("no valid JSON found in response: %s", text)
	}

	jsonText := text[jsonStart : jsonEnd+1]

	// Parse JSON
	var result SummarizeResponse
	if err := json.Unmarshal([]byte(jsonText), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w, text: %s", err, jsonText)
	}

	// Validate and clean the response
	result = c.validateAndCleanResponse(result)

	return &result, nil
}

// validateAndCleanResponse validates and cleans the response
func (c *GeminiClient) validateAndCleanResponse(result SummarizeResponse) SummarizeResponse {
	// Clean and validate outline
	if result.Outline == nil {
		result.Outline = []string{}
	}
	result.Outline = c.cleanStringArray(result.Outline, 10)

	// Clean and validate prerequisites
	if result.Prerequisites == nil {
		result.Prerequisites = []string{}
	}
	result.Prerequisites = c.cleanStringArray(result.Prerequisites, 10)

	// Clean and validate misconceptions
	if result.Misconceptions == nil {
		result.Misconceptions = []string{}
	}
	result.Misconceptions = c.cleanStringArray(result.Misconceptions, 10)

	// Clean and validate citations
	if result.Citations == nil {
		result.Citations = []string{}
	}
	result.Citations = c.cleanStringArray(result.Citations, 20)

	return result
}

// cleanStringArray cleans and limits a string array
func (c *GeminiClient) cleanStringArray(arr []string, maxLen int) []string {
	if len(arr) > maxLen {
		arr = arr[:maxLen]
	}

	cleaned := make([]string, 0, len(arr))
	for _, item := range arr {
		item = strings.TrimSpace(item)
		if item != "" && len(item) <= 200 { // Limit individual item length
			cleaned = append(cleaned, item)
		}
	}

	return cleaned
}

// Health checks the health of the Gemini client
func (c *GeminiClient) Health(ctx context.Context) error {
	// Create a simple test request
	testRequest := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{
						Text: "Hello, respond with 'OK'",
					},
				},
			},
		},
		GenerationConfig: GeminiGenerationConfig{
			Temperature:     0.1,
			MaxOutputTokens: 10,
		},
	}

	_, err := c.executeRequest(ctx, testRequest)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// SetAPIKey updates the API key
func (c *GeminiClient) SetAPIKey(apiKey string) {
	c.apiKey = apiKey
}

// SetModel updates the model
func (c *GeminiClient) SetModel(model string) {
	c.model = model
}

// SetBaseURL updates the base URL
func (c *GeminiClient) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

// ExplainWithOG generates an OpenGraph-style lesson using the Gemini model
func (c *GeminiClient) ExplainWithOG(ctx context.Context, topic, outline, misconceptions, context string) (*OGLesson, error) {
	c.logger.WithFields(logrus.Fields{
		"topic":          topic,
		"outline":        len(outline),
		"misconceptions": len(misconceptions),
		"context":        len(context),
		"model":          c.model,
	}).Info("Generating OG lesson with Gemini")

	// Construct the prompt
	prompt := c.buildExplainOGPrompt(topic, outline, misconceptions, context)

	// Create request
	request := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{
						Text: prompt,
					},
				},
			},
		},
		GenerationConfig: GeminiGenerationConfig{
			Temperature:     0.7,
			TopK:            40,
			TopP:            0.95,
			MaxOutputTokens: 2048,
		},
	}

	// Make API call
	response, err := c.executeRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	// Parse response
	if len(response.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	candidate := response.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return nil, fmt.Errorf("no content parts in response")
	}

	// Extract JSON from response
	responseText := candidate.Content.Parts[0].Text
	ogLesson, err := c.parseOGLessonResponse(responseText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OG lesson response: %w", err)
	}

	c.logger.WithField("topic", topic).Info("OG lesson generation completed")
	return ogLesson, nil
}

// buildExplainOGPrompt constructs the prompt for OG lesson generation
func (c *GeminiClient) buildExplainOGPrompt(topic, outline, misconceptions, context string) string {
	var promptBuilder strings.Builder

	promptBuilder.WriteString(fmt.Sprintf(`
You are an expert educator creating an OpenGraph-style lesson for the topic: %s

Your task is to produce a JSON object with exactly these fields:
- "big_picture": High-level overview and context (2-3 sentences)
- "metaphor": Analogical explanation to aid understanding (1-2 sentences)
- "core_mechanism": The fundamental how/why it works (2-3 sentences)
- "toy_example_code": Simple, runnable code example (if applicable, otherwise "N/A")
- "memory_hook": Mnemonic device or memorable phrase (1 sentence)
- "real_life": Real-world applications and examples (2-3 sentences)
- "best_practices": Key do's and don'ts (2-3 bullet points)

`, topic))

	if outline != "" {
		promptBuilder.WriteString("Learning Outline:\n")
		promptBuilder.WriteString(outline)
		promptBuilder.WriteString("\n\n")
	}

	if misconceptions != "" {
		promptBuilder.WriteString("Common Misconceptions to Address:\n")
		promptBuilder.WriteString(misconceptions)
		promptBuilder.WriteString("\n\n")
	}

	if context != "" {
		promptBuilder.WriteString("Additional Context:\n")
		promptBuilder.WriteString(context)
		promptBuilder.WriteString("\n\n")
	}

	promptBuilder.WriteString(`
Requirements:
- Produce JSON only, no markdown formatting
- Keep each section short and crisp
- Align content with the provided outline
- Address the specified misconceptions
- Make explanations accessible and memorable

Example JSON structure:
{
  "big_picture": "Brief overview...",
  "metaphor": "Analogy...",
  "core_mechanism": "How it works...",
  "toy_example_code": "Simple code or N/A",
  "memory_hook": "Memorable phrase...",
  "real_life": "Applications...",
  "best_practices": "Do this, avoid that..."
}

Your JSON response:
`)

	return promptBuilder.String()
}

// parseOGLessonResponse extracts and parses the OGLesson from the response text
func (c *GeminiClient) parseOGLessonResponse(responseText string) (*OGLesson, error) {
	// Find JSON in the response
	jsonStart := strings.Index(responseText, "{")
	if jsonStart == -1 {
		return nil, fmt.Errorf("no JSON found in response")
	}

	jsonEnd := strings.LastIndex(responseText, "}")
	if jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("invalid JSON structure in response")
	}

	jsonStr := responseText[jsonStart : jsonEnd+1]

	var ogLesson OGLesson
	if err := json.Unmarshal([]byte(jsonStr), &ogLesson); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OG lesson JSON: %w", err)
	}

	// Clean and validate the response
	ogLesson.BigPicture = strings.TrimSpace(ogLesson.BigPicture)
	ogLesson.Metaphor = strings.TrimSpace(ogLesson.Metaphor)
	ogLesson.CoreMechanism = strings.TrimSpace(ogLesson.CoreMechanism)
	ogLesson.ToyExampleCode = strings.TrimSpace(ogLesson.ToyExampleCode)
	ogLesson.MemoryHook = strings.TrimSpace(ogLesson.MemoryHook)
	ogLesson.RealLife = strings.TrimSpace(ogLesson.RealLife)
	ogLesson.BestPractices = strings.TrimSpace(ogLesson.BestPractices)

	return &ogLesson, nil
}

// CritiqueLesson analyzes a lesson and provides critique with patch plan
func (c *GeminiClient) CritiqueLesson(ctx context.Context, lessonJSON string) (*CritiqueResponse, error) {
	c.logger.WithFields(logrus.Fields{
		"lesson_length": len(lessonJSON),
		"model":         c.model,
	}).Info("Critiquing lesson with Gemini")

	// Construct the prompt
	prompt := c.buildCritiquePrompt(lessonJSON)

	// Create request
	request := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{
						Text: prompt,
					},
				},
			},
		},
		GenerationConfig: GeminiGenerationConfig{
			Temperature:     0.3, // Lower temperature for more consistent critique
			TopK:            40,
			TopP:            0.95,
			MaxOutputTokens: 2048,
		},
	}

	// Make API call
	response, err := c.executeRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	// Parse response
	if len(response.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	candidate := response.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return nil, fmt.Errorf("no content parts in response")
	}

	// Extract JSON from response
	responseText := candidate.Content.Parts[0].Text
	critiqueResponse, err := c.parseCritiqueResponse(responseText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse critique response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"issues_count":     len(critiqueResponse.Issues),
		"patch_plan_count": len(critiqueResponse.PatchPlan),
	}).Info("Lesson critique completed")

	return critiqueResponse, nil
}

// buildCritiquePrompt constructs the prompt for lesson critique
func (c *GeminiClient) buildCritiquePrompt(lessonJSON string) string {
	var promptBuilder strings.Builder

	promptBuilder.WriteString(`
You are an expert educational content reviewer. Your task is to critique a lesson and provide specific, actionable feedback.

Analyze the following lesson JSON and identify issues, then create a patch plan to fix them.

Lesson to critique:
`)

	promptBuilder.WriteString(lessonJSON)
	promptBuilder.WriteString(`

Your response must be a JSON object with exactly these fields:
- "issues": Array of issues found, each with:
  - "section": The section name (e.g., "big_picture", "metaphor", "core_mechanism", "toy_example_code", "memory_hook", "real_life", "best_practices")
  - "problem": Specific description of the problem
  - "severity": Severity level ("low", "medium", "high", "critical")

- "patch_plan": Array of fixes, each with:
  - "section": The section to modify
  - "change": Description of what needs to change
  - "replacement_text": The complete new text for that section

Evaluation Criteria:
1. **Clarity**: Is the content clear and understandable?
2. **Accuracy**: Is the information technically correct?
3. **Completeness**: Are all necessary concepts covered?
4. **Engagement**: Is the content engaging and memorable?
5. **Structure**: Does each section serve its intended purpose?
6. **Code Quality**: If code is present, is it correct and runnable?
7. **Length**: Are sections appropriately sized (not too short/long)?

Severity Guidelines:
- "critical": Factual errors, broken code, major misconceptions
- "high": Significant clarity issues, missing key concepts
- "medium": Minor clarity issues, could be more engaging
- "low": Minor improvements, style suggestions

Example JSON structure:
{
  "issues": [
    {
      "section": "big_picture",
      "problem": "Too vague and doesn't provide clear context",
      "severity": "high"
    }
  ],
  "patch_plan": [
    {
      "section": "big_picture",
      "change": "Provide more specific context and clear definition",
      "replacement_text": "Machine learning is a subset of artificial intelligence that enables computers to learn patterns from data without being explicitly programmed for each task."
    }
  ]
}

Your JSON response:
`)

	return promptBuilder.String()
}

// parseCritiqueResponse extracts and parses the CritiqueResponse from the response text
func (c *GeminiClient) parseCritiqueResponse(responseText string) (*CritiqueResponse, error) {
	// Find JSON in the response
	jsonStart := strings.Index(responseText, "{")
	if jsonStart == -1 {
		return nil, fmt.Errorf("no JSON found in response")
	}

	jsonEnd := strings.LastIndex(responseText, "}")
	if jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("invalid JSON structure in response")
	}

	jsonStr := responseText[jsonStart : jsonEnd+1]

	var critiqueResponse CritiqueResponse
	if err := json.Unmarshal([]byte(jsonStr), &critiqueResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal critique response JSON: %w", err)
	}

	// Clean and validate the response
	for i := range critiqueResponse.Issues {
		critiqueResponse.Issues[i].Section = strings.TrimSpace(critiqueResponse.Issues[i].Section)
		critiqueResponse.Issues[i].Problem = strings.TrimSpace(critiqueResponse.Issues[i].Problem)
		critiqueResponse.Issues[i].Severity = strings.TrimSpace(critiqueResponse.Issues[i].Severity)
	}

	for i := range critiqueResponse.PatchPlan {
		critiqueResponse.PatchPlan[i].Section = strings.TrimSpace(critiqueResponse.PatchPlan[i].Section)
		critiqueResponse.PatchPlan[i].Change = strings.TrimSpace(critiqueResponse.PatchPlan[i].Change)
		critiqueResponse.PatchPlan[i].ReplacementText = strings.TrimSpace(critiqueResponse.PatchPlan[i].ReplacementText)
	}

	return &critiqueResponse, nil
}

// VisualizeCore generates visual diagrams for lesson core mechanisms using Imagen
func (c *GeminiClient) VisualizeCore(ctx context.Context, lessonJSON, sessionID string) (*VisualizeResponse, error) {
	c.logger.WithFields(logrus.Fields{
		"lesson_length": len(lessonJSON),
		"session_id":    sessionID,
		"model":         c.model,
	}).Info("Generating visualizations with Imagen")

	// Parse the lesson to extract core mechanism
	var lesson OGLesson
	if err := json.Unmarshal([]byte(lessonJSON), &lesson); err != nil {
		return nil, fmt.Errorf("failed to parse lesson JSON: %w", err)
	}

	// Generate visualization prompts for the core mechanism
	prompts, err := c.buildVisualizationPrompts(lesson.CoreMechanism)
	if err != nil {
		return nil, fmt.Errorf("failed to build visualization prompts: %w", err)
	}

	// Generate images using Imagen (mock implementation for now)
	var images []ImageRef
	var captions []string

	for i, prompt := range prompts {
		// In a real implementation, this would call the Imagen API
		// For now, we'll create mock image references
		imageRef := ImageRef{
			URL:     fmt.Sprintf("https://storage.googleapis.com/%s/sessions/%s/diagram_%d.png", os.Getenv("GCS_BUCKET"), sessionID, i+1),
			AltText: fmt.Sprintf("Diagram %d illustrating %s", i+1, lesson.CoreMechanism),
			Caption: prompt.Caption,
		}
		images = append(images, imageRef)
		captions = append(captions, prompt.Caption)
	}

	response := &VisualizeResponse{
		Images:   images,
		Captions: captions,
	}

	c.logger.WithFields(logrus.Fields{
		"session_id":     sessionID,
		"images_count":   len(images),
		"captions_count": len(captions),
	}).Info("Visualization generation completed")

	return response, nil
}

// VisualizationPrompt represents a prompt for generating a diagram
type VisualizationPrompt struct {
	Prompt  string `json:"prompt"`  // The prompt for Imagen
	Caption string `json:"caption"` // The caption for the generated image
}

// buildVisualizationPrompts creates prompts for visualizing the core mechanism
func (c *GeminiClient) buildVisualizationPrompts(coreMechanism string) ([]VisualizationPrompt, error) {
	if coreMechanism == "" {
		return nil, fmt.Errorf("core mechanism is empty")
	}

	// Create prompts for different aspects of the core mechanism
	prompts := []VisualizationPrompt{
		{
			Prompt:  fmt.Sprintf("Create a minimal, clean diagram showing: %s. Use simple shapes, arrows, and labels. No text blocks, just visual elements.", coreMechanism),
			Caption: fmt.Sprintf("Core mechanism diagram: %s", coreMechanism),
		},
		{
			Prompt:  fmt.Sprintf("Create a flowchart diagram illustrating the process: %s. Use rectangles for steps, diamonds for decisions, and arrows for flow.", coreMechanism),
			Caption: fmt.Sprintf("Process flowchart: %s", coreMechanism),
		},
	}

	return prompts, nil
}

// GetModelInfo returns information about the current model
func (c *GeminiClient) GetModelInfo() map[string]interface{} {
	return map[string]interface{}{
		"model":       c.model,
		"base_url":    c.baseURL,
		"api_key_set": c.apiKey != "",
	}
}
