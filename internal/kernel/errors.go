// errors.go defines exit codes and the JSON error shape used by the kernel on failure.
package kernel

import (
	"gitlab.com/swytchcode/shell/internal/util"
	"io"
)

// Exit codes are part of the public contract and must not change casually.
const (
	ExitCodeOK             = 0
	ExitCodeInvalidInput   = 1
	ExitCodeToolNotFound   = 2
	ExitCodeAuthError      = 3
	ExitCodeSDKFailure     = 4
	ExitCodeInternalError  = 5
)

// errorResponse is the JSON error shape written to stderr on failure.
type errorResponse struct {
	Error string `json:"error"`
}

// writeErrorJSON writes a JSON error object to the provided writer.
// It is deliberately lossy: if encoding fails, it does not attempt to
// recover, since we are already in an error path.
func writeErrorJSON(w io.Writer, msg string) {
	_ = util.WriteJSON(w, errorResponse{Error: msg})
}

