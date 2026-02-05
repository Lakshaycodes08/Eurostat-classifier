package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/util"
)

// describeCmd implements `swytchcode describe <tool>`.
//
// Purpose: Inspect a tool or raw method without execution.
// Behavior:
//   - Verified tool → show I/O schema from tooling.json
//   - Raw method → show metadata from Wrekenfile
//   - No SDK calls, no side effects
var describeCmd = &cobra.Command{
	Use:   "describe <tool>",
	Short: "Describe a verified tool or raw method",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("tool name required")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		tool := args[0]
		isRaw := strings.HasPrefix(tool, "raw.")

		// TODO:
		// If isRaw:
		//   1. Parse "raw.library.operation" format
		//   2. Load Wrekenfile for library
		//   3. Extract metadata for the operation
		//   4. Output JSON with metadata
		// Else:
		//   1. Load tooling.json
		//   2. Find tool definition
		//   3. Extract I/O schema
		//   4. Output JSON with schema

		// Stub response
		response := map[string]interface{}{
			"tool":  tool,
			"isRaw": isRaw,
			"note":  "describe command not yet implemented; this is a stub response",
		}

		if isRaw {
			response["source"] = "wrekenfile"
		} else {
			response["source"] = "tooling.json"
		}

		if err := util.WriteJSON(os.Stdout, response); err != nil {
			return fmt.Errorf("write output: %w", err)
		}

		return nil
	},
}
