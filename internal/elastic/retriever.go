package elastic

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/InnoFusionTech/ExplainIQ/internal/llm"
	"github.com/sirupsen/logrus"
)

// Retriever represents a hybrid search retriever combining BM25 and vector search
type Retriever struct {
	client          *Client
	embeddingClient *llm.EmbeddingClient
	logger          *logrus.Logger
	bm25Weight      float64
	vectorWeight    float64
	mmrLambda       float64
}

// SearchHit represents a search result with combined scores
type SearchHit struct {
	Doc         Doc     `json:"doc"`
	Score       float64 `json:"score"`
	BM25Score   float64 `json:"bm25_score"`
	VectorScore float64 `json:"vector_score"`
	Snippet     string  `json:"snippet"`
}

// HybridSearchConfig represents configuration for hybrid search
type HybridSearchConfig struct {
	BM25Weight    float64 `json:"bm25_weight"`     // Weight for BM25 score (default: 0.3)
	VectorWeight  float64 `json:"vector_weight"`   // Weight for vector score (default: 0.7)
	MMRLambda     float64 `json:"mmr_lambda"`      // MMR diversification factor (default: 0.7)
	MaxSnippetLen int     `json:"max_snippet_len"` // Maximum snippet length (default: 200)
}

// NewRetriever creates a new hybrid search retriever
func NewRetriever(esClient *Client, embeddingClient *llm.EmbeddingClient) *Retriever {
	return &Retriever{
		client:          esClient,
		embeddingClient: embeddingClient,
		logger:          logrus.New(),
		bm25Weight:      0.3,
		vectorWeight:    0.7,
		mmrLambda:       0.7,
	}
}

// SetConfig sets the hybrid search configuration
func (r *Retriever) SetConfig(config HybridSearchConfig) {
	if config.BM25Weight > 0 {
		r.bm25Weight = config.BM25Weight
	}
	if config.VectorWeight > 0 {
		r.vectorWeight = config.VectorWeight
	}
	if config.MMRLambda >= 0 && config.MMRLambda <= 1 {
		r.mmrLambda = config.MMRLambda
	}
}

// HybridSearch performs hybrid search combining BM25 and vector similarity
func (r *Retriever) HybridSearch(ctx context.Context, index, query string, k int) ([]SearchHit, error) {
	if k <= 0 {
		k = 10 // Default to 10 results
	}

	r.logger.WithFields(logrus.Fields{
		"index": index,
		"query": query,
		"k":     k,
	}).Info("Starting hybrid search")

	// Step 1: Embed the query
	queryEmbedding, err := r.embedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Step 2: Execute Elasticsearch query with bool_should (BM25 + kNN)
	esResults, err := r.executeHybridQuery(ctx, index, query, queryEmbedding, k*2) // Get more for MMR
	if err != nil {
		return nil, fmt.Errorf("failed to execute hybrid query: %w", err)
	}

	// Step 3: Combine scores with weighted sum
	combinedResults := r.combineScores(esResults, query)

	// Step 4: Apply MMR diversification
	diversifiedResults := r.applyMMR(combinedResults, k)

	// Normalize scores and extract snippets
	finalResults := r.normalizeScoresAndExtractSnippets(diversifiedResults, query)

	r.logger.WithField("result_count", len(finalResults)).Info("Hybrid search completed")
	return finalResults, nil
}

// embedQuery embeds the query text using the embedding client
func (r *Retriever) embedQuery(ctx context.Context, query string) ([]float32, error) {
	r.logger.WithField("query", query).Debug("Embedding query")

	embeddings, err := r.embeddingClient.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embedding client failed: %w", err)
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return embeddings[0], nil
}

