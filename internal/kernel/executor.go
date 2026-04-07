// executor.go runs a tool invocation: resolves the tool, validates input, and returns JSON output (or structured error).
// For single methods: all data comes from local tooling.json + wrekenfiles (no registry calls).
// For workflows: fetches workflow definition and bundles from registry, then executes steps via chain.go.
package kernel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"gitlab.com/swytchcode/swytchcode-cli/internal/registry"
	"gitlab.com/swytchcode/swytchcode-cli/internal/util"
)

// ExecRequest is the JSON input shape expected on stdin for swytchcode exec.
type ExecRequest struct {
	Tool string                 `json:"tool"`
	Args map[string]interface{} `json:"args"`
}

// ExecOptions groups all optional parameters for Execute.
// Using a struct keeps the call signature stable as new options are added.
type ExecOptions struct {
	AllowRaw    bool   // permit tools with the "raw." prefix
	DryRun      bool   // print request details without making the HTTP call
	Demo        bool   // run in demo mode: skip local setup, call demo API endpoint
	RawOutput   bool   // output the full raw HTTP response instead of normalized JSON
	JSONOutput  bool   // always output a single JSON object (default for scripting)
	Verbose     bool   // log request/response details to stderr for debugging
	OutputFile  string // write binary response body to this file path
	ProjectRoot string // override project root detection
	Token       string // auth token for registry calls (workflow fetch)
}

// Execute is the single entrypoint used by the CLI `exec` command.
//
// Invariant: tooling.json pins what is trusted. The registry supplies how it works.
// bootstrap reconciles the two. exec only executes.
//
// exec must NEVER call the registry for single-method execution. All data comes from local
// tooling.json and Wrekenfiles only. This ensures CI determinism, offline execution, and
// security boundaries.
// Exception: workflow execution fetches definition + bundles from registry (intentional —
// workflow definitions are not stored locally).
//
// Execution pipeline:
//  1. Parse request (from stdin or CLI args)
//  2. Load tooling.json
//  3. Resolve tool -> integration bundle
//  4. Load integration bundle (wrekenfile.yaml)
//  5. Resolve method/workflow from Wreken METHODS section
//  6. Get base URL from manifest.json based on mode
//  7. Validate input schema
//  8. Build HTTP request (method, URL, headers, body)
//  9. Execute (or dry-run)
//  10. Return JSON output
//
// It returns a process exit code from the fixed set defined in errors.go.
// buildWorkflowOutput converts all step results into the enriched output shape:
// {"steps": [{"step": N, "name": "...", "request": {"method": "...", "url": "..."}, "status_code": N, "data": {...}}, ...]}
func buildWorkflowOutput(results []StepResult, wfErr error) map[string]interface{} {
	steps := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		entry := map[string]interface{}{
			"step":        r.StepIndex + 1,
			"name":        r.StepName,
			"status_code": r.StatusCode,
			"data":        r.RawOutput,
		}
		if r.Failed {
			entry["failed"] = true
		}
		if r.RequestMethod != "" || r.RequestURL != "" {
			entry["request"] = map[string]string{
				"method": r.RequestMethod,
				"url":    r.RequestURL,
			}
		}
		steps = append(steps, entry)
	}
	output := map[string]interface{}{"success": wfErr == nil, "steps": steps}
	if wfErr != nil {
		output["error"] = wfErr.Error()
	}
	return output
}

