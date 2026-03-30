// http_exec.go executes HTTP requests and handles responses.
package kernel

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gitlab.com/swytchcode/swytchcode-cli/internal/constants"
)

func clipBodyPrefix(body []byte, n int) string {
	if len(body) <= n {
		return string(body)
	}
	return string(body[:n])
}

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
		"url":     req.URL.String(), // full URL with path substituted and query string
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

func readRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}
	b, err := io.ReadAll(req.Body)
	_ = req.Body.Close()
	if err != nil {
		return nil, err
	}
	return b, nil
}

func rebuildHTTPRequest(req *http.Request, ctx context.Context, body []byte) (*http.Request, error) {
	r := req.Clone(ctx)
	r.Body = nil
	r.ContentLength = 0
	if len(body) > 0 {
		r.Body = io.NopCloser(bytes.NewReader(body))
		r.ContentLength = int64(len(body))
	}
	return r, nil
}

func drainAndCloseBody(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

func parseRetryAfter(h http.Header) time.Duration {
	v := strings.TrimSpace(h.Get("Retry-After"))
	if v == "" {
		return 0
	}
	if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
		return time.Duration(sec) * time.Second
	}
	return 0
}

func isRetryableStatus(code int) bool {
	return code == http.StatusTooManyRequests || code == http.StatusServiceUnavailable || code == http.StatusGatewayTimeout
}

func isRetryableNetErr(err error) bool {
	if err == nil {
		return false
	}
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return true
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "connection reset") ||
		strings.Contains(s, "broken pipe") ||
		strings.Contains(s, "eof") ||
		strings.Contains(s, "tls handshake")
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func jitterUpTo(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0
	}
	u := binary.BigEndian.Uint64(buf[:])
	return time.Duration(u % uint64(max))
}

func backoffDuration(policy ResolvedExecutionPolicy, attemptZeroBased int, retryAfter time.Duration) time.Duration {
	exp := attemptZeroBased
	if exp > 4 {
		exp = 4
	}
	shifted := policy.BaseDelay * time.Duration(uint64(1)<<uint(exp))
	const maxCap = 30 * time.Second
	if shifted > maxCap {
		shifted = maxCap
	}
	back := shifted + jitterUpTo(policy.BaseDelay)
	if retryAfter > back {
		return retryAfter
	}
	return back
}

// ExecuteHTTP performs the request with per-attempt timeout, retries (429/503/504/transient network),
// and optional Idempotency-Key when policy.IdempotencyMode is stripe_style.
func ExecuteHTTP(ctx context.Context, req *http.Request, policy ResolvedExecutionPolicy) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	body, err := readRequestBody(req)
	if err != nil {
		return nil, fmt.Errorf("read request body: %w", err)
	}

	idemHeader := strings.TrimSpace(policy.IdempotencyHeader)
	if idemHeader == "" {
		idemHeader = "Idempotency-Key"
	}
	var idemKey string
	if strings.EqualFold(strings.TrimSpace(policy.IdempotencyMode), "stripe_style") {
		if req.Header.Get(idemHeader) == "" {
			idemKey, err = randomHex(16)
			if err != nil {
				return nil, fmt.Errorf("idempotency key: %w", err)
			}
		}
	}

	client := constants.NewHTTPClientNoTimeout()
	var lastRetryAfter time.Duration

	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := backoffDuration(policy, attempt-1, lastRetryAfter)
			lastRetryAfter = 0
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		attemptCtx, cancel := context.WithTimeout(ctx, policy.HTTPTimeout)
		r, err := rebuildHTTPRequest(req, attemptCtx, body)
		if err != nil {
			cancel()
			return nil, err
		}
		if idemKey != "" {
			r.Header.Set(idemHeader, idemKey)
		}

		resp, err := client.Do(r)
		cancel()

		if err != nil {
			if attempt < policy.MaxRetries && isRetryableNetErr(err) {
				continue
			}
			return nil, fmt.Errorf("HTTP request failed: %w", err)
		}

		if isRetryableStatus(resp.StatusCode) {
			lastRetryAfter = parseRetryAfter(resp.Header)
			drainAndCloseBody(resp)
			if attempt == policy.MaxRetries {
				return nil, fmt.Errorf("HTTP %d after %d attempt(s)", resp.StatusCode, attempt+1)
			}
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("HTTP: exhausted retries")
}

