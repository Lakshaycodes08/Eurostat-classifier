// http_exec.go executes HTTP requests and handles responses.
package kernel

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"gitlab.com/swytchcode/cli/internal/constants"
)

// sensitiveHeaders contains header names whose values are redacted in dry-run output.
var sensitiveHeaders = map[string]bool{
	"Authorization": true,
	"X-Api-Key":     true,
}

// ExecuteDryRun outputs what would be executed without making the HTTP call.
// Output: method, full url (with query string), all headers (with sensitive values redacted), and body when present.
func ExecuteDryRun(req *http.Request, stdout io.Writer) int {
	// Headers as flat map[string]string so JSON is clean. Sensitive headers are redacted
	// so that piped/logged dry-run output does not leak credentials.
	headersMap := make(map[string]string)
	for name, vals := range req.Header {
		if len(vals) > 0 {
			if sensitiveHeaders[name] {
				headersMap[name] = "[REDACTED]"
			} else {
				headersMap[name] = vals[0]
			}
		}
	}
	dryRunOutput := map[string]interface{}{
		"method":  req.Method,
		"url":    req.URL.String(), // full URL with path substituted and query string
		"headers": headersMap,
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
	client := constants.NewHTTPClient(constants.HTTPClientTimeout)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	return resp, nil
}

// OutputRawResponse outputs the raw HTTP response. Error details are written to stderr; returns exit code only.
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
		"headers":     resp.Header,
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

	return ExitCodeOK
}

// OutputJSONResponse outputs normalized JSON response. Error details are written to stderr; returns exit code only.
// API-level errors (HTTP 4xx/5xx) exit 0 — status_code in the JSON conveys success/failure.
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

	return ExitCodeOK
}