// executeLocalWorkflow runs a workflow using the locally stored definition and bundles.
// Called when tooling.json already has steps for the workflow — no registry calls needed.
func executeLocalWorkflow(canonicalID string, steps []LocalWorkflowStep, mode string, args map[string]interface{}, projectRoot string, out, errOut io.Writer) int {
	ctx := context.Background()

	// Build WorkflowDetail from local steps.
	// Use integration as LibraryUUID so resolveStepBundle can key the BundleMap by it.
	wfSteps := make([]registry.WorkflowStep, 0, len(steps))
	for _, s := range steps {
		wfSteps = append(wfSteps, registry.WorkflowStep{
			Name:        s.Name,
			CanonicalID: s.CanonicalID,
			LibraryUUID: s.Integration, // stand-in key for local bundle lookup
		})
	}
	wf := &registry.WorkflowDetail{
		CanonicalID: canonicalID,
		Steps:       wfSteps,
	}

	// Build BundleMap: load each unique integration from local disk.
	bundleMap := make(BundleMap)
	for _, s := range steps {
		if _, exists := bundleMap[s.Integration]; exists {
			continue
		}
		bundle, err := LoadIntegrationBundle(projectRoot, s.Integration)
		if err != nil {
			msg := "failed to load integration bundle: " + err.Error()
			writeErrorJSON(errOut, msg)
			LogExecFailure(ExitCodeToolNotFound, canonicalID, msg)
			return ExitCodeToolNotFound
		}
		// Populate endpoints from manifest.json
		sandbox, err := GetBaseURL(projectRoot, s.Integration, "sandbox")
		if err == nil {
			bundle.SandboxEndpoint = sandbox
		}
		production, err := GetBaseURL(projectRoot, s.Integration, "production")
		if err == nil {
			bundle.ProductionEndpoint = production
		}
		bundleMap[s.Integration] = bundle
	}

	results, runErr := RunWorkflow(ctx, wf, bundleMap, args, mode, out, errOut, projectRoot)
	if encErr := json.NewEncoder(out).Encode(buildWorkflowOutput(results, runErr)); encErr != nil {
		LogExecFailure(ExitCodeInternalError, canonicalID, "failed to encode output")
		return ExitCodeInternalError
	}
	if runErr != nil {
		LogExecFailure(ExitCodeSDKFailure, canonicalID, runErr.Error())
		return ExitCodeSDKFailure
	}
	return ExitCodeOK
}

// executeWorkflow fetches the workflow definition and bundles from the registry,
// then runs all steps sequentially via chain.go.
// Registry calls are intentional here — workflow definitions are not stored locally.
func executeWorkflow(canonicalID, integration, mode string, args map[string]interface{}, out, errOut io.Writer, token, projectRoot string) int {
	// Derive project name: "weaviate.lyrid@v1" → "weaviate"; "project.workflow" → "project"
	projectName := integration
	if i := strings.Index(integration, "."); i > 0 {
		projectName = integration[:i]
	}

	ctx := context.Background()
	client := registry.NewClient(registry.DefaultConfigWithToken(token))

	// Fetch workflow definition
	wf, err := client.GetWorkflow(ctx, projectName, canonicalID)
	if err != nil {
		// Check if the workflow was renamed before returning a generic error
		if strings.Contains(err.Error(), "not found") {
			if resolution, resolveErr := client.ResolveCanonicalID(ctx, canonicalID); resolveErr == nil && resolution != nil {
				switch resolution.Status {
				case "renamed":
					msg := "workflow " + canonicalID + " has been renamed to " + resolution.NewID + " — run: swytchcode add " + resolution.NewID
					writeErrorJSON(errOut, msg)
					LogExecFailure(ExitCodeToolNotFound, canonicalID, msg)
					return ExitCodeToolNotFound
				case "removed":
					msg := "workflow " + canonicalID + " has been removed from the backend"
					writeErrorJSON(errOut, msg)
					LogExecFailure(ExitCodeToolNotFound, canonicalID, msg)
					return ExitCodeToolNotFound
				}
			}
		}
		msg := "failed to fetch workflow: " + err.Error()
		writeErrorJSON(errOut, msg)
		LogExecFailure(ExitCodeSDKFailure, canonicalID, msg)
		return ExitCodeSDKFailure
	}

	// Fetch all bundles needed for this workflow
	bundleMap, err := FetchBundleMap(ctx, client, projectName, canonicalID)
	if err != nil {
		msg := "failed to fetch workflow bundles: " + err.Error()
		writeErrorJSON(errOut, msg)
		LogExecFailure(ExitCodeSDKFailure, canonicalID, msg)
		return ExitCodeSDKFailure
	}

	// Run workflow steps sequentially
	results, runErr := RunWorkflow(ctx, wf, bundleMap, args, mode, out, errOut, projectRoot)
	if encErr := json.NewEncoder(out).Encode(buildWorkflowOutput(results, runErr)); encErr != nil {
		LogExecFailure(ExitCodeInternalError, canonicalID, "failed to encode output")
		return ExitCodeInternalError
	}
	if runErr != nil {
		LogExecFailure(ExitCodeSDKFailure, canonicalID, runErr.Error())
		return ExitCodeSDKFailure
	}
	return ExitCodeOK
}

