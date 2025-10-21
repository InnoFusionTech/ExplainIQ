# Quota System Documentation

## Overview

The ExplainIQ Agent system implements a comprehensive quota management system that combines rate limiting and cost tracking to prevent abuse and manage resource usage. The system tracks both per-IP request rates and per-session costs for LLM tokens and image generation.

## Architecture

### Components

1. **Rate Limiter** (`internal/rate_limiter/`)
   - Per-IP rate limiting using `golang.org/x/time/rate`
   - Configurable requests per second and burst limits
   - Automatic cleanup of old limiters

2. **Cost Tracker** (`internal/cost_tracker/`)
   - Session-based cost tracking for LLM calls and image generation
   - Firestore persistence for cost data
   - Real-time cost estimation and limit checking

3. **Quota Manager** (`internal/quota/`)
   - Combines rate limiting and cost tracking
   - Middleware for HTTP request handling
   - Quota information for SSE metadata

## Rate Limiting

### Configuration

```go
// Create rate limiter (10 requests per second, burst of 20)
rateLimiter := rate_limiter.NewLimiter(10.0, 20)
```

### Features

- **Per-IP Limiting**: Each IP address gets its own rate limiter
- **Token Bucket Algorithm**: Uses `golang.org/x/time/rate` for efficient rate limiting
- **Burst Handling**: Allows short bursts up to the burst limit
- **Automatic Cleanup**: Removes old limiters to prevent memory leaks

### Response

When rate limit is exceeded:

```json
{
  "error": "Rate limit exceeded",
  "message": "Too many requests from your IP. Please try again later.",
  "retry_after": 60,
  "quota_type": "rate_limit"
}
```

## Cost Tracking

### Cost Estimation

#### LLM Costs
- **Gemini Pro**: $0.50 per 1M input tokens, $1.50 per 1M output tokens
- **Gemini Pro Vision**: $0.50 per 1M input tokens, $1.50 per 1M output tokens
- **Default**: $1.00 per 1M input tokens, $2.00 per 1M output tokens

#### Image Generation Costs
- **Imagen**: $0.02 per image generated

### Default Limits

```go
CostLimits{
    MaxLLMCost:   10.0,  // $10 max for LLM calls
    MaxImageCost: 5.0,   // $5 max for image generation
    MaxTotalCost: 15.0,  // $15 max total cost
    MaxLLMCalls:  100,   // 100 LLM calls max
    MaxImageCalls: 50,   // 50 image calls max
}
```

### Cost Tracking Flow

1. **LLM Call Tracking**:
   ```go
   // Estimate tokens and track cost
   inputTokens := len(topic) + len(context)
   outputTokens := len(result.Outline) + len(result.Prerequisites)
   
   costTracker.TrackLLMCall(ctx, sessionID, userID, ipAddress, "gemini-pro", inputTokens, outputTokens)
   ```

2. **Image Call Tracking**:
   ```go
   // Track image generation cost
   costTracker.TrackImageCall(ctx, sessionID, userID, ipAddress, imageCount)
   ```

3. **Cost Persistence**:
   - Individual cost entries stored in Firestore
   - Session totals updated in real-time
   - Cost limits checked before each operation

## Quota Middleware

### Usage

```go
// Create quota manager
quotaManager := quota.NewQuotaManager(rateLimiter, costTracker)

// Apply middleware
router.Use(quotaManager.QuotaMiddleware())
```

### Middleware Behavior

1. **Rate Limit Check**: Verifies IP-based rate limits
2. **Cost Limit Check**: Verifies session-based cost limits
3. **Quota Information**: Adds quota data to request context
4. **Error Responses**: Returns 429 with detailed error information

### Response Format

When cost limit is exceeded:

```json
{
  "error": "Cost limit exceeded",
  "message": "You have exceeded your usage quota. Please try again later or contact support.",
  "quota_type": "cost_limit",
  "quota_remaining": {
    "llm_cost_remaining": 5.0,
    "image_cost_remaining": 2.0,
    "total_cost_remaining": 7.0,
    "llm_calls_remaining": 50,
    "image_calls_remaining": 25
  },
  "current_costs": {
    "total_cost": 8.0,
    "llm_cost": 6.0,
    "image_cost": 2.0,
    "llm_calls": 50,
    "image_calls": 25
  }
}
```

## SSE Integration

### Quota Metadata

Quota information is included in Server-Sent Events:

```json
{
  "type": "step-complete",
  "session_id": "session-123",
  "step_id": "step-1",
  "data": {
    "step_name": "summarizer",
    "status": "completed",
    "duration": 1500,
    "quota_info": {
      "session_id": "session-123",
      "current_costs": {
        "total_cost": 0.05,
        "llm_cost": 0.03,
        "image_cost": 0.02
      },
      "quota_remaining": {
        "total_cost_remaining": 14.95,
        "llm_cost_remaining": 9.97,
        "image_cost_remaining": 4.98
      }
    }
  },
  "timestamp": "2023-01-01T00:00:00Z"
}
```

## Configuration

### Environment Variables

```bash
# Google Cloud Project ID for Firestore
GCP_PROJECT_ID=your-project-id

# Service URL for cost tracking
SERVICE_URL=http://localhost:8080
```

### Custom Limits

