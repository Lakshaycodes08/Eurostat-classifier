// sync.go implements the sync command (re-fetches workflows/methods from the backend).
package cli

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/cli/internal/commands"
)

var syncCmd = &cobra.Command{
	Use:   "sync [project_name]",
	Short: "Sync workflows and methods from the backend for installed projects",
	Long: `Pulls the latest workflow and method list from the backend and updates local files.

Does NOT modify tooling.json — run 'swytchcode add <canonical_id>' to activate new or updated workflows.

Examples:
  swytchcode sync              # sync all installed projects
  swytchcode sync stripe       # sync only the stripe project`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := ""
		if len(args) == 1 {
			projectName = args[0]
		}
		return commands.RunSync(context.Background(), projectName, os.Stdout, os.Stderr)
	},
}
