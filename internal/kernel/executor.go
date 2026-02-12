// executor.go runs a single tool invocation: resolves the tool, validates input, and returns JSON output (or structured error).
package kernel

import (
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
// Execution pipeline:
//   1. Parse request (from stdin or CLI args)
//   2. Load tooling.json
//   3. Resolve tool -> integration bundle
//   4. Load integration bundle (wrekenfile.yaml)
//   5. Resolve method/workflow from Wreken METHODS section
//   6. Get base URL from manifest.json based on mode
//   7. Validate input schema
//   8. Build HTTP request (method, URL, headers, body)
//   9. Execute (or dry-run)
//   10. Return JSON output
//
// It returns a process exit code from the fixed set defined in errors.go.
func Execute(stdin io.Reader, stdout io.Writer, stderr io.Writer, allowRaw bool, dryRun bool, rawOutput bool, projectRoot string) int {
	var req ExecRequest
	if err := util.ReadJSON(stdin, &req); err != nil {
		writeErrorJSON(stderr, "invalid json input")
		return ExitCodeInvalidInput
	}

	if req.Tool == "" {
		writeErrorJSON(stderr, "tool is required")
		return ExitCodeInvalidInput
	}

	// Detect project root if not provided
	if projectRoot == "" {
		var err error
		projectRoot, err = util.ProjectRoot()
		if err != nil {
			writeErrorJSON(stderr, "failed to detect project root: "+err.Error())
			return ExitCodeInternalError
		}
	}

	// Enforce raw method execution policy
	isRaw := strings.HasPrefix(req.Tool, "raw.")
	if isRaw && !allowRaw {
		writeErrorJSON(stderr, "raw method execution requires --allow-raw flag")
		return ExitCodeInvalidInput
	}

	// Step 1: Resolve tool from tooling.json
	tool, err := ResolveTool(projectRoot, req.Tool, isRaw)
	if err != nil {
		writeErrorJSON(stderr, err.Error())
		return ExitCodeToolNotFound
	}

	// Step 2: Load integration bundle
	bundle, err := LoadIntegrationBundle(projectRoot, tool.Integration)
	if err != nil {
		writeErrorJSON(stderr, err.Error())
		return ExitCodeToolNotFound
	}

	// Step 3: Resolve method from Wreken METHODS section
	method, err := ResolveMethod(bundle, req.Tool)
	if err != nil {
		writeErrorJSON(stderr, err.Error())
		return ExitCodeToolNotFound
	}

	// Step 4: Get base URL from manifest based on mode
	baseURL, err := GetBaseURL(projectRoot, tool.Integration, tool.Mode)
	if err != nil {
		writeErrorJSON(stderr, err.Error())
		return ExitCodeInternalError
	}

	// Step 5: Validate input schema
	if err := ValidateInput(tool, req.Args); err != nil {
		writeErrorJSON(stderr, "input validation failed: "+err.Error())
		return ExitCodeInvalidInput
	}

	// Step 6: Build HTTP request
	httpReq, err := BuildRequest(method, baseURL, req.Args)
	if err != nil {
		writeErrorJSON(stderr, "failed to build request: "+err.Error())
		return ExitCodeInvalidInput
	}

	// Step 7: Execute or dry-run
	if dryRun {
		return ExecuteDryRun(httpReq, stdout)
	}

	// Step 8: Execute HTTP request
	resp, err := ExecuteHTTP(httpReq)
	if err != nil {
		writeErrorJSON(stderr, "execution failed: "+err.Error())
		return ExitCodeSDKFailure
	}

	// Step 9: Output response
	if rawOutput {
		return OutputRawResponse(resp, stdout, stderr)
	}

	return OutputJSONResponse(resp, stdout, stderr)
}
