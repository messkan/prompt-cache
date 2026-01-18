package http

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"time"
)

// ClientConfig holds HTTP client configuration
type ClientConfig struct {
	Timeout       time.Duration
	MaxRetries    int
	RetryBaseWait time.Duration
}

// DefaultClientConfig returns default HTTP client configuration
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Timeout:       30 * time.Second,
		MaxRetries:    3,
		RetryBaseWait: 500 * time.Millisecond,
	}
}

// NewClient creates a new HTTP client with the given configuration
func NewClient(cfg *ClientConfig) *http.Client {
	if cfg == nil {
		cfg = DefaultClientConfig()
	}

	return &http.Client{
		Timeout: cfg.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}

// RetryableClient wraps an HTTP client with retry logic
type RetryableClient struct {
	client        *http.Client
	maxRetries    int
	retryBaseWait time.Duration
}

// NewRetryableClient creates a new retryable HTTP client
func NewRetryableClient(cfg *ClientConfig) *RetryableClient {
	if cfg == nil {
		cfg = DefaultClientConfig()
	}

	return &RetryableClient{
		client:        NewClient(cfg),
		maxRetries:    cfg.MaxRetries,
		retryBaseWait: cfg.RetryBaseWait,
	}
}

// NewRetryableClientWithHTTPClient creates a RetryableClient with a custom http.Client (for testing)
func NewRetryableClientWithHTTPClient(client *http.Client) *RetryableClient {
	return &RetryableClient{
		client:        client,
		maxRetries:    0, // No retries when using custom client (usually for testing)
		retryBaseWait: 0,
	}
}

// Do executes the request with retry logic
func (c *RetryableClient) Do(req *http.Request) (*http.Response, error) {
	var lastErr error
	
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// Clone the request for retry (body needs special handling)
		reqClone := req.Clone(req.Context())
		
		resp, err := c.client.Do(reqClone)
		if err != nil {
			lastErr = err
			
			// Check if context was cancelled
			if req.Context().Err() != nil {
				return nil, req.Context().Err()
			}
			
			// Wait before retry
			if attempt < c.maxRetries {
				c.backoff(req.Context(), attempt)
			}
			continue
		}

		// Check for retryable status codes
		if c.isRetryable(resp.StatusCode) {
			lastErr = fmt.Errorf("HTTP %d: retryable error", resp.StatusCode)
			resp.Body.Close()
			
			if attempt < c.maxRetries {
				c.backoff(req.Context(), attempt)
			}
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// isRetryable checks if a status code is retryable
func (c *RetryableClient) isRetryable(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

// backoff waits with exponential backoff and jitter
func (c *RetryableClient) backoff(ctx context.Context, attempt int) {
	// Exponential backoff: baseWait * 2^attempt
	wait := c.retryBaseWait * time.Duration(math.Pow(2, float64(attempt)))
	
	// Add jitter (0-25% of wait time)
	jitter := time.Duration(rand.Float64() * 0.25 * float64(wait))
	wait += jitter

	// Cap at 30 seconds
	if wait > 30*time.Second {
		wait = 30 * time.Second
	}

	select {
	case <-time.After(wait):
	case <-ctx.Done():
	}
}

// GetClient returns the underlying HTTP client
func (c *RetryableClient) GetClient() *http.Client {
	return c.client
}
