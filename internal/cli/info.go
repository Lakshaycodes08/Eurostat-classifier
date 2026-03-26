// info.go implements swytchcode info: displays information about a tool by canonical_id.
package cli

import (
	"context"
	"errors"
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/swytchcode-cli/internal/commands"
)

var (
	infoJSON bool
)

// infoCmd implements `swytchcode info`.
var infoCmd = &cobra.Command{
	Use:   "info <canonical_id>",
	Short: "Show information about a tool by canonical ID",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("canonical_id required")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		canonicalID := args[0]

		toolInfos, err := commands.RunInfo(context.Background(), canonicalID, os.Stdout, os.Stderr)
		if err != nil {
			return err
		}

		return commands.FormatInfoOutput(toolInfos, infoJSON, os.Stdout)
	},
}

func init() {
	infoCmd.Flags().BoolVar(&infoJSON, "json", false, "output as JSON")
}
