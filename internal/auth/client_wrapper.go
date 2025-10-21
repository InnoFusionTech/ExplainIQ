package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// AuthenticatedHTTPClient wraps an HTTP client with authentication
type AuthenticatedHTTPClient struct {
	authClient *Client
	httpClient *http.Client
	baseURL    string
}

// NewAuthenticatedHTTPClient creates a new authenticated HTTP client
func NewAuthenticatedHTTPClient(authClient *Client, baseURL string) *AuthenticatedHTTPClient {
	return &AuthenticatedHTTPClient{
		authClient: authClient,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
	}
}

// Do performs an authenticated HTTP request
func (c *AuthenticatedHTTPClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Get ID token for the target service
	token, err := c.authClient.GetIDToken(ctx, c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get ID token: %w", err)
	}

	// Add Authorization header
	req.Header.Set("Authorization", "Bearer "+token)

	// Perform the request
	return c.httpClient.Do(req)
}

// Post performs an authenticated POST request
func (c *AuthenticatedHTTPClient) Post(ctx context.Context, path string, body interface{}) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	return c.Do(ctx, req)
}

// Get performs an authenticated GET request
func (c *AuthenticatedHTTPClient) Get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	return c.Do(ctx, req)
}



