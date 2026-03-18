// init.go provides shared init command logic for CLI and MCP.
package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"gitlab.com/swytchcode/cli/internal/constants"
	"gitlab.com/swytchcode/cli/internal/editors"
	"gitlab.com/swytchcode/cli/internal/output"
	"gitlab.com/swytchcode/cli/internal/util"
)

// RunInit runs the init command: creates .swytchcode/, tooling.json, and editor-specific config.
// editor and mode are required (non-interactive mode).
func RunInit(projectRoot, editor, mode string, stdout, stderr io.Writer) error {
	// Accumulate validation errors so both are shown at once
	var validationErrs []string
	if mode != "production" && mode != "sandbox" {
		validationErrs = append(validationErrs, fmt.Sprintf("invalid mode %q (expected production or sandbox)", mode))
	}
	validEditors := map[string]bool{"cursor": true, "claude": true, "none": true}
	if editor != "" && !validEditors[editor] {
		validationErrs = append(validationErrs, fmt.Sprintf("unknown editor %q (expected cursor|claude|none)", editor))
	}
	if len(validationErrs) > 0 {
		output.ValidationErrors(stderr, validationErrs)
		return fmt.Errorf("validation failed")
	}

	swytchDir := util.SwytchDir(projectRoot)
	if err := util.EnsureDir(swytchDir, 0o755); err != nil {
		return fmt.Errorf("create .swytchcode directory: %w", err)
	}
	// Create integrations directory where all integration data (wrekenfile, methods, workflows)
	// will be stored by swytchcode get.
	if err := util.EnsureDir(util.IntegrationsDir(projectRoot), 0o755); err != nil {
		return fmt.Errorf("create integrations directory: %w", err)
	}

	// Create or update tooling.json with mode
	toolingPath := util.ToolingPath(projectRoot)
	var tooling map[string]interface{}

	if data, err := os.ReadFile(toolingPath); err == nil {
		// Load existing tooling.json
		if err := json.Unmarshal(data, &tooling); err != nil {
			// If invalid, start fresh
			tooling = make(map[string]interface{})
		}
	} else {
		// Create new tooling.json
		tooling = make(map[string]interface{})
	}

	// Ensure tools and integrations maps exist (integrations = pinned versions for determinism)
	if _, ok := tooling["tools"]; !ok {
		tooling["tools"] = make(map[string]interface{})
	}
	if _, ok := tooling["integrations"]; !ok {
		tooling["integrations"] = make(map[string]interface{})
	}

	// Set mode and version so they are visible and project-specific.
	// version is owned by the kernel: only set when absent (never overwrite on re-init).
	tooling["mode"] = mode
	if _, hasVersion := tooling["version"]; !hasVersion {
		tooling["version"] = constants.Version
	}

	// Write tooling.json
	data, err := json.MarshalIndent(tooling, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tooling.json: %w", err)
	}
	if err := os.WriteFile(toolingPath, data, 0o644); err != nil {
		return fmt.Errorf("write tooling.json: %w", err)
	}

	// Write editor config if editor is specified
	if editor != "" && editor != "none" {
		switch editor {
		case "cursor":
			if err := editors.WriteCursorRules(projectRoot); err != nil {
				return fmt.Errorf("write Cursor rules: %w", err)
			}
			if err := editors.WriteCursorMCPConfig(); err != nil {
				output.Warn(stderr, "could not update ~/.cursor/mcp.json: "+err.Error())
			}
			startMCPDaemonIfNeeded(projectRoot, stderr)
		case "claude":
			if err := editors.WriteClaudeConfig(projectRoot); err != nil {
				return fmt.Errorf("write Claude config: %w", err)
			}
			if err := editors.WriteClaudeMCPConfig(); err != nil {
				output.Warn(stderr, "could not update ~/.claude/settings.json: "+err.Error())
			}
			startMCPDaemonIfNeeded(projectRoot, stderr)
		}
	}

	fmt.Fprintf(stdout, "Swytchcode initialized for project at %s\n", projectRoot)
	return nil
}

// startMCPDaemonIfNeeded starts the MCP HTTP daemon if one isn't already running.
// Spawns the HTTP server directly (single fork) with devNull I/O and a new session
// so it never interacts with the caller's terminal.
// Non-fatal: warns on error so init still succeeds.
func startMCPDaemonIfNeeded(projectRoot string, stderr io.Writer) {
	pidPath := util.MCPPIDPath(projectRoot)
	if data, err := os.ReadFile(pidPath); err == nil {
		if pid, err := strconv.Atoi(string(data)); err == nil {
			if proc, err := os.FindProcess(pid); err == nil {
				if proc.Signal(syscall.Signal(0)) == nil {
					return // already running
				}
			}
		}
	}

	executable, err := os.Executable()
	if err != nil {
		output.Warn(stderr, "could not resolve executable path: "+err.Error())
		return
	}

	// Spawn the HTTP server directly — no intermediate "daemon parent" process.
	// Redirect I/O to /dev/null so the child never interacts with our terminal.
	cmd := exec.Command(executable, "mcp", "serve", "--transport", "http",
		"--port", fmt.Sprintf("%d", constants.MCPDefaultPort))
	if devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0); err == nil {
		cmd.Stdin = devNull
		cmd.Stdout = devNull
		cmd.Stderr = devNull
	}
	cmd.SysProcAttr = daemonSysProcAttr() // detach from terminal (Setsid on Unix)

	if err := cmd.Start(); err != nil {
		output.Warn(stderr, "could not start MCP daemon: "+err.Error())
		return
	}

	// Write PID file immediately so `mcp status` works before child finishes starting.
	_ = os.WriteFile(pidPath, []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0o644)

	cmd.Process.Release() //nolint:errcheck
	fmt.Fprintf(stderr, "MCP server started on http://localhost:%d/sse\n", constants.MCPDefaultPort)
}
