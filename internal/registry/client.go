// client.go provides an HTTP client for the registry API (timeouts, base URL from config).
package registry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client wraps HTTP client for registry API calls.
type Client struct {
	httpClient *http.Client
	config     *Config
}

// NewClient creates a new registry client with default configuration.
func NewClient(config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
		config: config,
	}
}

// Get performs a GET request to the registry API.
func (c *Client) Get(ctx context.Context, path string) (*http.Response, error) {
	url := c.config.APIBasePath() + path
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// Post performs a POST request to the registry API.
func (c *Client) Post(ctx context.Context, path string, body io.Reader) (*http.Response, error) {
	url := c.config.APIBasePath() + path
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// CloseIdleConnections closes idle connections in the HTTP client.
func (c *Client) CloseIdleConnections() {
	c.httpClient.Transport.(*http.Transport).CloseIdleConnections()
}
