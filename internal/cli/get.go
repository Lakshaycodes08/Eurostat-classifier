// get.go implements swytchcode get: fetches and installs a Wrekenfile for an integration (exploratory only; does not modify tooling.json).
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/commands"
	"gitlab.com/swytchcode/shell/internal/registry"
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

		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return fmt.Errorf("detect project root: %w", err)
		}
		regClient := registry.NewClient(registry.ConfigFromProjectRoot(projectRoot))
		ctx := context.Background()

		var library string
		if len(args) > 0 {
			library = args[0]
		} else if interactive {
			listResp, err := regClient.ListIntegrations(ctx)
			if err != nil {
				return fmt.Errorf("fetch available integrations: %w", err)
			}
			if len(listResp.Projects) == 0 {
				return errors.New("no integrations available")
			}
			options := make([]string, len(listResp.Projects))
			for i, project := range listResp.Projects {
				options[i] = project.ProjectName
			}
			fmt.Println()
			_, library = util.SelectWithRetry("Which library do you want to add?", options)
		}

		if library == "" {
			return errors.New("library name required")
		}

		return commands.RunGet(ctx, library, getAutoYes, os.Stdout, os.Stderr)
	},
}

func init() {
	getCmd.Flags().BoolVar(&getAutoYes, "yes", false, "auto-confirm overwrite in non-interactive mode")
	getCmd.Flags().BoolVar(&getNonInteractive, "non-interactive", false, "disable prompts; suitable for CI")
}
