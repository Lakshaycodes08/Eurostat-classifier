// http_exec.go executes HTTP requests and handles responses.
package kernel

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"gitlab.com/swytchcode/shell/internal/constants"
)

// ExecuteDryRun outputs what would be executed without making the HTTP call.
func ExecuteDryRun(req *http.Request, stdout io.Writer) int {
	dryRunOutput := map[string]interface{}{
		"method":  req.Method,
		"url":     req.URL.String(),
		"headers": req.Header,
	}

	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err == nil {
			var bodyJSON interface{}
			if json.Unmarshal(bodyBytes, &bodyJSON) == nil {
				dryRunOutput["body"] = bodyJSON
			} else {
				dryRunOutput["body"] = string(bodyBytes)
			}
		}
	}

	if err := json.NewEncoder(stdout).Encode(dryRunOutput); err != nil {
		writeErrorJSON(stdout, "failed to encode dry-run output")
		return ExitCodeInternalError
	}

	return ExitCodeOK
}

// ExecuteHTTP executes the HTTP request and returns the response.
func ExecuteHTTP(req *http.Request) (*http.Response, error) {
	client := &http.Client{
		Timeout: constants.HTTPClientTimeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	return resp, nil
}

// OutputRawResponse outputs the raw HTTP response.
func OutputRawResponse(resp *http.Response, req *http.Request, stdout io.Writer, stderr io.Writer) int {
	defer resp.Body.Close()

	// Output request URL for verification, then status and headers
	output := map[string]interface{}{
		"request": map[string]string{
			"method": req.Method,
			"url":    req.URL.String(),
		},
		"status_code": resp.StatusCode,
		"status":      resp.Status,
		"headers":    resp.Header,
	}

	// Read body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		writeErrorJSON(stderr, "failed to read response body: "+err.Error())
		return ExitCodeSDKFailure
	}

	output["body"] = string(bodyBytes)

	if err := json.NewEncoder(stdout).Encode(output); err != nil {
		writeErrorJSON(stderr, "failed to encode response")
		return ExitCodeInternalError
	}

	if resp.StatusCode >= 400 {
		return ExitCodeSDKFailure
	}

	return ExitCodeOK
}

// OutputJSONResponse outputs normalized JSON response.
func OutputJSONResponse(resp *http.Response, req *http.Request, stdout io.Writer, stderr io.Writer) int {
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		writeErrorJSON(stderr, "failed to read response body: "+err.Error())
		return ExitCodeSDKFailure
	}

	// Parse JSON response
	var responseJSON interface{}
	if err := json.Unmarshal(bodyBytes, &responseJSON); err != nil {
		// If not JSON, return as-is with error
		writeErrorJSON(stderr, "response is not valid JSON")
		return ExitCodeSDKFailure
	}

	// Output normalized JSON; include request URL so caller can verify base URL was applied
	output := map[string]interface{}{
		"request": map[string]string{
			"method": req.Method,
			"url":    req.URL.String(),
		},
		"status_code": resp.StatusCode,
		"data":        responseJSON,
	}

	if err := json.NewEncoder(stdout).Encode(output); err != nil {
		writeErrorJSON(stderr, "failed to encode response")
		return ExitCodeInternalError
	}

	if resp.StatusCode >= 400 {
		return ExitCodeSDKFailure
	}

	return ExitCodeOK
}