```go
// Set custom cost limits
customLimits := cost_tracker.CostLimits{
    MaxLLMCost:   5.0,   // $5 max for LLM calls
    MaxImageCost: 2.0,   // $2 max for image generation
    MaxTotalCost: 7.0,   // $7 max total cost
    MaxLLMCalls:  50,    // 50 LLM calls max
    MaxImageCalls: 25,   // 25 image calls max
}
quotaManager.SetCostLimits(customLimits)
```

## Implementation Details

### Storage Schema

#### Cost Entries
```
cost_entry:{session_id}:{timestamp}
{
  "session_id": "session-123",
  "user_id": "user-456",
  "ip_address": "127.0.0.1",
  "timestamp": "2023-01-01T00:00:00Z",
  "operation": "llm_call",
  "model": "gemini-pro",
  "input_tokens": 100,
  "output_tokens": 50,
  "estimated_cost": 0.000125,
  "metadata": {
    "input_tokens": 100,
    "output_tokens": 50
  }
}
```

#### Session Costs
```
session_costs:{session_id}
{
  "session_id": "session-123",
  "user_id": "user-456",
  "ip_address": "127.0.0.1",
  "total_llm_cost": 0.05,
  "total_image_cost": 0.02,
  "total_cost": 0.07,
  "llm_calls": 5,
  "image_calls": 1,
  "last_updated": "2023-01-01T00:00:00Z",
  "created_at": "2023-01-01T00:00:00Z"
}
```

### Error Handling

1. **Storage Failures**: Cost tracking continues without persistence
2. **Rate Limit Failures**: Requests are rejected with 429 status
3. **Cost Limit Failures**: Requests are rejected with detailed quota info
4. **Network Issues**: Graceful degradation with logging

## Testing

### Unit Tests

```bash
# Run quota system tests
cd internal/quota
go test -v

# Run rate limiter tests
cd internal/rate_limiter
go test -v

# Run cost tracker tests
cd internal/cost_tracker
go test -v
```

### Integration Tests

```bash
# Test with real Firestore
export GCP_PROJECT_ID=your-project-id
go test -v -tags=integration
```

## Monitoring

### Metrics

1. **Rate Limit Metrics**:
   - Requests per IP per second
   - Rate limit violations
   - Cleanup operations

2. **Cost Metrics**:
   - Total costs per session
   - LLM vs image costs
   - Cost limit violations

3. **Quota Metrics**:
   - Quota utilization
   - Remaining quota
   - Limit violations

### Logging

```go
// Rate limit violations
logger.WithFields(logrus.Fields{
    "ip":   "127.0.0.1",
    "path": "/api/sessions",
}).Warn("Rate limit exceeded")

// Cost limit violations
logger.WithFields(logrus.Fields{
    "session_id":     "session-123",
    "total_cost":     8.0,
    "max_cost":       7.0,
    "llm_cost":       6.0,
    "image_cost":     2.0,
}).Warn("Cost limit exceeded")
```

## Deployment

### Google Cloud Run

1. **Firestore**: Enable Firestore API
2. **Service Account**: Ensure proper permissions
3. **Environment Variables**: Set GCP_PROJECT_ID
4. **Resource Limits**: Configure memory and CPU limits

### Local Development

1. **Firestore Emulator**: Use local emulator for development
2. **Mock Storage**: Use mock storage for testing
3. **Rate Limits**: Use higher limits for development

## Security Considerations

### Rate Limiting

- **IP Spoofing**: Use X-Forwarded-For headers carefully
- **Distributed Attacks**: Consider per-user limits
- **Burst Protection**: Configure appropriate burst limits

### Cost Tracking

- **Token Estimation**: Use accurate token counting in production
- **Cost Validation**: Validate cost estimates against actual usage
- **Limit Enforcement**: Ensure limits are enforced consistently

### Data Privacy

- **IP Addresses**: Consider anonymization for privacy
- **User Data**: Minimize user data collection
- **Retention**: Implement data retention policies

## Future Enhancements

### Planned Features

1. **Dynamic Limits**: Adjust limits based on user tier
2. **Cost Optimization**: Suggest cost-effective alternatives
3. **Usage Analytics**: Detailed usage reports and insights
4. **Alert System**: Notifications when approaching limits

### Performance Improvements

1. **Caching**: Cache quota information for better performance
2. **Batch Operations**: Batch cost tracking operations
3. **Async Processing**: Process cost tracking asynchronously
4. **Database Optimization**: Optimize Firestore queries

## Troubleshooting

### Common Issues

1. **Rate Limit Too Strict**:
   - Increase requests per second limit
   - Increase burst limit
   - Check for IP address issues

2. **Cost Limits Too Low**:
   - Increase cost limits
   - Check token estimation accuracy
   - Verify cost calculation logic

3. **Storage Issues**:
   - Check Firestore permissions
   - Verify GCP_PROJECT_ID
   - Check network connectivity

### Debug Mode

Enable debug logging:

```bash
export LOG_LEVEL=debug
```

### Health Checks

Test quota system:

```bash
# Test rate limiting
for i in {1..15}; do curl http://localhost:8080/api/sessions; done

# Test cost tracking
curl -X POST http://localhost:8080/api/sessions \
  -H "Content-Type: application/json" \
  -d '{"topic": "test topic"}'
```



