// mcp.go implements swytchcode mcp serve: starts the MCP server.
package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/swytchcode-cli/internal/constants"
	"gitlab.com/swytchcode/swytchcode-cli/internal/mcp"
	"gitlab.com/swytchcode/swytchcode-cli/internal/util"
)

var (
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

The server exposes all swytchcode commands as MCP tools:
- swytchcode_init, swytchcode_bootstrap, swytchcode_version
- swytchcode_list, swytchcode_search, swytchcode_get, swytchcode_add
- swytchcode_info, swytchcode_exec

Supports both stdio and HTTP transports.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read daemon flag value
		daemonMode, _ := cmd.Flags().GetBool("daemon")

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

		// Handle signals for graceful shutdown (cross-platform)
		sigChan := make(chan os.Signal, 1)
		if runtime.GOOS == "windows" {
			// Windows: only Interrupt is available
			signal.Notify(sigChan, os.Interrupt)
		} else {
			// Unix/Linux/macOS: Interrupt and SIGTERM
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		}
		go func() {
			<-sigChan
			cancel()
		}()

		// Start server based on transport
		if daemonMode {
			// Daemon mode: fork process and run server in background
			// Note: stdio transport doesn't work in daemon mode (needs stdin/stdout)
			if mcpTransport != "http" {
				return fmt.Errorf("stdio transport cannot be used in daemon mode; use --transport http")
			}

			// Re-execute ourselves without -d flag to run as child process
			executable, err := os.Executable()
			if err != nil {
				return fmt.Errorf("get executable path: %w", err)
			}

			// Build command args (without -d flag to avoid re-forking)
			args := []string{"mcp", "serve", "--transport", mcpTransport, "--port", fmt.Sprintf("%d", mcpPort)}
			if mcpLogFile != "" {
				args = append(args, "--log-file", mcpLogFile)
			}

			cmd := exec.Command(executable, args...)
			
			// Platform-specific daemonization
			cmd.SysProcAttr = getSysProcAttr()
			
			// Redirect stdin/stdout/stderr to null device (cross-platform)
			// The child process will handle log file redirection if --log-file is specified
			devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
			if err == nil {
				cmd.Stdin = devNull
				cmd.Stdout = devNull
				cmd.Stderr = devNull
				// Don't close devNull here - let it be inherited by child process
				// The child will redirect to log file if needed
			}

			// Start process in background (properly daemonized)
			if err := cmd.Start(); err != nil {
				return fmt.Errorf("start daemon process: %w", err)
			}

			// Write PID file for the child process
			projectRoot, err := util.ProjectRoot()
			if err != nil {
				cmd.Process.Kill()
				return fmt.Errorf("detect project root: %w", err)
			}

			// Write PID file with child process PID
			pidFile := mcp.PIDFile(projectRoot)
			pidDir := filepath.Dir(pidFile)
			if err := os.MkdirAll(pidDir, 0o755); err != nil {
				cmd.Process.Kill()
				return fmt.Errorf("create pid directory: %w", err)
			}
			if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0o644); err != nil {
				cmd.Process.Kill()
				return fmt.Errorf("write PID file: %w", err)
			}

			// Detach from child process and return control
			cmd.Process.Release()
			return nil
		}

		// Non-daemon mode: block until server stops
		// Write PID file for tracking (even in non-daemon mode, for consistency)
		projectRoot, err := util.ProjectRoot()
		if err == nil {
			mcp.WritePID(projectRoot) // Ignore errors in non-daemon mode
		}

		if mcpTransport == "http" {
			fmt.Fprintf(os.Stderr, "MCP server listening on http://127.0.0.1:%d/sse\nPress Ctrl+C to stop.\n", mcpPort)
			httpServer := mcp.NewHTTPServer(server.GetMCPServer(), mcpPort)
			err := httpServer.Serve(ctx)
			// Clean up PID file when server stops
			if projectRoot != "" {
				mcp.RemovePID(projectRoot)
			}
			return err
		} else {
			// stdio transport
			err := server.ServeStdio(ctx)
			// Clean up PID file when server stops
			if projectRoot != "" {
				mcp.RemovePID(projectRoot)
			}
			return err
		}
	},
}

// mcpStatusCmd implements `swytchcode mcp status`.
var mcpStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check MCP server status",
	Long:  "Check if the MCP server is running (daemon mode only).",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return fmt.Errorf("detect project root: %w", err)
		}

		pid, err := mcp.ReadPID(projectRoot)
		if err != nil {
			fmt.Println("MCP server is not running")
			return nil
		}

		if mcp.IsProcessRunning(pid) {
			fmt.Printf("MCP server is running (PID: %d)\n", pid)
			return nil
		}

		// PID file exists but process is not running - clean up stale PID file
		mcp.RemovePID(projectRoot)
		fmt.Println("MCP server is not running (stale PID file removed)")
		return nil
	},
}

// mcpStopCmd implements `swytchcode mcp stop`.
var mcpStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the MCP server",
	Long:  "Stop the running MCP server (daemon mode only).",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return fmt.Errorf("detect project root: %w", err)
		}

		pid, err := mcp.ReadPID(projectRoot)
		if err != nil {
			return fmt.Errorf("MCP server is not running: %w", err)
		}

		if !mcp.IsProcessRunning(pid) {
			// Stale PID file - clean it up
			mcp.RemovePID(projectRoot)
			return fmt.Errorf("MCP server is not running (stale PID file removed)")
		}

		if err := mcp.StopProcess(pid); err != nil {
			return fmt.Errorf("stop MCP server: %w", err)
		}

		// Wait a moment for graceful shutdown, then remove PID file
		// Note: In a real implementation, you might want to wait and check if process actually stopped
		mcp.RemovePID(projectRoot)
		fmt.Printf("MCP server stopped (PID: %d)\n", pid)
		return nil
	},
}

func init() {
	mcpCmd.AddCommand(mcpServeCmd)
	mcpCmd.AddCommand(mcpStatusCmd)
	mcpCmd.AddCommand(mcpStopCmd)
	// Note: Both -d and --daemon work, but only -d is documented
	mcpServeCmd.Flags().BoolP("daemon", "d", false, "run in daemon mode (background, no output)")
	mcpServeCmd.Flags().StringVar(&mcpLogFile, "log-file", "", "path to log file (only used in daemon mode)")
	mcpServeCmd.Flags().StringVar(&mcpTransport, "transport", "stdio", "transport type (stdio or http)")
	mcpServeCmd.Flags().IntVar(&mcpPort, "port", constants.MCPDefaultPort, "port for HTTP transport")
}
