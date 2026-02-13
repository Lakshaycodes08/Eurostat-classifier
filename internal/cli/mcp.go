// mcp.go implements swytchcode mcp serve: starts the MCP server.
package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/constants"
	"gitlab.com/swytchcode/shell/internal/mcp"
)

var (
	mcpDaemon   bool
	mcpLogFile  string
	mcpTransport string
	mcpPort     int
)

// mcpCmd implements `swytchcode mcp`.
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP server commands",
	Long:  "Commands for running the Model Context Protocol server.",
}

// mcpServeCmd implements `swytchcode mcp serve`.
var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server",
	Long: `Start the Model Context Protocol server.

The server exposes swytchcode commands (list, get, add, exec) as MCP tools.
Supports both stdio and HTTP transports.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// In MCP mode, we cannot stream output to stdout/stderr as they're used for JSON-RPC protocol
		// All tool output must be captured and returned as part of the MCP tool result
		streamOutput := false

		// Create server
		server, err := mcp.NewServer(streamOutput, mcpLogFile)
		if err != nil {
			return fmt.Errorf("create MCP server: %w", err)
		}
		defer server.Close()

		// Setup context with cancellation
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle signals for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigChan
			cancel()
		}()

		// Start server based on transport
		if mcpTransport == "http" {
			httpServer := mcp.NewHTTPServer(server.GetMCPServer(), mcpPort)
			if mcpDaemon {
				// Daemon mode: return control immediately
				go func() {
					if err := httpServer.Serve(ctx); err != nil {
						fmt.Fprintf(os.Stderr, "MCP HTTP server error: %v\n", err)
					}
				}()
				return nil
			}
			return httpServer.Serve(ctx)
		} else {
			// stdio transport
			return server.ServeStdio(ctx)
		}
	},
}

func init() {
	mcpCmd.AddCommand(mcpServeCmd)
	mcpServeCmd.Flags().BoolVarP(&mcpDaemon, "daemon", "d", false, "run in daemon mode (background, no output)")
	mcpServeCmd.Flags().StringVar(&mcpLogFile, "log-file", "", "path to log file (only used in daemon mode)")
	mcpServeCmd.Flags().StringVar(&mcpTransport, "transport", "stdio", "transport type (stdio or http)")
	mcpServeCmd.Flags().IntVar(&mcpPort, "port", constants.MCPDefaultPort, "port for HTTP transport")
}
