// exec.go implements swytchcode exec: the single execution path.
// Supports both CLI args and JSON stdin for maximum flexibility.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/swytchcode-cli/internal/auth"
	"gitlab.com/swytchcode/swytchcode-cli/internal/kernel"
	"gitlab.com/swytchcode/swytchcode-cli/internal/telemetry"
	"gitlab.com/swytchcode/swytchcode-cli/internal/util"
)

// readStdinIfAvailable reads all of stdin. Caller can use the result to merge args when in CLI-args mode.
func readStdinIfAvailable() []byte {
	data, err := io.ReadAll(os.Stdin)
	if err != nil || len(data) == 0 {
		return nil
	}
	return data
}

var (
	execAllowRaw   bool
	execDryRun     bool
	execBodyFile   string
	execInput      []string // key=value pairs
	execParam      []string // query params key=value
	execHeader     []string // header key=value pairs
	execRaw        bool     // output raw HTTP response
	execJSON       bool     // output JSON (default: true for exec; flag for explicit scripting)
	execVerbose    bool     // log request/response details to stderr
	execOutputFile string   // write binary response body to this file
)

// execCmd implements `swytchcode exec`.
//
// It must:
//   - Never prompt.
//   - Never branch on TTY presence (except for help).
//   - Accept both CLI args and JSON stdin.
//   - Emit JSON on stdout/stderr.
//   - Require --allow-raw for raw method execution.
//
// Usage:
//   - CLI args: swytchcode exec <canonical_id> [flags]
//   - JSON stdin: echo '{"tool":"...","args":{...}}' | swytchcode exec
var execCmd = &cobra.Command{
	Use:   "exec [canonical_id]",
	Short: "Execute a tool via the Swytchcode kernel",
	Long: `Execute a tool (method or workflow) via the Swytchcode kernel.

Supports two input modes:
  1. CLI args: swytchcode exec api.cluster.create --body file.json
  2. JSON stdin: echo '{"tool":"api.cluster.create","args":{...}}' | swytchcode exec

The kernel is pure, deterministic, non-interactive, and offline-capable.
It reads only local files (tooling.json, integration bundles) and never calls the registry.`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Allow 0 args (JSON stdin mode) or 1 arg (CLI args mode)
		if len(args) > 1 {
			return cmd.Help()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var exitCode int
		apiURL := auth.ResolveAPIURL()
		token, fromSession, authErr := auth.ResolveToken()
		if token == "" {
			if authErr != nil {
				util.WriteJSON(os.Stderr, map[string]string{"error": authErr.Error()})
				os.Exit(kernel.ExitCodeAuthError)
			}
			telemetry.MaybeHintNoAuth()
		}

		if len(args) == 0 {
			// JSON stdin mode — canonical_id is embedded in the JSON payload, unknown here
			start := time.Now()
			exitCode = kernel.Execute(os.Stdin, os.Stdout, os.Stderr, kernel.ExecOptions{
				AllowRaw: execAllowRaw, DryRun: execDryRun, RawOutput: execRaw, JSONOutput: execJSON,
				Verbose: execVerbose, OutputFile: execOutputFile, Token: token,
			})
			opts := &telemetry.EventOpts{DurationMs: time.Since(start).Milliseconds()}
			// Use synchronous telemetry for exec since the process exits immediately after.
			telemetry.SendEventSync(apiURL, token, fromSession, "exec", "", outcomeErr(exitCode), opts)
		} else {
			// CLI args mode: canonical_id provided as argument
			canonicalID := args[0]

			// Build args map from flags
			argsMap := make(map[string]interface{})

			// Read body from file if provided
			if execBodyFile != "" {
				bodyPath, err := filepath.Abs(execBodyFile)
				if err != nil {
					util.WriteJSON(os.Stderr, map[string]string{"error": "invalid body file path"})
					os.Exit(kernel.ExitCodeInvalidInput)
				}
				bodyData, err := os.ReadFile(bodyPath)
				if err != nil {
					util.WriteJSON(os.Stderr, map[string]string{"error": "failed to read body file: " + err.Error()})
					os.Exit(kernel.ExitCodeInvalidInput)
				}
				var bodyJSON map[string]interface{}
				if err := json.Unmarshal(bodyData, &bodyJSON); err != nil {
					msg := "body file must be valid JSON"
					if hint := util.ExecJSONInvalidHint(bodyData); hint != "" {
						msg += " " + hint
					}
					util.WriteJSON(os.Stderr, map[string]string{"error": msg})
					os.Exit(kernel.ExitCodeInvalidInput)
				}
				argsMap["body"] = bodyJSON
			}

			// Parse --input key=value pairs
			for _, input := range execInput {
				// Parse key=value format
				parts := splitKeyValue(input)
				if len(parts) == 2 {
					argsMap[parts[0]] = parts[1]
				}
			}

			// Parse --param key=value pairs (query params)
			params := make(map[string]string)
			for _, param := range execParam {
				parts := splitKeyValue(param)
				if len(parts) == 2 {
					params[parts[0]] = parts[1]
				}
			}
			if len(params) > 0 {
				argsMap["params"] = params
			}

			// Parse --header key=value pairs (request headers)
			headers := make(map[string]string)
			for _, h := range execHeader {
				parts := splitKeyValue(h)
				if len(parts) == 2 {
					headers[parts[0]] = parts[1]
				}
			}
			if len(headers) > 0 {
				argsMap["headers"] = headers
			}

			// Merge stdin into args when runtime sends JSON (e.g. {"project_name":"swytchcode"} or {"tool":"...","args":{...}}).
			// Only read stdin if it's piped — skip when stdin is a TTY to avoid blocking.
			fi, _ := os.Stdin.Stat()
			if fi.Mode()&os.ModeCharDevice == 0 {
				if stdinBytes := readStdinIfAvailable(); len(stdinBytes) > 0 {
					var stdinObj map[string]interface{}
					if json.Unmarshal(stdinBytes, &stdinObj) == nil {
						if nested, ok := stdinObj["args"].(map[string]interface{}); ok && stdinObj["tool"] != nil {
							for k, v := range nested {
								argsMap[k] = v
							}
						} else {
							for k, v := range stdinObj {
								argsMap[k] = v
							}
						}
					}
				}
			}

			// Create exec request
			req := kernel.ExecRequest{
				Tool: canonicalID,
				Args: argsMap,
			}

			// Convert to JSON and pass to kernel
			reqJSON, err := json.Marshal(req)
			if err != nil {
				util.WriteJSON(os.Stderr, map[string]string{"error": "failed to marshal request"})
				os.Exit(kernel.ExitCodeInternalError)
			}

			// Create a reader from the JSON bytes
			reqReader := util.NewJSONReader(reqJSON)
			start := time.Now()
			exitCode = kernel.Execute(reqReader, os.Stdout, os.Stderr, kernel.ExecOptions{
				AllowRaw: execAllowRaw, DryRun: execDryRun, RawOutput: execRaw, JSONOutput: execJSON,
				Verbose: execVerbose, OutputFile: execOutputFile, Token: token,
			})
			opts := &telemetry.EventOpts{DurationMs: time.Since(start).Milliseconds()}
			// Use synchronous telemetry for exec since the process exits immediately after.
			telemetry.SendEventSync(apiURL, token, fromSession, "exec", canonicalID, outcomeErr(exitCode), opts)
		}

		os.Exit(exitCode)
	},
}