// executeHybridQuery executes the Elasticsearch query with bool_should
func (r *Retriever) executeHybridQuery(ctx context.Context, index, query string, embedding []float32, size int) ([]SearchHit, error) {
	// Execute the search using the existing client
	results, err := r.client.HybridSearch(ctx, index, query, embedding, size)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch query failed: %w", err)
	}

	// Convert to our internal format
	var searchResults []SearchHit
	for _, hit := range results.Hits {
		// Extract document from source
		doc := r.extractDocFromSource(hit.Source)

		// For now, we'll use the ES score as both BM25 and vector score
		// In a real implementation, you'd separate these scores
		searchResults = append(searchResults, SearchHit{
			Doc:         doc,
			Score:       hit.Score,
			BM25Score:   hit.Score * 0.5, // Placeholder - would need separate queries
			VectorScore: hit.Score * 0.5, // Placeholder - would need separate queries
		})
	}

	return searchResults, nil
}

// combineScores combines BM25 and vector scores with weighted sum
func (r *Retriever) combineScores(results []SearchHit, query string) []SearchHit {
	// Apply weighted combination of BM25 and vector scores
	for i := range results {
		// Combine scores using weighted sum
		results[i].Score = r.bm25Weight*results[i].BM25Score + r.vectorWeight*results[i].VectorScore
	}

	return results
}

// applyMMR applies Maximal Marginal Relevance diversification
func (r *Retriever) applyMMR(results []SearchHit, k int) []SearchHit {
	if len(results) <= k {
		return results
	}

	// MMR algorithm
	selected := make([]SearchHit, 0, k)
	remaining := make([]SearchHit, len(results))
	copy(remaining, results)

	// Select first result (highest score)
	if len(remaining) > 0 {
		selected = append(selected, remaining[0])
		remaining = remaining[1:]
	}

	// Select remaining results using MMR
	for len(selected) < k && len(remaining) > 0 {
		bestIdx := 0
		bestScore := -1.0

		for i, candidate := range remaining {
			// Calculate MMR score: λ * relevance - (1-λ) * max_similarity
			relevance := candidate.Score
			maxSimilarity := r.calculateMaxSimilarity(candidate, selected)
			mmrScore := r.mmrLambda*relevance - (1-r.mmrLambda)*maxSimilarity

			if mmrScore > bestScore {
				bestScore = mmrScore
				bestIdx = i
			}
		}

		// Add best candidate to selected
		selected = append(selected, remaining[bestIdx])
		// Remove from remaining
		remaining = append(remaining[:bestIdx], remaining[bestIdx+1:]...)
	}

	return selected
}

// calculateMaxSimilarity calculates maximum similarity between candidate and selected results
func (r *Retriever) calculateMaxSimilarity(candidate SearchHit, selected []SearchHit) float64 {
	if len(selected) == 0 {
		return 0.0
	}

	maxSim := 0.0
	for _, selected := range selected {
		// Calculate similarity based on topic and section overlap
		similarity := r.calculateTextSimilarity(candidate.Doc, selected.Doc)
		if similarity > maxSim {
			maxSim = similarity
		}
	}

	return maxSim
}

// calculateTextSimilarity calculates similarity between two documents
func (r *Retriever) calculateTextSimilarity(doc1, doc2 Doc) float64 {
	// Simple Jaccard similarity based on topic and section
	topic1 := strings.ToLower(doc1.Topic)
	topic2 := strings.ToLower(doc2.Topic)
	section1 := strings.ToLower(doc1.Section)
	section2 := strings.ToLower(doc2.Section)

	// Calculate Jaccard similarity for topic
	topicSim := r.jaccardSimilarity(topic1, topic2)
	sectionSim := r.jaccardSimilarity(section1, section2)

	// Weighted average
	return 0.7*topicSim + 0.3*sectionSim
}

