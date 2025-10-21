package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/sirupsen/logrus"
)

// Client represents an Elasticsearch client with hybrid search capabilities
type Client struct {
	es     *elasticsearch.Client
	logger *logrus.Logger
}

// Doc represents a document for indexing with hybrid search capabilities
type Doc struct {
	ID        string            `json:"id"`
	Topic     string            `json:"topic"`
	Section   string            `json:"section"`
	Text      string            `json:"text"`
	Embedding []float32         `json:"embedding"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt string            `json:"created_at,omitempty"`
	UpdatedAt string            `json:"updated_at,omitempty"`
}

// NewClient creates a new Elasticsearch client using the official Go client
func NewClient(ctx context.Context, baseURL, apiKey string) (*Client, error) {
	logger := logrus.New()

	// Configure Elasticsearch client
	cfg := elasticsearch.Config{
		Addresses: []string{baseURL},
		APIKey:    apiKey,
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	client := &Client{
		es:     es,
		logger: logger,
	}

	// Test connection
	if err := client.Health(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to Elasticsearch: %w", err)
	}

	return client, nil
}

// CreateIndex creates an index with hybrid search mapping (BM25 + dense vectors)
func (c *Client) CreateIndex(ctx context.Context, name string, mapping interface{}) error {
	c.logger.WithField("index", name).Info("Creating index with hybrid search mapping")

	// Default hybrid search mapping if none provided
	if mapping == nil {
		mapping = c.getHybridSearchMapping()
	}

	// Convert mapping to JSON
	mappingJSON, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("failed to marshal mapping: %w", err)
	}

	// Create index request
	req := esapi.IndicesCreateRequest{
		Index: name,
		Body:  bytes.NewReader(mappingJSON),
	}

	// Execute request
	res, err := req.Do(ctx, c.es)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("index creation failed: %s", string(body))
	}

	c.logger.WithField("index", name).Info("Index created successfully")
	return nil
}

// UpsertDocs upserts documents using the bulk API
func (c *Client) UpsertDocs(ctx context.Context, index string, docs []Doc) error {
	if len(docs) == 0 {
		return nil
	}

	c.logger.WithFields(logrus.Fields{
		"index": index,
		"count": len(docs),
	}).Info("Upserting documents")

	var buf bytes.Buffer
	for _, doc := range docs {
		// Index action
		action := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": index,
				"_id":    doc.ID,
			},
		}

		// Serialize action
		actionJSON, err := json.Marshal(action)
		if err != nil {
			return fmt.Errorf("failed to marshal action: %w", err)
		}
		buf.Write(actionJSON)
		buf.WriteString("\n")

		// Serialize document
		docJSON, err := json.Marshal(doc)
		if err != nil {
			return fmt.Errorf("failed to marshal document: %w", err)
		}
		buf.Write(docJSON)
		buf.WriteString("\n")
	}

	// Bulk request
	req := esapi.BulkRequest{
		Body:    &buf,
		Refresh: "true",
	}

	// Execute request
	res, err := req.Do(ctx, c.es)
	if err != nil {
		return fmt.Errorf("failed to execute bulk request: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("bulk request failed: %s", string(body))
	}

	c.logger.WithField("count", len(docs)).Info("Documents upserted successfully")
	return nil
}

// getHybridSearchMapping returns the mapping for hybrid search (BM25 + dense vectors)
func (c *Client) getHybridSearchMapping() map[string]interface{} {
	return map[string]interface{}{
		"settings": map[string]interface{}{
			"number_of_shards":   1,
			"number_of_replicas": 0,
			"analysis": map[string]interface{}{
				"analyzer": map[string]interface{}{
					"hybrid_analyzer": map[string]interface{}{
						"type":      "custom",
						"tokenizer": "standard",
						"filter":    []string{"lowercase", "stop", "snowball"},
					},
				},
			},
		},
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type": "keyword",
				},
				"topic": map[string]interface{}{
					"type":     "text",
					"analyzer": "hybrid_analyzer",
					"fields": map[string]interface{}{
						"keyword": map[string]interface{}{
							"type": "keyword",
						},
					},
				},
				"section": map[string]interface{}{
					"type":     "text",
					"analyzer": "hybrid_analyzer",
					"fields": map[string]interface{}{
						"keyword": map[string]interface{}{
							"type": "keyword",
						},
					},
				},
				"text": map[string]interface{}{
					"type":     "text",
					"analyzer": "hybrid_analyzer",
					"fields": map[string]interface{}{
						"keyword": map[string]interface{}{
							"type": "keyword",
						},
					},
				},
				"embedding": map[string]interface{}{
					"type":       "dense_vector",
					"dims":       1536, // OpenAI ada-002 dimensions
					"index":      true,
					"similarity": "cosine",
				},
				"metadata": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"*": map[string]interface{}{
							"type": "keyword",
						},
					},
				},
				"created_at": map[string]interface{}{
					"type": "date",
				},
				"updated_at": map[string]interface{}{
					"type": "date",
				},
			},
		},
	}
}

// HybridSearch performs hybrid search combining BM25 and vector similarity
func (c *Client) HybridSearch(ctx context.Context, index, query string, embedding []float32, size int) (*SearchResult, error) {
	if size <= 0 {
		size = 10
	}

	searchBody := map[string]interface{}{
		"size": size,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []interface{}{
					// BM25 text search
					map[string]interface{}{
						"multi_match": map[string]interface{}{
							"query":     query,
							"fields":    []string{"text^2", "topic^1.5", "section"},
							"type":      "best_fields",
							"fuzziness": "AUTO",
						},
					},
					// Vector similarity search
					map[string]interface{}{
						"knn": map[string]interface{}{
							"field":          "embedding",
							"query_vector":   embedding,
							"k":              size * 2, // Get more candidates for reranking
							"num_candidates": size * 4,
						},
					},
				},
			},
		},
		// MMR reranking hook
		"rescore": []interface{}{
			map[string]interface{}{
				"window_size": size * 2,
				"query": map[string]interface{}{
					"rescore_query": map[string]interface{}{
						"script_score": map[string]interface{}{
							"query": map[string]interface{}{
								"match_all": map[string]interface{}{},
							},
							"script": map[string]interface{}{
								"source": `
									// MMR (Maximal Marginal Relevance) reranking
									def lambda = params.lambda;
									def diversity_bonus = 0.0;
									
									// Calculate diversity bonus based on topic/section similarity
									if (doc['topic.keyword'].size() > 0 && doc['section.keyword'].size() > 0) {
										// Simple diversity calculation - can be enhanced
										diversity_bonus = Math.random() * 0.1; // Placeholder for actual diversity calculation
									}
									
									// Combine relevance and diversity
									return _score * lambda + diversity_bonus * (1 - lambda);
								`,
								"params": map[string]interface{}{
									"lambda": 0.7, // Balance between relevance and diversity
								},
							},
						},
					},
					"query_weight":         0.7,
					"rescore_query_weight": 0.3,
				},
			},
		},
	}

	searchJSON, err := json.Marshal(searchBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search query: %w", err)
	}

	// Execute search
	req := esapi.SearchRequest{
		Index: []string{index},
		Body:  bytes.NewReader(searchJSON),
	}

	res, err := req.Do(ctx, c.es)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("search failed: %s", string(body))
	}

	// Parse response
	var searchResponse map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	return c.parseSearchResponse(searchResponse)
}

// SearchResult represents a search result
type SearchResult struct {
	Hits  []Hit `json:"hits"`
	Total int   `json:"total"`
}

// Hit represents a search hit
type Hit struct {
	ID     string                 `json:"id"`
	Score  float64                `json:"score"`
	Source map[string]interface{} `json:"source"`
}

// parseSearchResponse parses Elasticsearch search response
func (c *Client) parseSearchResponse(response map[string]interface{}) (*SearchResult, error) {
	hits, ok := response["hits"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid search response format")
	}

	total, ok := hits["total"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid total format in search response")
	}

	totalValue, ok := total["value"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid total value format")
	}

	hitsArray, ok := hits["hits"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid hits array format")
	}

	var resultHits []Hit
	for _, hit := range hitsArray {
		hitMap, ok := hit.(map[string]interface{})
		if !ok {
			continue
		}

		id, _ := hitMap["_id"].(string)
		score, _ := hitMap["_score"].(float64)
		source, _ := hitMap["_source"].(map[string]interface{})

		resultHits = append(resultHits, Hit{
			ID:     id,
			Score:  score,
			Source: source,
		})
	}

	return &SearchResult{
		Hits:  resultHits,
		Total: int(totalValue),
	}, nil
}

// Health checks the health of the Elasticsearch client
func (c *Client) Health(ctx context.Context) error {
	req := esapi.ClusterHealthRequest{}
	res, err := req.Do(ctx, c.es)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("health check failed: %s", string(body))
	}

	c.logger.Debug("Elasticsearch health check passed")
	return nil
}

// DeleteIndex deletes an index
func (c *Client) DeleteIndex(ctx context.Context, name string) error {
	c.logger.WithField("index", name).Info("Deleting index")

	req := esapi.IndicesDeleteRequest{
		Index: []string{name},
	}

	res, err := req.Do(ctx, c.es)
	if err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("index deletion failed: %s", string(body))
	}

	c.logger.WithField("index", name).Info("Index deleted successfully")
	return nil
}

// IndexExists checks if an index exists
func (c *Client) IndexExists(ctx context.Context, name string) (bool, error) {
	req := esapi.IndicesExistsRequest{
		Index: []string{name},
	}

	res, err := req.Do(ctx, c.es)
	if err != nil {
		return false, fmt.Errorf("failed to check index existence: %w", err)
	}
	defer res.Body.Close()

	return res.StatusCode == 200, nil
}
