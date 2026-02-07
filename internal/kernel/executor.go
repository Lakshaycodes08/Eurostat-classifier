// executor.go runs a single tool invocation: resolves the tool, validates input, and returns JSON output (or structured error).
package kernel

import (
	"encoding/json"
	"io"
	"strings"

	"gitlab.com/swytchcode/shell/internal/util"
)

// ExecRequest is the JSON input shape expected on stdin for swytchcode exec.
type ExecRequest struct {
	Tool string                 `json:"tool"`
	Args map[string]interface{} `json:"args"`
}

// Execute is the single entrypoint used by the CLI `exec` command.
//
// Invariant: tooling.json pins what is trusted. The registry supplies how it works.
// bootstrap reconciles the two. exec only executes.
//
// exec must NEVER call the registry. All data comes from local tooling.json and Wrekenfiles
// only. This ensures CI determinism, offline execution, and security boundaries.
//
// Decision tree:
//   - If tool starts with "raw.":
//       - If --allow-raw is false → FAIL (exit code 1)
//       - Resolve via Wrekenfile (bypass tooling.json)
//   - Else:
//       - Resolve via tooling.json only
//
// It:
//   1. Reads and parses JSON from stdin.
//   2. (TODO) Determines if tool is raw or verified.
//   3. (TODO) Loads tooling.json (for verified tools) or Wrekenfile (for raw).
//   4. (TODO) Resolves tool -> Wrekenfile.
//   5. (TODO) Validates args and required env vars.
//   6. (TODO) Executes the SDK call via language-specific adapters.
//   7. Writes JSON to stdout on success, stderr on failure.
//
// It returns a process exit code from the fixed set defined in errors.go.
func Execute(stdin io.Reader, stdout io.Writer, stderr io.Writer, allowRaw bool) int {
	var req ExecRequest
	if err := util.ReadJSON(stdin, &req); err != nil {
		writeErrorJSON(stderr, "invalid json input")
		return ExitCodeInvalidInput
	}

	if req.Tool == "" {
		writeErrorJSON(stderr, "tool is required")
		return ExitCodeInvalidInput
	}

	// Enforce raw method execution policy
	isRaw := strings.HasPrefix(req.Tool, "raw.")
	if isRaw && !allowRaw {
		writeErrorJSON(stderr, "raw method execution requires --allow-raw flag")
		return ExitCodeInvalidInput
	}

	// NOTE: for now we are not enforcing any particular structure on Args.
	// As the Wrekenfile and tooling.json schemas are fleshed out, this will
	// be validated against those contracts.

	// TODO: Implement decision tree:
	// If isRaw:
	//   1. Parse "raw.library.operation" format
	//   2. Load Wrekenfile for library (bypass tooling.json)
	//   3. Resolve operation from Wrekenfile
	//   4. Validate args against Wrekenfile schema
	// Else:
	//   1. Load tooling.json
	//   2. Verify tool exists in tooling.json
	//   3. Resolve tool -> library -> Wrekenfile
	//   4. Validate args against tooling.json schema
	// Common:
	//   5. Gather env credentials (util.GetEnvRequired)
	//   6. Apply policy (retries, idempotency keys)
	//   7. Execute SDK call and normalize output

	// Temporary stub response to prove the wiring.
	resp := map[string]any{
		"tool":  req.Tool,
		"args":  req.Args,
		"isRaw": isRaw,
		"note":  "kernel execution path not implemented yet; this is a stub response",
	}

	if err := json.NewEncoder(stdout).Encode(resp); err != nil {
		writeErrorJSON(stderr, "failed to encode response")
		return ExitCodeInternalError
	}

	return ExitCodeOK
}

