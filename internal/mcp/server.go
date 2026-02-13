// server.go implements the MCP server.
package mcp

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server wraps the MCP server with configuration.
type Server struct {
	mcpServer   *mcp.Server
	streamOutput bool
	logFile     *os.File
}

// NewServer creates a new MCP server.
func NewServer(streamOutput bool, logFilePath string) (*Server, error) {
	server := &Server{
		mcpServer:   mcp.NewServer(&mcp.Implementation{Name: "swytchcode", Version: "1.0.0"}, nil),
		streamOutput: streamOutput,
	}

	// Setup logging
	if logFilePath != "" {
		logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("open log file: %w", err)
		}
		server.logFile = logFile
		log.SetOutput(logFile)
	} else if !streamOutput {
		// Daemon mode without log file: suppress logs
		log.SetOutput(io.Discard)
	}

	// Register tools
	if err := RegisterTools(server.mcpServer, streamOutput); err != nil {
		return nil, fmt.Errorf("register tools: %w", err)
	}

	return server, nil
}

// ServeStdio serves the MCP server over stdio.
func (s *Server) ServeStdio(ctx context.Context) error {
	return s.mcpServer.Run(ctx, &mcp.StdioTransport{})
}

// GetMCPServer returns the underlying MCP server.
func (s *Server) GetMCPServer() *mcp.Server {
	return s.mcpServer
}

// Close closes the server and any open resources.
func (s *Server) Close() error {
	if s.logFile != nil {
		return s.logFile.Close()
	}
	return nil
}
