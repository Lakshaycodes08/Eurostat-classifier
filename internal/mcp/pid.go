// pid.go manages PID files for MCP server daemon mode.
package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
)

// PIDFile returns the path to the MCP server PID file.
func PIDFile(projectRoot string) string {
	return filepath.Join(projectRoot, ".swytchcode", "mcp.pid")
}

// WritePID writes the current process ID to the PID file.
func WritePID(projectRoot string) error {
	pidPath := PIDFile(projectRoot)
	dir := filepath.Dir(pidPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create pid directory: %w", err)
	}

	pid := os.Getpid()
	data := []byte(strconv.Itoa(pid))
	return os.WriteFile(pidPath, data, 0o644)
}

// ReadPID reads the process ID from the PID file.
func ReadPID(projectRoot string) (int, error) {
	pidPath := PIDFile(projectRoot)
	data, err := os.ReadFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("PID file not found: %s", pidPath)
		}
		return 0, fmt.Errorf("read PID file: %w", err)
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %w", err)
	}

	return pid, nil
}

// RemovePID removes the PID file.
func RemovePID(projectRoot string) error {
	pidPath := PIDFile(projectRoot)
	if err := os.Remove(pidPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove PID file: %w", err)
	}
	return nil
}

// IsProcessRunning checks if a process with the given PID is running.
func IsProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists (cross-platform)
	// On Unix: signal 0 checks if process exists without sending a signal
	// On Windows: Signal(0) also works to check process existence
	err = process.Signal(syscall.Signal(0))
	if runtime.GOOS == "windows" {
		// On Windows, if the error indicates process finished, it's not running
		if err != nil {
			errStr := err.Error()
			return errStr != "os: process already finished" && errStr != "Access is denied."
		}
	}
	return err == nil
}

// StopProcess sends SIGTERM (Unix) or terminates (Windows) the process with the given PID.
func StopProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}

	if runtime.GOOS == "windows" {
		// Windows: use Kill instead of SIGTERM (Windows doesn't have SIGTERM)
		if err := process.Kill(); err != nil {
			return fmt.Errorf("kill process %d: %w", pid, err)
		}
	} else {
		// Unix/Linux/macOS: send SIGTERM for graceful shutdown
		if err := process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("send SIGTERM to process %d: %w", pid, err)
		}
	}

	return nil
}
