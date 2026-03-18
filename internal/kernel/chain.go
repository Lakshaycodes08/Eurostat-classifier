// chain.go orchestrates sequential multi-step workflow execution.
// Each step's output is merged into the next step's input args.
package kernel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"gitlab.com/swytchcode/cli/internal/registry"
)

// StepResult holds the outcome of a single workflow step execution.
type StepResult struct {
	StepIndex     int
	StepName      string
	LibraryUUID   string
	Output        map[string]interface{} // object fields for step chaining (empty for array/scalar responses)
	RawOutput     interface{}            // actual parsed response value (array, object, scalar, or nil)
	StatusCode    int
	Error         error
	RequestMethod string // HTTP method of the step request (e.g. "POST")
	RequestURL    string // Full URL of the step request
}

// WorkflowError is returned when a step fails, carrying partial progress.
type WorkflowError struct {
	FailedStep     int
	StepName       string
	Err            error
	CompletedSteps []int
}

func (e *WorkflowError) Error() string {
	return fmt.Sprintf("workflow failed at step %d (%s): %v", e.FailedStep+1, e.StepName, e.Err)
}

// RunWorkflow executes all steps of a workflow sequentially.
// Each step's HTTP response body is merged into the accumulated args for subsequent steps.
// On step failure, returns a partial []StepResult and a *WorkflowError.
func RunWorkflow(
	_ context.Context,
	workflow *registry.WorkflowDetail,
	bundleMap BundleMap,
	initialInputs map[string]interface{},
	mode string,
	out, errOut io.Writer,
) ([]StepResult, error) {
	results := make([]StepResult, 0, len(workflow.Steps))
	completedIndexes := make([]int, 0)

	// Accumulate merged args across steps: start with caller's initial inputs.
	mergedArgs := make(map[string]interface{}, len(initialInputs))
	for k, v := range initialInputs {
		mergedArgs[k] = v
	}

	for i, step := range workflow.Steps {
		stepName := step.Name
		if stepName == "" {
			stepName = step.CanonicalID
		}

		fmt.Fprintf(errOut, "  [%d/%d] %s", i+1, len(workflow.Steps), stepName)

		// Resolve the bundle for this step
		bundle := resolveStepBundle(step, bundleMap)
		if bundle == nil {
			err := fmt.Errorf("no bundle found for step %q (library_uuid=%q)", stepName, step.LibraryUUID)
			fmt.Fprintf(errOut, " ✗ %v\n", err)
			return results, &WorkflowError{
				FailedStep:     i,
				StepName:       stepName,
				Err:            err,
				CompletedSteps: completedIndexes,
			}
		}

		// Resolve the method from the bundle
		method, err := ResolveMethod(bundle, step.CanonicalID)
		if err != nil {
			fmt.Fprintf(errOut, " ✗ method not found: %v\n", err)
			return results, &WorkflowError{
				FailedStep:     i,
				StepName:       stepName,
				Err:            err,
				CompletedSteps: completedIndexes,
			}
		}

		// Get base URL from the bundle's endpoints
		baseURL := bundle.SandboxEndpoint
		if mode == "production" {
			baseURL = bundle.ProductionEndpoint
		}
		if baseURL == "" {
			err := fmt.Errorf("no base URL for bundle %q (mode=%s)", bundle.Library, mode)
			fmt.Fprintf(errOut, " ✗ %v\n", err)
			return results, &WorkflowError{
				FailedStep:     i,
				StepName:       stepName,
				Err:            err,
				CompletedSteps: completedIndexes,
			}
		}

		// Build and execute the HTTP request
		httpReq, err := BuildRequest(method, baseURL, mergedArgs)
		if err != nil {
			fmt.Fprintf(errOut, " ✗ build request: %v\n", err)
			return results, &WorkflowError{
				FailedStep:     i,
				StepName:       stepName,
				Err:            fmt.Errorf("build request: %w", err),
				CompletedSteps: completedIndexes,
			}
		}

		resp, err := ExecuteHTTP(httpReq)
		if err != nil {
			fmt.Fprintf(errOut, " ✗ %v\n", err)
			return results, &WorkflowError{
				FailedStep:     i,
				StepName:       stepName,
				Err:            err,
				CompletedSteps: completedIndexes,
			}
		}

		// Parse response body
		stepOutput, rawOutput, statusCode, execErr := parseStepResponse(resp)
		result := StepResult{
			StepIndex:     i,
			StepName:      stepName,
			LibraryUUID:   step.LibraryUUID,
			Output:        stepOutput,
			RawOutput:     rawOutput,
			StatusCode:    statusCode,
			Error:         execErr,
			RequestMethod: httpReq.Method,
			RequestURL:    httpReq.URL.String(),
		}
		results = append(results, result)

		if execErr != nil {
			fmt.Fprintf(errOut, " ✗ HTTP %d\n", statusCode)
			return results, &WorkflowError{
				FailedStep:     i,
				StepName:       stepName,
				Err:            execErr,
				CompletedSteps: completedIndexes,
			}
		}

		fmt.Fprintf(errOut, " ✓ (HTTP %d)\n", statusCode)
		completedIndexes = append(completedIndexes, i)

		// Merge this step's output into accumulated args for subsequent steps
		for k, v := range stepOutput {
			mergedArgs[k] = v
		}
	}

	return results, nil
}