// isBinaryContentType returns true when ct is not JSON or text — i.e. the body is binary.
// An empty Content-Type is treated as non-binary (assume JSON/text by default).
func isBinaryContentType(ct string) bool {
	if ct == "" {
		return false
	}
	ct = strings.ToLower(ct)
	return !strings.Contains(ct, "json") && !strings.HasPrefix(ct, "text/")
}

// LogVerboseRequest writes a JSON verbose-request line to w for debugging.
// Sensitive header values are redacted. Body is previewed up to 500 bytes.
// The request body is read and restored so ExecuteHTTP can still use it.
func LogVerboseRequest(w io.Writer, req *http.Request) {
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
	entry := map[string]interface{}{
		"verbose": "request",
		"method":  req.Method,
		"url":     req.URL.String(),
		"headers": headersMap,
	}
	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err == nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes)) // restore for ExecuteHTTP
			if len(bodyBytes) > 0 {
				entry["body_preview"] = clipBodyPrefix(bodyBytes, 500)
			}
		}
	}
	_ = json.NewEncoder(w).Encode(entry)
}

// logVerboseResponse writes a JSON verbose-response line to w for debugging.
// Body bytes are already read by the caller; only the first 500 bytes are previewed.
func logVerboseResponse(w io.Writer, resp *http.Response, bodyBytes []byte) {
	headersMap := make(map[string]string)
	for name, vals := range resp.Header {
		if len(vals) > 0 {
			headersMap[name] = vals[0]
		}
	}
	entry := map[string]interface{}{
		"verbose":     "response",
		"status_code": resp.StatusCode,
		"headers":     headersMap,
	}
	if len(bodyBytes) > 0 {
		entry["body_preview"] = clipBodyPrefix(bodyBytes, 500)
	}
	_ = json.NewEncoder(w).Encode(entry)
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
// When verbose is true, a verbose-response line is written to stderr before output.
// When outputFile is non-empty and the response is binary, bytes are written to that file instead of stdout.
func OutputJSONResponse(resp *http.Response, req *http.Request, stdout io.Writer, stderr io.Writer, verbose bool, outputFile string) int {
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		writeErrorJSON(stderr, "failed to read response body: "+err.Error())
		return ExitCodeSDKFailure
	}

	if verbose {
		logVerboseResponse(stderr, resp, bodyBytes)
	}

	// Binary response path: save to file when --output is set
	ct := resp.Header.Get("Content-Type")
	if isBinaryContentType(ct) {
		if outputFile == "" {
			writeClassifiedError(stderr,
				fmt.Sprintf("binary response (Content-Type: %s) — use --output <file> to save to disk", ct),
				"api_error", false)
			return ExitCodeSDKFailure
		}
		if err := os.WriteFile(outputFile, bodyBytes, 0644); err != nil {
			writeErrorJSON(stderr, "failed to write output file: "+err.Error())
			return ExitCodeInternalError
		}
		output := map[string]interface{}{
			"saved_to":     outputFile,
			"size_bytes":   len(bodyBytes),
			"content_type": ct,
			"status_code":  resp.StatusCode,
		}
		if err := json.NewEncoder(stdout).Encode(output); err != nil {
			writeErrorJSON(stderr, "failed to encode response")
			return ExitCodeInternalError
		}
		return ExitCodeOK
	}

	// Parse JSON response (empty body is valid: treat as {})
	var responseJSON interface{}
	if len(bodyBytes) == 0 {
		responseJSON = map[string]interface{}{}
	} else if err := json.Unmarshal(bodyBytes, &responseJSON); err != nil {
		writeErrorJSON(stderr, "response is not valid JSON: "+clipBodyPrefix(bodyBytes, 200))
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
