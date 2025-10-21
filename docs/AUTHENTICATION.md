# Authentication System

## Overview

The ExplainIQ Agent system implements Google-signed JWT authentication for secure service-to-service communication. All agent services verify JWT tokens from the orchestrator, while the orchestrator obtains ID tokens from the Google Cloud metadata server for each agent call.

## Architecture

### Components

1. **Authentication Client** (`internal/auth/client.go`)
   - Google JWT verification
   - ID token retrieval from metadata server
   - Public key fetching from Google

2. **Authentication Middleware** (`internal/auth/middleware.go`)
   - JWT token validation
   - Service authentication
   - Public endpoint handling

3. **ADK Client Integration** (`internal/adk/client.go`)
   - Authenticated HTTP requests
   - Token injection in headers

## Authentication Flow

### Service-to-Service Authentication

```
Orchestrator → Agent Services
     ↓
1. Orchestrator calls metadata server
2. Gets ID token for target agent
3. Sends request with Bearer token
4. Agent validates JWT token
5. Processes request if valid
```

### JWT Token Validation

1. **Extract Token**: From `Authorization: Bearer <token>` header
2. **Parse Header**: Get `kid` (key ID) from JWT header
3. **Fetch Public Key**: Download Google's public keys
4. **Verify Signature**: Validate JWT signature with public key
5. **Validate Claims**: Check issuer, audience, expiration
6. **Extract User Info**: Get user ID, email, etc.

## Configuration

### Environment Variables

```bash
# Service URL (used as audience in JWT validation)
SERVICE_URL=http://localhost:8080

# Google Cloud metadata server (for ID token retrieval)
METADATA_URL=http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity

# Google's public key endpoint
GOOGLE_CERT_URL=https://www.googleapis.com/oauth2/v3/certs
```

### Service URLs

- **Orchestrator**: `http://localhost:8080`
- **Agent Summarizer**: `http://localhost:8081`
- **Agent Explainer**: `http://localhost:8082`
- **Agent Critic**: `http://localhost:8083`
- **Agent Visualizer**: `http://localhost:8084`
- **Frontend**: `http://localhost:3000`

## Implementation Details

### JWT Claims Validation

```go
// Required claims
{
  "iss": "https://accounts.google.com",  // Issuer
  "aud": "http://localhost:8081",       // Audience (service URL)
  "sub": "user123",                     // Subject (user ID)
  "email": "user@example.com",          // User email
  "iat": 1234567890,                    // Issued at
  "exp": 1234567890                     // Expiration
}
```

### Public Endpoints

The following endpoints skip authentication:

- `/healthz`
- `/health`
- `/metrics`
- `/api/v1/health`
- `/api/v1/status`

### Protected Endpoints

All other endpoints require authentication:

- `/api/sessions/{id}/result` (orchestrator)
- `/task` (all agents)

## Usage Examples

### Orchestrator Making Authenticated Requests

```go
// Get ID token for target agent
token, err := authClient.GetIDToken(ctx, "http://localhost:8081")
if err != nil {
    return fmt.Errorf("failed to get ID token: %w", err)
}

// Create authenticated ADK client
client := adk.NewClient("http://localhost:8081",
    adk.WithAuthToken(token),
)

// Make authenticated request
response, err := client.DoTask(ctx, "/task", taskReq)
```

### Agent Service with Authentication

```go
// Create auth client
authClient := auth.NewClient("http://localhost:8081")

// Setup routes with authentication
router.POST("/task", auth.ServiceAuthMiddleware(authClient), handler)
```

### Frontend (No Authentication Required)

The frontend communicates directly with the orchestrator without authentication for user-initiated requests.

## Security Considerations

### Token Validation

- **Issuer Verification**: Must be `https://accounts.google.com`
- **Audience Verification**: Must match service URL
- **Expiration Check**: Tokens expire after 1 hour
- **Signature Verification**: Uses Google's public keys

### Public Key Caching

- Google's public keys are fetched on-demand
- Keys are cached for performance
- Key rotation is handled automatically

### Error Handling

- Invalid tokens return 401 Unauthorized
- Missing tokens return 401 Unauthorized
- Network errors are logged and handled gracefully

## Testing

### Unit Tests

```bash
# Run auth middleware tests
cd internal/auth
go test -v

# Run specific test
go test -v -run TestAuthMiddleware
```

### Integration Tests

```bash
# Test with real Google Cloud environment
export SERVICE_URL=http://localhost:8080
go test -v -tags=integration
```

## Deployment

### Google Cloud Run

1. **Service Account**: Ensure proper service account permissions
2. **Metadata Server**: Available in Cloud Run environment
3. **Network**: Services can communicate via internal URLs

### Local Development

1. **Mock Tokens**: Use development tokens for testing
2. **Service Discovery**: Use localhost URLs
3. **Certificate Validation**: May need to disable in development

## Troubleshooting

### Common Issues

1. **Token Validation Fails**
   - Check service URL configuration
   - Verify Google Cloud credentials
   - Check network connectivity

2. **Metadata Server Unavailable**
   - Ensure running in Google Cloud environment
   - Check service account permissions
   - Verify metadata server URL

3. **Public Key Fetch Fails**
   - Check internet connectivity
   - Verify Google's certificate URL
   - Check firewall settings

### Debug Mode

Enable debug logging:

```bash
export LOG_LEVEL=debug
```

### Health Checks

Test authentication:

```bash
# Test orchestrator health
curl http://localhost:8080/health

# Test agent health (no auth required)
curl http://localhost:8081/healthz

# Test protected endpoint (should fail without auth)
curl http://localhost:8081/task
```

## Future Enhancements

### Planned Features

1. **Token Refresh**: Automatic token renewal
2. **Rate Limiting**: Per-user request limits
3. **Audit Logging**: Authentication event tracking
4. **Multi-tenant**: Support for multiple organizations

### Security Improvements

1. **Certificate Pinning**: Pin Google's public keys
2. **Token Binding**: Bind tokens to specific connections
3. **Encryption**: Encrypt sensitive data in transit
4. **Monitoring**: Real-time security monitoring



