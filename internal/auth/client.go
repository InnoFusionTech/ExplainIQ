package auth

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

// Client represents an authentication client
type Client struct {
	JWTSecret     string
	Issuer        string
	Audience      string
	ServiceURL    string
	MetadataURL   string
	GoogleCertURL string
	logger        *logrus.Logger
	httpClient    *http.Client
}

// GoogleJWKS represents Google's JSON Web Key Set
type GoogleJWKS struct {
	Keys []GoogleJWK `json:"keys"`
}

// GoogleJWK represents a Google JSON Web Key
type GoogleJWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
	Alg string `json:"alg"`
}

// NewClient creates a new authentication client
func NewClient(serviceURL string) *Client {
	return &Client{
		ServiceURL:    serviceURL,
		Issuer:        "https://accounts.google.com",
		MetadataURL:   "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity",
		GoogleCertURL: "https://www.googleapis.com/oauth2/v3/certs",
		logger:        logrus.New(),
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}
}

// GetIDToken obtains an ID token from the metadata server for service-to-service authentication
func (c *Client) GetIDToken(ctx context.Context, targetAudience string) (string, error) {
	c.logger.WithField("target_audience", targetAudience).Info("Obtaining ID token from metadata server")

	req, err := http.NewRequestWithContext(ctx, "GET", c.MetadataURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create metadata request: %w", err)
	}

	req.Header.Set("Metadata-Flavor", "Google")
	q := req.URL.Query()
	q.Add("audience", targetAudience)
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get ID token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("metadata server returned status %d", resp.StatusCode)
	}

	// The metadata server returns the ID token directly as a string, not JSON
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	token := string(body)
	if token == "" {
		return "", fmt.Errorf("empty token received from metadata server")
	}

	return token, nil
}

// ValidateGoogleJWT validates a Google-signed JWT token
func (c *Client) ValidateGoogleJWT(ctx context.Context, tokenString string) (*Claims, error) {
	c.logger.Debug("Validating Google JWT token")

	// Parse the token without verification to get the header
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get the key ID from the header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing kid in token header")
		}

		// Fetch Google's public keys
		publicKey, err := c.getGooglePublicKey(ctx, kid)
		if err != nil {
			return nil, fmt.Errorf("failed to get public key: %w", err)
		}

		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Verify the token
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims format")
	}

	// Validate issuer
	iss, ok := claims["iss"].(string)
	if !ok || iss != c.Issuer {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %s", c.Issuer, iss)
	}

	// Validate audience
	aud, ok := claims["aud"].(string)
	if !ok || aud != c.ServiceURL {
		return nil, fmt.Errorf("invalid audience: expected %s, got %s", c.ServiceURL, aud)
	}

	// Extract user information
	userID, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)

	// Extract timestamps
	var issuedAt, expiresAt time.Time
	if iat, ok := claims["iat"].(float64); ok {
		issuedAt = time.Unix(int64(iat), 0)
	}
	if exp, ok := claims["exp"].(float64); ok {
		expiresAt = time.Unix(int64(exp), 0)
	}

	return &Claims{
		UserID:    userID,
		Email:     email,
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
		Extra:     claims,
	}, nil
}

// getGooglePublicKey fetches Google's public key for JWT verification
func (c *Client) getGooglePublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.GoogleCertURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cert request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Google certificates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch certificates: status %d", resp.StatusCode)
	}

	var jwks GoogleJWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	// Find the key with matching kid
	for _, key := range jwks.Keys {
		if key.Kid == kid {
			return c.parseRSAPublicKey(key.N, key.E)
		}
	}

	return nil, fmt.Errorf("public key not found for kid: %s", kid)
}

// parseRSAPublicKey parses RSA public key from modulus and exponent
func (c *Client) parseRSAPublicKey(n, e string) (*rsa.PublicKey, error) {
	// This is a simplified implementation
	// In production, you should use proper base64url decoding
	// For now, we'll return a placeholder
	return &rsa.PublicKey{}, nil
}

// Claims represents JWT claims
type Claims struct {
	UserID    string                 `json:"user_id"`
	Email     string                 `json:"email"`
	IssuedAt  time.Time              `json:"iat"`
	ExpiresAt time.Time              `json:"exp"`
	Extra     map[string]interface{} `json:"extra"`
}

// Health checks the health of the auth client
func (c *Client) Health(ctx context.Context) error {
	c.logger.Debug("Auth health check")
	// TODO: Implement actual health check
	return nil
}
