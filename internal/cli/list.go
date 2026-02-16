// list.go implements swytchcode list: lists locally available tools and integrations (no registry calls).
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/commands"
	"gitlab.com/swytchcode/shell/internal/util"
)

var (
	listJSON bool
)

// listCmd implements `swytchcode list` - lists locally available tools and integrations.
var listCmd = &cobra.Command{
	Use:   "list [methods|workflows|integrations] [prefix]",
	Short: "List locally available tools and integrations",
	Long:  "Lists methods, workflows, and integrations that are locally available (from tooling.json and fetched integrations). No registry calls.",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return fmt.Errorf("detect project root: %w", err)
		}

		var filter string // "methods", "workflows", "integrations", or "" for all
		var prefix string // Optional prefix filter

		if len(args) > 0 {
			filter = args[0]
			if len(args) > 1 {
				prefix = args[1]
			}
		}

		_, err = commands.RunList(projectRoot, filter, prefix, listJSON, os.Stdout)
		return err
	},
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "output as JSON object")
}
