// demo.go implements demo-mode execution for `swytchcode exec --demo`.
// No local project setup, no API keys, no auth required.
// Calls the swytchcode demo API endpoint and returns a formatted real response.
package kernel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"gitlab.com/swytchcode/swytchcode-cli/internal/auth"
	"gitlab.com/swytchcode/swytchcode-cli/internal/constants"
	"gitlab.com/swytchcode/swytchcode-cli/internal/util"
)

// executeDemoMode calls the swytchcode demo API endpoint.
// It bypasses all local file lookups — works on a fresh machine with zero setup.
// If jsonOutput is true, raw JSON is written to out; otherwise a human-readable
// summary is printed (optimised for terminals and videos).
func executeDemoMode(tool string, args map[string]interface{}, out, errOut io.Writer, jsonOutput bool) int {
	apiURL := auth.ResolveAPIURL()

	// Progress indicator — only when stderr is a TTY (terminal/video, not CI pipe)
	if isTTY(os.Stderr) {
		fmt.Fprintf(errOut, "Executing %s...\n", tool)
	}

	payload := map[string]interface{}{
		"tool": tool,
		"args": args,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		util.WriteJSON(errOut, map[string]string{"error": "failed to encode demo request: " + err.Error()})
		return ExitCodeInternalError
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.HTTPClientTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL+constants.DemoExecPath, bytes.NewReader(body))
	if err != nil {
		util.WriteJSON(errOut, map[string]string{"error": "failed to build demo request: " + err.Error()})
		return ExitCodeInternalError
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Use NewHTTPClient so SWYTCHCODE_INSECURE is respected (e.g. local dev with self-signed certs).
	resp, err := constants.NewHTTPClient(constants.HTTPClientTimeout).Do(httpReq)
	if err != nil {
		fmt.Fprintf(errOut, "\nDemo service temporarily unavailable.\n\nTry again or set up your own integration:%s", constants.DemoInitHint)
		return ExitCodeSDKFailure
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		util.WriteJSON(errOut, map[string]string{"error": "failed to read demo response: " + err.Error()})
		return ExitCodeInternalError
	}

	if resp.StatusCode >= 400 {
		if json.Valid(respBody) {
			var apiErr map[string]interface{}
			if json.Unmarshal(respBody, &apiErr) == nil {
				if msg, ok := apiErr["error"].(string); ok {
					fmt.Fprintf(errOut, "\nDemo unavailable for %s: %s%s", tool, msg, constants.DemoInitHint)
					return ExitCodeSDKFailure
				}
			}
		}
		fmt.Fprintf(errOut, "\nDemo unavailable for %s (HTTP %d).%s", tool, resp.StatusCode, constants.DemoInitHint)
		return ExitCodeSDKFailure
	}

	if jsonOutput {
		fmt.Fprintf(out, "%s\n", respBody)
	} else {
		printDemoResponse(out, tool, respBody)
	}

	fmt.Fprint(errOut, constants.DemoUpgradeHint)
	return ExitCodeOK
}

// printDemoResponse renders the demo API response in a human-readable format.
// Expected response shape: {"summary": "...", "data": {...}}
// Falls back to flat key→value rendering for any valid JSON object.
func printDemoResponse(out io.Writer, tool string, raw []byte) {
	var envelope struct {
		Summary string                 `json:"summary"`
		Data    map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(raw, &envelope); err != nil || (envelope.Summary == "" && envelope.Data == nil) {
		var flat map[string]interface{}
		if json.Unmarshal(raw, &flat) == nil {
			fmt.Fprintf(out, "\n✔  %s (demo)\n\n", tool)
			printFields(out, flat)
		} else {
			fmt.Fprintf(out, "%s\n", raw)
		}
		return
	}

	summary := envelope.Summary
	if summary == "" {
		summary = tool
	}
	fmt.Fprintf(out, "\n✔  %s (demo)\n\n", summary)
	printFields(out, envelope.Data)
}

// printFields renders a map as `→ key: value` lines, skipping nil values.
func printFields(out io.Writer, fields map[string]interface{}) {
	for k, v := range fields {
		if v == nil {
			continue
		}
		switch val := v.(type) {
		case map[string]interface{}, []interface{}:
			compact, err := json.Marshal(val)
			if err == nil {
				fmt.Fprintf(out, "→ %s: %s\n", k, compact)
			}
		default:
			fmt.Fprintf(out, "→ %s: %v\n", k, val)
		}
	}
}

// isTTY reports whether w is a terminal (character device).
func isTTY(w *os.File) bool {
	stat, err := w.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
