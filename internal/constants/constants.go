// constants.go defines shared constants for timeouts, MCP settings, and other configuration.
package constants

import (
	"crypto/tls"
	"net/http"
	"os"
	"time"
)

// HTTP client timeouts
const (
	// HTTPClientTimeout is the timeout for HTTP client requests (registry, exec).
	HTTPClientTimeout = 30 * time.Second

	// HTTPIdleConnTimeout is the timeout for idle HTTP connections.
	HTTPIdleConnTimeout = 90 * time.Second

	// HTTPMaxIdleConns is the maximum number of idle connections.
	HTTPMaxIdleConns = 100

	// HTTPMaxIdleConnsPerHost is the maximum number of idle connections per host.
	HTTPMaxIdleConnsPerHost = 10
)

// Auth session timing
const (
	// SessionTokenDurationSecs is the lifetime of a Firebase JWT issued at login.
	SessionTokenDurationSecs int64 = 3600

	// SessionRefreshBufferSecs is how many seconds before expiry to treat a token as expired.
	// Firebase ID tokens last 3600s; refresh when less than 300s remain.
	SessionRefreshBufferSecs int64 = 300

	// AuthRequestTimeout is the timeout for authentication HTTP requests (login, refresh).
	AuthRequestTimeout = 10 * time.Second
)

// MCP server configuration
const (
	// MCPDefaultPort is the default port for HTTP transport.
	MCPDefaultPort = 3000

	// MCPBearerToken is the bearer token for HTTP transport authentication (temporary constant).
	MCPBearerToken = "swytchcode-mcp-token"

	// MCPRequestTimeout is the timeout for MCP tool execution.
	MCPRequestTimeout = 5 * time.Minute
)

// NewHTTPClient returns an *http.Client with the given timeout.
// When SWYTCHCODE_INSECURE=1 is set, TLS certificate verification is skipped
// (intended for local dev with self-signed certificates only).
func NewHTTPClient(timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if os.Getenv("SWYTCHCODE_INSECURE") == "1" {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}
	return &http.Client{Timeout: timeout, Transport: transport}
}

// Application configuration (build-time constants)
const (
	// Version is the Swytchcode shell version.
	Version = "1.0.2"

	// RegistryURL is the default registry base URL (build-time constant).
	// Set this at build time; runtime environment variables are not used.
	// RegistryURL = "https://dev-api-v2.swytchcode.world"
	RegistryURL = "https://api-v2.swytchcode.com"
)
