package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/util"
)

var (
	listRaw      bool
	listVerified bool
)

// listCmd implements `swytchcode list <library>`.
//
// Purpose: Discover available methods without executing anything.
// Output: JSON to stdout showing verified tools and raw methods.
var listCmd = &cobra.Command{
	Use:   "list <library>",
	Short: "List available verified tools and raw methods for a library",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("library name required")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		library := args[0]

		// TODO:
		// 1. Load tooling.json to get verified tools for this library
		// 2. Load Wrekenfile for the library to get all raw methods
		// 3. Filter based on --raw and --verified flags
		// 4. Output JSON

		// For now, output stub JSON
		output := map[string]interface{}{
			"library":  library,
			"verified": []string{},
			"raw":      []string{},
			"note":     "list command not yet implemented; this is a stub response",
		}

		// Apply filters (when implemented, this will filter the actual lists)
		if listRaw && !listVerified {
			output = map[string]interface{}{
				"raw":  []string{},
				"note": "list command not yet implemented; this is a stub response",
			}
		} else if listVerified && !listRaw {
			output = map[string]interface{}{
				"verified": []string{},
				"note":     "list command not yet implemented; this is a stub response",
			}
		}

		if err := util.WriteJSON(os.Stdout, output); err != nil {
			return fmt.Errorf("write output: %w", err)
		}

		return nil
	},
}

func init() {
	listCmd.Flags().BoolVar(&listRaw, "raw", false, "show only raw methods")
	listCmd.Flags().BoolVar(&listVerified, "verified", false, "show only verified tools")
}
