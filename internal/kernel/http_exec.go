// http_exec.go executes HTTP requests and handles responses.
package kernel

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"gitlab.com/swytchcode/cli/internal/constants"
)

// ExecuteDryRun outputs what would be executed without making the HTTP call.
// Output: method, full url (with query string), all headers (Authorization + from args.headers), and body when present.
func ExecuteDryRun(req *http.Request, stdout io.Writer) int {
	// Headers as flat map[string]string so JSON is clean (spec: "Authorization": "Bearer token123")
	headersMap := make(map[string]string)
	for name, vals := range req.Header {
		if len(vals) > 0 {
			headersMap[name] = vals[0]
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

// OutputRawResponse outputs the raw HTTP response. Returns (exitCode, errMsg); errMsg is non-empty only when exitCode != ExitCodeOK.
func OutputRawResponse(resp *http.Response, req *http.Request, stdout io.Writer, stderr io.Writer) (int, string) {
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
		msg := "failed to read response body: " + err.Error()
		writeErrorJSON(stderr, msg)
		return ExitCodeSDKFailure, msg
	}

	output["body"] = string(bodyBytes)

	if err := json.NewEncoder(stdout).Encode(output); err != nil {
		msg := "failed to encode response"
		writeErrorJSON(stderr, msg)
		return ExitCodeInternalError, msg
	}

	if resp.StatusCode >= 400 {
		return ExitCodeSDKFailure, fmt.Sprintf("HTTP status %d %s", resp.StatusCode, resp.Status)
	}

	return ExitCodeOK, ""
}

// OutputJSONResponse outputs normalized JSON response. Returns (exitCode, errMsg); errMsg is non-empty only when exitCode != ExitCodeOK.
func OutputJSONResponse(resp *http.Response, req *http.Request, stdout io.Writer, stderr io.Writer) (int, string) {
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		msg := "failed to read response body: " + err.Error()
		writeErrorJSON(stderr, msg)
		return ExitCodeSDKFailure, msg
	}

	// Parse JSON response
	var responseJSON interface{}
	if err := json.Unmarshal(bodyBytes, &responseJSON); err != nil {
		msg := "response is not valid JSON"
		writeErrorJSON(stderr, msg)
		return ExitCodeSDKFailure, msg
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
		msg := "failed to encode response"
		writeErrorJSON(stderr, msg)
		return ExitCodeInternalError, msg
	}

	if resp.StatusCode >= 400 {
		return ExitCodeSDKFailure, fmt.Sprintf("HTTP status %d %s", resp.StatusCode, resp.Status)
	}

	return ExitCodeOK, ""
}
