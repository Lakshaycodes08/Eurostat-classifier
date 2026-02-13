// output.go provides output capture and streaming for MCP tool handlers.
package mcp

import (
	"bytes"
	"io"
	"os"
)

// OutputCapture captures stdout and stderr while optionally streaming to terminal.
type OutputCapture struct {
	stdoutBuf *bytes.Buffer
	stderrBuf *bytes.Buffer
	stdoutWr  io.Writer
	stderrWr  io.Writer
	stream    bool // If true, also write to os.Stdout/os.Stderr
}

// NewOutputCapture creates a new output capture.
// If stream is true, output is also written to terminal.
func NewOutputCapture(stream bool) *OutputCapture {
	oc := &OutputCapture{
		stdoutBuf: &bytes.Buffer{},
		stderrBuf: &bytes.Buffer{},
		stream:    stream,
	}

	if stream {
		// Multi-writer: capture + terminal
		oc.stdoutWr = io.MultiWriter(oc.stdoutBuf, os.Stdout)
		oc.stderrWr = io.MultiWriter(oc.stderrBuf, os.Stderr)
	} else {
		// Only capture
		oc.stdoutWr = oc.stdoutBuf
		oc.stderrWr = oc.stderrBuf
	}

	return oc
}

// Stdout returns the stdout writer.
func (oc *OutputCapture) Stdout() io.Writer {
	return oc.stdoutWr
}

// Stderr returns the stderr writer.
func (oc *OutputCapture) Stderr() io.Writer {
	return oc.stderrWr
}

// GetOutput returns the captured stdout and stderr as strings.
func (oc *OutputCapture) GetOutput() (stdout, stderr string) {
	return oc.stdoutBuf.String(), oc.stderrBuf.String()
}

// GetCombinedOutput returns combined stdout and stderr.
func (oc *OutputCapture) GetCombinedOutput() string {
	stdout, stderr := oc.GetOutput()
	if stderr == "" {
		return stdout
	}
	if stdout == "" {
		return stderr
	}
	return stdout + "\n" + stderr
}
