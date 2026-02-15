// errors.go defines exit codes and the JSON error shape used by the kernel on failure.
package kernel

import (
	"encoding/json"
	"io"
	"log"
	"strings"

	"gitlab.com/swytchcode/shell/internal/util"
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

// sensitiveKeys are argument keys whose values are redacted in request logs.
var sensitiveKeys = map[string]bool{
	"authorization": true, "token": true, "api_key": true, "apikey": true,
	"password": true, "secret": true, "bearer": true,
}

// sanitizeArgs returns a copy of args with sensitive values replaced by "[REDACTED]".
func sanitizeArgs(args map[string]interface{}) map[string]interface{} {
	if len(args) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(args))
	for k, v := range args {
		keyLower := strings.ToLower(k)
		if sensitiveKeys[keyLower] || strings.Contains(keyLower, "token") || strings.Contains(keyLower, "secret") {
			out[k] = "[REDACTED]"
		} else {
			out[k] = v
		}
	}
	return out
}

// LogExecRequest logs the exec input request (tool and sanitized args). Used for both CLI and MCP.
func LogExecRequest(tool string, args map[string]interface{}) {
	sanitized := sanitizeArgs(args)
	var argsStr string
	if len(sanitized) > 0 {
		b, _ := json.Marshal(sanitized)
		argsStr = " " + string(b)
	}
	log.Printf("[swytchcode exec] request tool=%s%s", tool, argsStr)
}

// LogExecFailure logs when exec fails so it appears in process log (e.g. MCP daemon log file).
// tool may be empty if the request could not be parsed.
func LogExecFailure(exitCode int, tool, errMsg string) {
	if tool == "" {
		log.Printf("[swytchcode exec] failed exit_code=%d error=%s", exitCode, errMsg)
	} else {
		log.Printf("[swytchcode exec] failed tool=%s exit_code=%d error=%s", tool, exitCode, errMsg)
	}
}
