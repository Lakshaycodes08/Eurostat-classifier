// transport.go implements HTTP transport for MCP server.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gitlab.com/swytchcode/cli/internal/constants"
)

// HTTPServer wraps HTTP transport for MCP.
type HTTPServer struct {
	mcpServer *mcp.Server
	port      int
}

// NewHTTPServer creates a new HTTP transport server.
func NewHTTPServer(mcpServer *mcp.Server, port int) *HTTPServer {
	return &HTTPServer{
		mcpServer: mcpServer,
		port:      port,
	}
}

// Serve starts the HTTP server.
func (s *HTTPServer) Serve(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", s.handleMCP)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(ctx)
	}()

	return server.ListenAndServe()
}

// handleMCP handles MCP requests over HTTP.
func (s *HTTPServer) handleMCP(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "missing Authorization header", http.StatusUnauthorized)
		return
	}

	expectedAuth := "Bearer " + constants.MCPBearerToken
	if authHeader != expectedAuth {
		http.Error(w, "invalid authorization token", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// For HTTP transport, we need to create a session and handle requests
	// This is a simplified version - full implementation would use HTTP transport
	// from the MCP SDK
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "HTTP transport not yet implemented"})
}
