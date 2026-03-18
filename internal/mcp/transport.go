// transport.go implements HTTP/SSE transport for the MCP server.
package mcp

import (
	"context"
	"fmt"
	"net/http"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// HTTPServer wraps HTTP/SSE transport for MCP.
type HTTPServer struct {
	mcpServer *sdkmcp.Server
	port      int
}

// NewHTTPServer creates a new HTTP/SSE transport server.
func NewHTTPServer(mcpServer *sdkmcp.Server, port int) *HTTPServer {
	return &HTTPServer{
		mcpServer: mcpServer,
		port:      port,
	}
}

// Serve starts the SSE HTTP server on localhost only.
// Editors connect to /sse for the event stream; JSON-RPC messages are posted to /message.
// Auth is not required — the server binds to 127.0.0.1 and is not reachable externally.
func (s *HTTPServer) Serve(ctx context.Context) error {
	handler := sdkmcp.NewSSEHandler(func(_ *http.Request) *sdkmcp.Server {
		return s.mcpServer
	}, nil)

	mux := http.NewServeMux()
	mux.Handle("/sse", handler)
	mux.Handle("/message", handler)

	srv := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", s.port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background()) //nolint:errcheck
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