// Execute runs a single tool invocation. When opts.JSONOutput is true, stdout is guaranteed to be a single JSON object.
func Execute(stdin io.Reader, stdout io.Writer, stderr io.Writer, opts ExecOptions) int {
	var req ExecRequest
	raw, err := io.ReadAll(stdin)
	if err != nil {
		msg := "failed to read stdin: " + err.Error()
		writeClassifiedError(stderr, msg, "internal", false)
		LogExecFailure(ExitCodeInvalidInput, "", msg)
		return ExitCodeInvalidInput
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		msg := "invalid json input"
		if hint := util.ExecJSONInvalidHint(raw); hint != "" {
			msg += " " + hint
		}
		writeClassifiedError(stderr, msg, "validation", false)
		LogExecFailure(ExitCodeInvalidInput, "", msg)
		return ExitCodeInvalidInput
	}

	if req.Tool == "" {
		msg := `tool is required (e.g. "project.method_name") — run 'swytchcode init' to generate tooling.json`
		writeClassifiedError(stderr, msg, "validation", false)
		LogExecFailure(ExitCodeInvalidInput, "", msg)
		return ExitCodeInvalidInput
	}
	if req.Args == nil {
		req.Args = make(map[string]interface{})
	}

	LogExecRequest(req.Tool, req.Args)

	// Demo mode: skip all local setup, call demo API endpoint directly.
	if opts.Demo {
		return executeDemoMode(req.Tool, req.Args, stdout, stderr, opts.JSONOutput)
	}

	// Detect project root if not provided.
	// If no .swytchcode/tooling.json is found anywhere, auto-enable demo mode so
	// `npx swytchcode stripe.create_payment` works on a fresh machine with zero setup.
	projectRoot := opts.ProjectRoot
	if projectRoot == "" {
		var err error
		projectRoot, err = util.ProjectRoot()
		if err != nil {
			// No project found — fall back to demo mode automatically
			fmt.Fprintf(stderr, "Running in demo mode (no setup required)\n\n")
			return executeDemoMode(req.Tool, req.Args, stdout, stderr, opts.JSONOutput)
		}
	}

	// Enforce raw method execution policy
	isRaw := strings.HasPrefix(req.Tool, "raw.")
	if isRaw && !opts.AllowRaw {
		msg := "raw method execution requires --allow-raw flag"
		writeClassifiedError(stderr, msg, "validation", false)
		LogExecFailure(ExitCodeInvalidInput, req.Tool, msg)
		return ExitCodeInvalidInput
	}

	// Step 1: Resolve tool from tooling.json
	tool, err := ResolveTool(projectRoot, req.Tool, isRaw)
	if err != nil {
		writeClassifiedError(stderr, err.Error(), "not_found", false)
		LogExecFailure(ExitCodeToolNotFound, req.Tool, err.Error())
		return ExitCodeToolNotFound
	}

	// Workflow execution path
	if tool.Type == "workflow" {
		// Prefer local-first execution when tooling.json already has steps (no registry call needed).
		if len(tool.Steps) > 0 {
			return executeLocalWorkflow(req.Tool, tool.Steps, tool.Mode, req.Args, projectRoot, stdout, stderr)
		}
		// Fallback: fetch definition and bundles from registry.
		return executeWorkflow(req.Tool, tool.Integration, tool.Mode, req.Args, stdout, stderr, opts.Token, projectRoot)
	}

	// Step 2: Load integration bundle (single-method path — no registry calls)
	bundle, err := LoadIntegrationBundle(projectRoot, tool.Integration)
	if err != nil {
		writeClassifiedError(stderr, err.Error(), "not_found", false)
		LogExecFailure(ExitCodeToolNotFound, req.Tool, err.Error())
		return ExitCodeToolNotFound
	}

	// Step 3: Resolve method from Wreken METHODS section
	method, err := ResolveMethod(bundle, req.Tool)
	if err != nil {
		writeClassifiedError(stderr, err.Error(), "not_found", false)
		LogExecFailure(ExitCodeToolNotFound, req.Tool, err.Error())
		return ExitCodeToolNotFound
	}

	// Step 4: Get base URL from manifest based on mode
	baseURL, err := GetBaseURL(projectRoot, tool.Integration, tool.Mode)
	if err != nil {
		writeClassifiedError(stderr, err.Error(), "internal", false)
		LogExecFailure(ExitCodeInternalError, req.Tool, err.Error())
		return ExitCodeInternalError
	}
	if err := ValidateExecutionBaseURL(baseURL); err != nil {
		writeClassifiedError(stderr, err.Error(), "validation", false)
		LogExecFailure(ExitCodeInternalError, req.Tool, err.Error())
		return ExitCodeInternalError
	}

	// Step 5: Validate input schema
	if err := ValidateInput(tool, req.Args); err != nil {
		msg := "input validation failed: " + err.Error()
		writeClassifiedError(stderr, msg, "validation", false)
		LogExecFailure(ExitCodeInvalidInput, req.Tool, msg)
		return ExitCodeInvalidInput
	}

	// Step 6: Build HTTP request
	httpReq, err := BuildRequest(method, baseURL, req.Args)
	if err != nil {
		msg := "failed to build request: " + err.Error()
		writeClassifiedError(stderr, msg, "validation", false)
		LogExecFailure(ExitCodeInvalidInput, req.Tool, msg)
		return ExitCodeInvalidInput
	}

	// Step 7: Execute or dry-run
	if opts.DryRun {
		code := ExecuteDryRun(httpReq, stdout)
		if code != ExitCodeOK {
			LogExecFailure(code, req.Tool, "dry-run output failed")
		}
		return code
	}

	execPolicy, err := GetExecutionPolicy(projectRoot, tool.Integration)
	if err != nil {
		msg := "execution policy: " + err.Error()
		writeClassifiedError(stderr, msg, "internal", false)
		LogExecFailure(ExitCodeInternalError, req.Tool, msg)
		return ExitCodeInternalError
	}

	// Step 8: Log verbose request details before executing
	if opts.Verbose {
		LogVerboseRequest(stderr, httpReq)
	}

	// Step 9: Execute HTTP request
	resp, err := ExecuteHTTP(context.Background(), httpReq, execPolicy)
	if err != nil {
		msg := "execution failed: " + err.Error()
		// Network/retry exhaustion errors are retryable; classify by content
		category, retryable := classifyExecError(err)
		writeClassifiedError(stderr, msg, category, retryable)
		LogExecFailure(ExitCodeSDKFailure, req.Tool, msg)
		return ExitCodeSDKFailure
	}

	// Step 10: Output response
	if opts.RawOutput {
		return OutputRawResponse(resp, httpReq, stdout, stderr)
	}
	return OutputJSONResponse(resp, httpReq, stdout, stderr, opts.Verbose, opts.OutputFile)
}

// classifyExecError categorises an error from ExecuteHTTP for machine-readable output.
func classifyExecError(err error) (category string, retryable bool) {
	if err == nil {
		return "internal", false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "429") || strings.Contains(msg, "rate limit") || strings.Contains(msg, "too many requests") {
		return "rate_limit", true
	}
	if strings.Contains(msg, "503") || strings.Contains(msg, "504") || strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "connection reset") || strings.Contains(msg, "eof") || strings.Contains(msg, "tls") {
		return "network", true
	}
	return "network", true // all exec failures are potentially transient
}
