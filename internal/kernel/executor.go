// executor.go runs a tool invocation: resolves the tool, validates input, and returns JSON output (or structured error).
// For single methods: all data comes from local tooling.json + wrekenfiles (no registry calls).
// For workflows: fetches workflow definition and bundles from registry, then executes steps via chain.go.
package kernel

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	"gitlab.com/swytchcode/cli/internal/registry"
	"gitlab.com/swytchcode/cli/internal/util"
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
// exec must NEVER call the registry for single-method execution. All data comes from local
// tooling.json and Wrekenfiles only. This ensures CI determinism, offline execution, and
// security boundaries.
// Exception: workflow execution fetches definition + bundles from registry (intentional —
// workflow definitions are not stored locally).
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

	results, runErr := RunWorkflow(ctx, wf, bundleMap, args, mode, out, errOut)
	if encErr := json.NewEncoder(out).Encode(buildWorkflowOutput(results, runErr)); encErr != nil {
		LogExecFailure(ExitCodeInternalError, canonicalID, "failed to encode output")
		return ExitCodeInternalError
	}
	if runErr != nil {
		LogExecFailure(ExitCodeOK, canonicalID, runErr.Error())
	}
	return ExitCodeOK
}

// executeWorkflow fetches the workflow definition and bundles from the registry,
// then runs all steps sequentially via chain.go.
// Registry calls are intentional here — workflow definitions are not stored locally.
func executeWorkflow(canonicalID, integration, mode string, args map[string]interface{}, out, errOut io.Writer, token string) int {
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
	results, runErr := RunWorkflow(ctx, wf, bundleMap, args, mode, out, errOut)
	if encErr := json.NewEncoder(out).Encode(buildWorkflowOutput(results, runErr)); encErr != nil {
		LogExecFailure(ExitCodeInternalError, canonicalID, "failed to encode output")
		return ExitCodeInternalError
	}
	if runErr != nil {
		LogExecFailure(ExitCodeOK, canonicalID, runErr.Error())
	}
	return ExitCodeOK
}

// Execute runs a single tool invocation. When jsonOutput is true, stdout is guaranteed to be a single JSON object (normalized or raw per rawOutput).
func Execute(stdin io.Reader, stdout io.Writer, stderr io.Writer, allowRaw bool, dryRun bool, rawOutput bool, jsonOutput bool, projectRoot string, token string) int {
	var req ExecRequest
	if err := util.ReadJSON(stdin, &req); err != nil {
		msg := "invalid json input"
		writeErrorJSON(stderr, msg)
		LogExecFailure(ExitCodeInvalidInput, "", msg)
		return ExitCodeInvalidInput
	}

	if req.Tool == "" {
		msg := `tool is required (e.g. "project.method_name") — run 'swytchcode init' to generate tooling.json`
		writeErrorJSON(stderr, msg)
		LogExecFailure(ExitCodeInvalidInput, "", msg)
		return ExitCodeInvalidInput
	}
	if req.Args == nil {
		req.Args = make(map[string]interface{})
	}

	LogExecRequest(req.Tool, req.Args)

	// Detect project root if not provided
	if projectRoot == "" {
		var err error
		projectRoot, err = util.ProjectRoot()
		if err != nil {
			msg := "failed to detect project root: " + err.Error()
			writeErrorJSON(stderr, msg)
			LogExecFailure(ExitCodeInternalError, req.Tool, msg)
			return ExitCodeInternalError
		}
	}

	// Enforce raw method execution policy
	isRaw := strings.HasPrefix(req.Tool, "raw.")
	if isRaw && !allowRaw {
		msg := "raw method execution requires --allow-raw flag"
		writeErrorJSON(stderr, msg)
		LogExecFailure(ExitCodeInvalidInput, req.Tool, msg)
		return ExitCodeInvalidInput
	}

	// Step 1: Resolve tool from tooling.json
	tool, err := ResolveTool(projectRoot, req.Tool, isRaw)
	if err != nil {
		writeErrorJSON(stderr, err.Error())
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
		return executeWorkflow(req.Tool, tool.Integration, tool.Mode, req.Args, stdout, stderr, token)
	}

	// Step 2: Load integration bundle (single-method path — no registry calls)
	bundle, err := LoadIntegrationBundle(projectRoot, tool.Integration)
	if err != nil {
		writeErrorJSON(stderr, err.Error())
		LogExecFailure(ExitCodeToolNotFound, req.Tool, err.Error())
		return ExitCodeToolNotFound
	}

	// Step 3: Resolve method from Wreken METHODS section
	method, err := ResolveMethod(bundle, req.Tool)
	if err != nil {
		writeErrorJSON(stderr, err.Error())
		LogExecFailure(ExitCodeToolNotFound, req.Tool, err.Error())
		return ExitCodeToolNotFound
	}

	// Step 4: Get base URL from manifest based on mode
	baseURL, err := GetBaseURL(projectRoot, tool.Integration, tool.Mode)
	if err != nil {
		writeErrorJSON(stderr, err.Error())
		LogExecFailure(ExitCodeInternalError, req.Tool, err.Error())
		return ExitCodeInternalError
	}

	// Step 5: Validate input schema
	if err := ValidateInput(tool, req.Args); err != nil {
		msg := "input validation failed: " + err.Error()
		writeErrorJSON(stderr, msg)
		LogExecFailure(ExitCodeInvalidInput, req.Tool, msg)
		return ExitCodeInvalidInput
	}

	// Step 6: Build HTTP request
	httpReq, err := BuildRequest(method, baseURL, req.Args)
	if err != nil {
		msg := "failed to build request: " + err.Error()
		writeErrorJSON(stderr, msg)
		LogExecFailure(ExitCodeInvalidInput, req.Tool, msg)
		return ExitCodeInvalidInput
	}

	// Step 7: Execute or dry-run
	if dryRun {
		code := ExecuteDryRun(httpReq, stdout)
		if code != ExitCodeOK {
			LogExecFailure(code, req.Tool, "dry-run output failed")
		}
		return code
	}

	// Step 8: Execute HTTP request
	resp, err := ExecuteHTTP(httpReq)
	if err != nil {
		msg := "execution failed: " + err.Error()
		writeErrorJSON(stderr, msg)
		LogExecFailure(ExitCodeSDKFailure, req.Tool, msg)
		return ExitCodeSDKFailure
	}

	// Step 9: Output response (include request URL so caller can verify base URL)
	if rawOutput {
		return OutputRawResponse(resp, httpReq, stdout, stderr)
	}
	return OutputJSONResponse(resp, httpReq, stdout, stderr)
}
