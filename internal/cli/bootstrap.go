// bootstrap.go implements swytchcode bootstrap: fetches all integrations declared in tooling.json.
package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/swytchcode-cli/internal/commands"
	"gitlab.com/swytchcode/swytchcode-cli/internal/util"
)

// bootstrapCmd implements `swytchcode bootstrap`.
var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Fetch all integrations declared in tooling.json",
	Long: `Fetches all integrations declared in tooling.json that are not already installed.
Reads the integrations section from tooling.json and calls 'swytchcode get' for each missing integration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return fmt.Errorf("detect project root: %w", err)
		}

		ctx := context.Background()
		return commands.RunBootstrap(ctx, projectRoot, os.Stdout, os.Stderr)
	},
}

func init() {
	// bootstrapCmd will be registered in root.go
}
