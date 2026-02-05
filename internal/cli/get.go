package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/util"
)

var (
	getAutoYes        bool
	getNonInteractive bool
)

// getCmd implements `swytchcode get`.
//
// Interaction rules:
//   - On a TTY without --non-interactive, get MAY prompt (library selection,
//     overwrite confirmation).
//   - In non-interactive mode (--non-interactive or no TTY), it must not
//     prompt and should rely on flags such as --yes.
var getCmd = &cobra.Command{
	Use:   "get [library]",
	Short: "Fetch Wrekenfile for a library",
	Args: func(cmd *cobra.Command, args []string) error {
		interactive := util.IsInteractive() && !getNonInteractive
		if len(args) == 0 && !interactive {
			// Non-interactive usage requires an explicit library name.
			return errors.New("library name required when running non-interactively")
		}
		// When interactive, zero args are allowed; the actual prompt is handled in RunE.
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		interactive := util.IsInteractive() && !getNonInteractive

		var library string
		if len(args) > 0 {
			library = args[0]
		} else if interactive {
			// TODO: Implement interactive selection of library when running on a TTY:
			//   - Present a small curated list (e.g. stripe, openai, slack, ...)
			//   - Or allow freeform entry.
			return errors.New("interactive library selection not yet implemented; rerun with a library argument")
		}

		if library == "" {
			return errors.New("library name required")
		}

		// TODO:
		// 1. Resolve library -> registry endpoint.
		// 2. Fetch Wrekenfile JSON/YAML.
		// 3. Validate schema via internal/wreken.
		// 4. Check if .swytchcode/wrekenfiles/<library>.json exists.
		//    - If interactive and not --non-interactive: prompt before overwrite.
		//    - If non-interactive and overwrite required:
		//        - Overwrite only when --yes is set.
		//        - Otherwise fail without prompting.
		// 5. Write to .swytchcode/wrekenfiles/<library>.json.

		// For now we only stub behavior to make the CLI shape concrete.
		fmt.Printf("Wrekenfile for %s would be fetched and stored (implementation pending)\n", library)
		return nil
	},
}

func init() {
	getCmd.Flags().BoolVar(&getAutoYes, "yes", false, "auto-confirm overwrite in non-interactive mode")
	getCmd.Flags().BoolVar(&getNonInteractive, "non-interactive", false, "disable prompts; suitable for CI")
}

