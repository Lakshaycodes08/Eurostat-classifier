// discover.go implements swytchcode discover: semantic capability discovery.
package cli

import (
	"context"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/cli/internal/auth"
	"gitlab.com/swytchcode/cli/internal/commands"
	"gitlab.com/swytchcode/cli/internal/output"
	"gitlab.com/swytchcode/cli/internal/telemetry"
)

var discoverCmd = &cobra.Command{
	Use:   "discover <intent>",
	Short: "Find API capabilities matching a natural language intent",
	Long: `Searches for API methods and workflows that match a natural language description.

Examples:
  swytchcode discover "charge a customer"
  swytchcode discover "send a welcome email" --project sendgrid
  swytchcode discover "create payment" --top 10 --json

Flags:
  -p, --project  Scope the search to a specific project (default: all projects)
  -n, --top      Number of results to return (default: 5)
  -j, --json     Output raw JSON instead of human-readable format

Optional environment variables:
  SWYTCHCODE_API_URL   Backend base URL (default: https://api-v2.swytchcode.com)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		intent := args[0]
		project, _ := cmd.Flags().GetString("project")
		library, _ := cmd.Flags().GetString("library")
		topK, _ := cmd.Flags().GetInt("top")
		jsonOut, _ := cmd.Flags().GetBool("json")

		apiURL := auth.ResolveAPIURL()
		token, fromSession, _ := auth.ResolveToken()

		ctx := context.Background()
		start := time.Now()
		err := commands.RunDiscover(ctx, intent, project, library, topK, jsonOut, os.Stdout, os.Stderr)
		opts := &telemetry.EventOpts{DurationMs: time.Since(start).Milliseconds()}
		telemetry.SendEvent(apiURL, token, fromSession, "discover", intent, err, opts)

		if err != nil {
			output.Error(os.Stderr, err.Error())
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	discoverCmd.Flags().StringP("project", "p", "", "Scope search to a specific project")
	discoverCmd.Flags().StringP("library", "l", "", "Scope search to a specific library within a project")
	discoverCmd.Flags().IntP("top", "n", 5, "Number of results to return")
	discoverCmd.Flags().BoolP("json", "j", false, "Output as JSON")
}