// jaccardSimilarity calculates Jaccard similarity between two strings
func (r *Retriever) jaccardSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	// Simple word-based Jaccard similarity
	words1 := strings.Fields(s1)
	words2 := strings.Fields(s2)

	set1 := make(map[string]bool)
	for _, word := range words1 {
		set1[word] = true
	}

	set2 := make(map[string]bool)
	for _, word := range words2 {
		set2[word] = true
	}

	// Calculate intersection
	intersection := 0
	for word := range set1 {
		if set2[word] {
			intersection++
		}
	}

	// Calculate union
	union := len(set1) + len(set2) - intersection

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// normalizeScoresAndExtractSnippets normalizes scores and extracts snippets
func (r *Retriever) normalizeScoresAndExtractSnippets(results []SearchHit, query string) []SearchHit {
	if len(results) == 0 {
		return results
	}

	// Find min and max scores for normalization
	minScore := results[0].Score
	maxScore := results[0].Score
	for _, result := range results {
		if result.Score < minScore {
			minScore = result.Score
		}
		if result.Score > maxScore {
			maxScore = result.Score
		}
	}

	// Normalize scores to [0, 1] range
	scoreRange := maxScore - minScore
	if scoreRange == 0 {
		scoreRange = 1 // Avoid division by zero
	}

	for i := range results {
		// Normalize score
		results[i].Score = (results[i].Score - minScore) / scoreRange

		// Extract snippet
		results[i].Snippet = r.extractSnippet(results[i].Doc.Text, query, 200)
	}

	// Sort by normalized score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// extractSnippet extracts a relevant snippet from text
func (r *Retriever) extractSnippet(text, query string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}

	// Find the best position to start the snippet
	queryWords := strings.Fields(strings.ToLower(query))
	textLower := strings.ToLower(text)

	bestPos := 0
	maxMatches := 0

	// Slide a window through the text to find the best snippet position
	for i := 0; i <= len(text)-maxLen; i += 10 {
		window := textLower[i : i+maxLen]
		matches := 0

		for _, word := range queryWords {
			if strings.Contains(window, word) {
				matches++
			}
		}

		if matches > maxMatches {
			maxMatches = matches
			bestPos = i
		}
	}

	// Extract snippet with some context
	start := bestPos
	if start > 20 {
		start -= 20
	}

	end := start + maxLen
	if end > len(text) {
		end = len(text)
	}

	snippet := text[start:end]

	// Add ellipsis if needed
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(text) {
		snippet = snippet + "..."
	}

	return snippet
}

// extractDocFromSource extracts a Doc from Elasticsearch source
func (r *Retriever) extractDocFromSource(source map[string]interface{}) Doc {
	doc := Doc{}

	if id, ok := source["id"].(string); ok {
		doc.ID = id
	}
	if topic, ok := source["topic"].(string); ok {
		doc.Topic = topic
	}
	if section, ok := source["section"].(string); ok {
		doc.Section = section
	}
	if text, ok := source["text"].(string); ok {
		doc.Text = text
	}
	if createdAt, ok := source["created_at"].(string); ok {
		doc.CreatedAt = createdAt
	}
	if updatedAt, ok := source["updated_at"].(string); ok {
		doc.UpdatedAt = updatedAt
	}

	// Extract metadata
	if metadata, ok := source["metadata"].(map[string]interface{}); ok {
		doc.Metadata = make(map[string]string)
		for k, v := range metadata {
			if str, ok := v.(string); ok {
				doc.Metadata[k] = str
			}
		}
	}

	return doc
}

// GetConfig returns the current configuration
func (r *Retriever) GetConfig() HybridSearchConfig {
	return HybridSearchConfig{
		BM25Weight:    r.bm25Weight,
		VectorWeight:  r.vectorWeight,
		MMRLambda:     r.mmrLambda,
		MaxSnippetLen: 200,
	}
}

// SetWeights sets the BM25 and vector weights
func (r *Retriever) SetWeights(bm25Weight, vectorWeight float64) {
	if bm25Weight > 0 && vectorWeight > 0 {
		// Normalize weights
		total := bm25Weight + vectorWeight
		r.bm25Weight = bm25Weight / total
		r.vectorWeight = vectorWeight / total
	}
}

// SetMMRLambda sets the MMR diversification factor
func (r *Retriever) SetMMRLambda(lambda float64) {
	if lambda >= 0 && lambda <= 1 {
		r.mmrLambda = lambda
	}
}
