# Elasticsearch Client

A comprehensive Elasticsearch client for the ExplainIQ platform with hybrid search capabilities, supporting both BM25 text search and dense vector similarity search with MMR (Maximal Marginal Relevance) reranking.

## Features

- **Official Elasticsearch Go Client**: Uses `github.com/elastic/go-elasticsearch/v8`
- **Hybrid Search**: Combines BM25 text search with dense vector similarity
- **MMR Reranking**: Implements Maximal Marginal Relevance for result diversification
- **Bulk Operations**: Efficient document indexing using Elasticsearch bulk API
- **Index Management**: Create, delete, and check index existence
- **Comprehensive Mapping**: Pre-configured mapping for hybrid search scenarios

## Installation

The client uses the official Elasticsearch Go client:

```bash
go get github.com/elastic/go-elasticsearch/v8
```

## Quick Start

```go
import "github.com/creduntvitam/explainiq/internal/elastic"

// Create client
client, err := elastic.NewClient(ctx, "https://your-cluster.com:9200", "your-api-key")
if err != nil {
    log.Fatal(err)
}

// Create index with hybrid search mapping
err = client.CreateIndex(ctx, "my-index", nil)

// Index documents
docs := []elastic.Doc{
    {
        ID:        "doc-1",
        Topic:     "Machine Learning",
        Section:   "Introduction", 
        Text:      "Machine learning is...",
        Embedding: []float32{0.1, 0.2, ...}, // 1536 dimensions
        Metadata:  map[string]string{"author": "John"},
    },
}

err = client.UpsertDocs(ctx, "my-index", docs)

// Perform hybrid search
results, err := client.HybridSearch(ctx, "my-index", "machine learning", embedding, 10)
```

## Document Structure

The `Doc` struct represents documents for indexing:

```go
type Doc struct {
    ID         string            `json:"id"`          // Unique document identifier
    Topic      string            `json:"topic"`       // Document topic/category
    Section    string            `json:"section"`     // Document section
    Text       string            `json:"text"`        // Main text content
    Embedding  []float32         `json:"embedding"`   // Dense vector (1536 dims)
    Metadata   map[string]string `json:"metadata"`    // Additional metadata
    CreatedAt  string            `json:"created_at"`  // Creation timestamp
    UpdatedAt  string            `json:"updated_at"`  // Update timestamp
}
```

## Hybrid Search Mapping

The default mapping supports both BM25 and vector search:

### Text Fields (BM25)
- **topic**: Analyzed text with keyword subfield
- **section**: Analyzed text with keyword subfield  
- **text**: Main content with custom analyzer

### Vector Field
- **embedding**: Dense vector field (1536 dimensions)
- **Similarity**: Cosine similarity
- **Indexing**: Enabled for fast vector search

### Analyzer Configuration
- **Tokenizer**: Standard tokenizer
- **Filters**: lowercase, stop words, snowball stemming
- **Custom**: `hybrid_analyzer` for consistent text processing

## Hybrid Search Query

The `HybridSearch` method combines:

1. **BM25 Text Search**: Multi-match query across text fields
2. **Vector Similarity**: kNN search using dense vectors
3. **MMR Reranking**: Script-based reranking for diversity

### Query Structure
```json
{
  "query": {
    "bool": {
      "should": [
        {
          "multi_match": {
            "query": "search text",
            "fields": ["text^2", "topic^1.5", "section"],
            "type": "best_fields",
            "fuzziness": "AUTO"
          }
        },
        {
          "knn": {
            "field": "embedding",
            "query_vector": [0.1, 0.2, ...],
            "k": 20,
            "num_candidates": 40
          }
        }
      ]
    }
  },
  "rescore": [
    {
      "window_size": 20,
      "query": {
        "rescore_query": {
          "script_score": {
            "script": {
              "source": "/* MMR reranking script */"
            }
          }
        }
      }
    }
  ]
}
```

## MMR Reranking

Maximal Marginal Relevance (MMR) balances relevance and diversity:

- **Lambda Parameter**: Controls relevance vs diversity balance (0.7 default)
- **Diversity Calculation**: Based on topic/section similarity
- **Reranking Window**: Processes top candidates for optimal results

## API Reference

### NewClient
```go
func NewClient(ctx context.Context, baseURL, apiKey string) (*Client, error)
```
Creates a new Elasticsearch client with connection validation.

### CreateIndex
```go
func (c *Client) CreateIndex(ctx context.Context, name string, mapping interface{}) error
```
Creates an index with hybrid search mapping. Pass `nil` for default mapping.

### UpsertDocs
```go
func (c *Client) UpsertDocs(ctx context.Context, index string, docs []Doc) error
```
Upserts documents using Elasticsearch bulk API with refresh.

### HybridSearch
```go
func (c *Client) HybridSearch(ctx context.Context, index, query string, embedding []float32, size int) (*SearchResult, error)
```
Performs hybrid search combining BM25 and vector similarity with MMR reranking.

### Index Management
```go
func (c *Client) DeleteIndex(ctx context.Context, name string) error
func (c *Client) IndexExists(ctx context.Context, name string) (bool, error)
func (c *Client) Health(ctx context.Context) error
```

## Configuration

### Environment Variables
- `EXPLAINIQ_ELASTIC_URL`: Elasticsearch cluster URL
- `EXPLAINIQ_ELASTIC_API_KEY`: API key for authentication

### Index Settings
- **Shards**: 1 (configurable)
- **Replicas**: 0 (configurable)
- **Refresh**: Immediate for bulk operations
- **Analysis**: Custom analyzer for text processing

## Performance Considerations

### Bulk Operations
- Use `UpsertDocs` for batch indexing
- Refresh is enabled for immediate searchability
- Consider batch size for optimal performance

### Search Performance
- kNN search uses `num_candidates` for efficiency
- MMR reranking processes limited window size
- Vector dimensions affect memory usage

### Memory Usage
- Dense vectors (1536 dims) require significant memory
- Consider vector quantization for large datasets
- Monitor cluster memory usage

## Error Handling

The client provides detailed error messages for:
- Connection failures
- Index creation errors
- Bulk operation failures
- Search query errors
- Response parsing errors

## Testing

Run tests with:
```bash
go test ./internal/elastic
go test -v ./internal/elastic  # Verbose output
```

Test coverage includes:
- Client creation and validation
- Document structure validation
- Mapping generation
- Search result parsing
- Error handling scenarios

## Examples

See `example_usage.go` for comprehensive examples:
- Basic client usage
- Custom mapping configuration
- Bulk operations
- Index management
- Hybrid search implementation

## Security

- Uses API key authentication
- Supports TLS/SSL connections
- No sensitive data in logs
- Follows Elasticsearch security best practices

## Monitoring

The client includes structured logging for:
- Index operations
- Bulk operations
- Search queries
- Error conditions
- Performance metrics

Use with your preferred logging framework for centralized monitoring.