// resolveStepBundle finds the correct bundle for a step.
// Uses step.LibraryUUID as the key; falls back to the single bundle if the map has only one entry.
func resolveStepBundle(step registry.WorkflowStep, bundleMap BundleMap) *IntegrationBundle {
	if step.LibraryUUID != "" {
		if b, ok := bundleMap[step.LibraryUUID]; ok {
			return b
		}
	}
	// Fallback: if only one bundle, use it (single-library workflow)
	if len(bundleMap) == 1 {
		for _, b := range bundleMap {
			return b
		}
	}
	return nil
}

// parseStepResponse reads the HTTP response and returns:
//   - mergeMap: object fields for chaining into the next step's args (empty map for array/scalar responses)
//   - rawOutput: the actual parsed JSON value (array, object, or nil) for display in workflow output
//   - statusCode: the HTTP status code
//   - err: non-nil when status >= 400
func parseStepResponse(resp *http.Response) (mergeMap map[string]interface{}, rawOutput interface{}, statusCode int, err error) {
	defer resp.Body.Close()
	statusCode = resp.StatusCode

	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, nil, statusCode, fmt.Errorf("read response body: %w", readErr)
	}

	mergeMap = map[string]interface{}{}
	if len(bodyBytes) > 0 {
		var parsed interface{}
		if jsonErr := json.Unmarshal(bodyBytes, &parsed); jsonErr == nil {
			rawOutput = parsed
			// Only merge into next step args when the response is a JSON object
			if m, ok := parsed.(map[string]interface{}); ok {
				mergeMap = m
			}
			// Arrays and scalars are preserved in RawOutput but not merged into args
		} else {
			// Non-JSON body — wrap as raw string; don't leak into next step's query params
			rawStr := string(bodyBytes)
			rawOutput = rawStr
		}
	}

	if statusCode >= 400 {
		return mergeMap, rawOutput, statusCode, fmt.Errorf("HTTP %d: %s", statusCode, http.StatusText(statusCode))
	}

	return mergeMap, rawOutput, statusCode, nil
}

// PrintWorkflowError writes a human-readable failure summary to errOut.
func PrintWorkflowError(errOut io.Writer, wfErr *WorkflowError) {
	fmt.Fprintf(errOut, "\nWorkflow failed at step %d: %s\n", wfErr.FailedStep+1, wfErr.StepName)
	fmt.Fprintf(errOut, "Error: %v\n", wfErr.Err)
	if len(wfErr.CompletedSteps) > 0 {
		fmt.Fprintf(errOut, "Completed steps:")
		for _, idx := range wfErr.CompletedSteps {
			fmt.Fprintf(errOut, " %d", idx+1)
		}
		fmt.Fprintln(errOut)
	}
	if wfErr.FailedStep > len(wfErr.CompletedSteps) {
		fmt.Fprintf(errOut, "Steps not reached: %d onwards\n", wfErr.FailedStep+2)
	}
}
