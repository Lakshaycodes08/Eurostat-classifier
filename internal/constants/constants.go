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
	// SessionTokenDurationSecs is the lifetime of a CLI session token issued at login.
	SessionTokenDurationSecs int64 = 10800

	// SessionRefreshBufferSecs is how many seconds before expiry to treat a token as expired.
	// Sessions last 10800s (3h); refresh when less than 300s remain.
	SessionRefreshBufferSecs int64 = 300

	// AuthRequestTimeout is the timeout for authentication HTTP requests (login, refresh).
	AuthRequestTimeout = 10 * time.Second
)

// MCP server configuration
const (
	// MCPDefaultPort is the default port for HTTP transport.
	MCPDefaultPort = 5476

	// MCPBearerToken is the bearer token for HTTP transport authentication (temporary constant).
	MCPBearerToken = "swytchcode-mcp-token"

	// MCPRequestTimeout is the timeout for MCP tool execution.
	MCPRequestTimeout = 5 * time.Minute
)

// Telemetry configuration
const (
	// TelemetryTimeout is the max duration for the telemetry POST request (spec: 2s).
	TelemetryTimeout = 2 * time.Second

	// TelemetryBatchSizeMax is the maximum events per batch request (spec: 100).
	TelemetryBatchSizeMax = 100

	// EnvVarTelemetryMCP is the env var that, when set to "1", marks execution as MCP-sourced.
	EnvVarTelemetryMCP = "SWYTCHCODE_MCP"

	// EnvVarTelemetryDebug enables verbose telemetry logging when set (any non-empty value).
	// When enabled, telemetry sends and skips are logged to stderr with a [telemetry] prefix.
	EnvVarTelemetryDebug = "SWYTCHCODE_TELEMETRY_DEBUG"
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

// EnvVarsCI lists environment variable names that indicate CI execution (for telemetry source detection).
var EnvVarsCI = []string{
	"CI", "GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI", "TRAVIS", "JENKINS_URL", "BUILDKITE",
}

// Application configuration
//
// Note: Version must be a variable (not a const) so it can be overridden at build time
// via `-ldflags "-X gitlab.com/swytchcode/cli/internal/constants.Version=<version>"` (Goreleaser).
var (
	// Version is the Swytchcode CLI version (overridden by release builds).
	Version = "1.0.8"
)

// RegistryURL is the default registry base URL (build-time constant).
// Set this at build time; runtime environment variables are not used.
// RegistryURL = "https://dev-api-v2.swytchcode.world"
const RegistryURL = "https://api-v2.swytchcode.com"

// Project directory and file names
const (
	// SwytchDirName is the name of the per-project swytchcode directory.
	SwytchDirName = ".swytchcode"

	// IntegrationsDirName is the subdirectory within SwytchDirName that stores integration bundles.
	IntegrationsDirName = "integrations"

	// ToolingJSONFile is the filename for the project tooling configuration.
	ToolingJSONFile = "tooling.json"

	// WrekenfileYAMLFile is the filename for the Wrekenfile integration spec.
	WrekenfileYAMLFile = "wrekenfile.yaml"

	// MethodsJSONFile is the filename for the cached method list.
	MethodsJSONFile = "methods.json"

	// WorkflowsJSONFile is the filename for the cached workflow list.
	WorkflowsJSONFile = "workflows.json"

	// ManifestJSONFile is the filename for the integration manifest.
	ManifestJSONFile = "manifest.json"

	// MCPPIDFile is the filename for the MCP daemon PID file.
	MCPPIDFile = "mcp.pid"
)

// DefaultLocalEndpoint is the fallback base URL when a bundle has no endpoint configured.
// It indicates that the target API must be running locally.
const DefaultLocalEndpoint = "http://localhost"

// Wrekenfile section and field keys
const (
	WrekenMethods    = "METHODS"
	WrekenWorkflows  = "WORKFLOWS"
	WrekenStructs    = "STRUCTS"
	WrekenInputs     = "INPUTS"
	WrekenReturns    = "RETURNS"
	WrekenSummary    = "SUMMARY"
	WrekenDesc       = "DESC"
	WrekenHTTP       = "HTTP"
	WrekenEndpoint   = "ENDPOINT"
	WrekenHTTPMethod = "METHOD"
	WrekenHeaders    = "HEADERS"
	WrekenLocation   = "LOCATION"
)
