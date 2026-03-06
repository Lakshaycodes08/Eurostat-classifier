// constants.go defines shared constants for timeouts, MCP settings, and other configuration.
package constants

import "time"

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

// MCP server configuration
const (
	// MCPDefaultPort is the default port for HTTP transport.
	MCPDefaultPort = 3000

	// MCPBearerToken is the bearer token for HTTP transport authentication (temporary constant).
	MCPBearerToken = "swytchcode-mcp-token"

	// MCPRequestTimeout is the timeout for MCP tool execution.
	MCPRequestTimeout = 5 * time.Minute
)

// Application configuration (build-time constants)
const (
	// Version is the Swytchcode shell version.
	Version = "1.0.2"

	// RegistryURL is the default registry base URL (build-time constant).
	// Set this at build time; runtime environment variables are not used.
	// RegistryURL = "https://dev-api-v2.swytchcode.world"
	RegistryURL = "https://api-v2.swytchcode.com"
)
