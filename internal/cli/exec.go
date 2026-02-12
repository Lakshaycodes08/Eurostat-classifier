// exec.go implements swytchcode exec: the single execution path.
// Supports both CLI args and JSON stdin for maximum flexibility.
package cli

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/kernel"
	"gitlab.com/swytchcode/shell/internal/util"
)

var (
	execAllowRaw bool
	execDryRun   bool
	execBodyFile string
	execInput    []string // key=value pairs
	execParam    []string // query params key=value
	execRaw      bool     // output raw HTTP response
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

		if len(args) == 0 {
			// JSON stdin mode
			exitCode = kernel.Execute(os.Stdin, os.Stdout, os.Stderr, execAllowRaw, execDryRun, execRaw, "")
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
					util.WriteJSON(os.Stderr, map[string]string{"error": "body file must be valid JSON"})
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
			reqReader := &jsonReader{data: reqJSON}
			exitCode = kernel.Execute(reqReader, os.Stdout, os.Stderr, execAllowRaw, execDryRun, execRaw, "")
		}

		os.Exit(exitCode)
	},
}

// jsonReader implements io.Reader for JSON data
type jsonReader struct {
	data []byte
	pos  int
}

func (r *jsonReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	if r.pos >= len(r.data) {
		err = io.EOF
	}
	return n, err
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
	execCmd.Flags().BoolVar(&execRaw, "raw", false, "output raw HTTP response instead of normalized JSON")
}