// outcomeErr returns a non-nil error value when exitCode indicates failure,
// so telemetry.SendEvent can record the correct outcome.
func outcomeErr(exitCode int) error {
	if exitCode != kernel.ExitCodeOK {
		return fmt.Errorf("exit code %d", exitCode)
	}
	return nil
}

// splitKeyValue splits "key=value" into ["key", "value"]
func splitKeyValue(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s} // No '=' found, return as single value
}

func init() {
	execCmd.Flags().BoolVar(&execAllowRaw, "allow-raw", false, "allow execution of raw methods (required for tools starting with 'raw.')")
	execCmd.Flags().BoolVar(&execDryRun, "dry-run", false, "show what would be executed without making the HTTP call")
	execCmd.Flags().StringVar(&execBodyFile, "body", "", "path to JSON file containing request body")
	execCmd.Flags().StringArrayVar(&execInput, "input", []string{}, "input key=value pairs (can be specified multiple times)")
	execCmd.Flags().StringArrayVar(&execParam, "param", []string{}, "query parameter key=value pairs (can be specified multiple times)")
	execCmd.Flags().StringArrayVar(&execHeader, "header", []string{}, "request header key=value pairs (can be specified multiple times)")
	execCmd.Flags().BoolVar(&execRaw, "raw", false, "output raw HTTP response instead of normalized JSON")
	execCmd.Flags().BoolVar(&execJSON, "json", false, "output response as JSON (single JSON object to stdout)")
	execCmd.Flags().BoolVar(&execVerbose, "verbose", false, "log request and response details to stderr for debugging (headers are redacted)")
	execCmd.Flags().StringVar(&execOutputFile, "output", "", "write binary response body to this file (required for non-JSON/text responses)")
}
