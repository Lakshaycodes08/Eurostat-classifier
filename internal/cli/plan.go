// plan.go implements swytchcode plan: display workflow steps.
package cli

import (
	"context"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/swytchcode-cli/internal/auth"
	"gitlab.com/swytchcode/swytchcode-cli/internal/commands"
	"gitlab.com/swytchcode/swytchcode-cli/internal/output"
	"gitlab.com/swytchcode/swytchcode-cli/internal/telemetry"
)

var planCmd = &cobra.Command{
	Use:   "plan <canonical_id>",
	Short: "Show the steps for a workflow",
	Long: `Fetches and displays the steps of a workflow by its canonical ID.

The project name is automatically derived from the canonical ID prefix
(e.g. "stripe" from "stripe.charge_customer"), or can be set explicitly.

Examples:
  swytchcode plan stripe.charge_customer
  swytchcode plan order.payment_flow --project mystore
  swytchcode plan stripe.refund --json

Flags:
  -p, --project  Project name (defaults to prefix of canonical_id)
  -j, --json     Output raw JSON instead of human-readable format

Optional environment variables:
  SWYTCHCODE_API_URL   Backend base URL (default: https://api-v2.swytchcode.com)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		canonicalID := args[0]
		project, _ := cmd.Flags().GetString("project")
		jsonOut, _ := cmd.Flags().GetBool("json")

		apiURL := auth.ResolveAPIURL()
		token, fromSession, _ := auth.ResolveToken()

		ctx := context.Background()
		start := time.Now()
		err := commands.RunPlan(ctx, canonicalID, project, jsonOut, os.Stdout, os.Stderr)
		opts := &telemetry.EventOpts{DurationMs: time.Since(start).Milliseconds()}
		telemetry.SendEvent(apiURL, token, fromSession, "plan", canonicalID, err, opts)

		if err != nil {
			output.Error(os.Stderr, err.Error())
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	planCmd.Flags().StringP("project", "p", "", "Project name (defaults to prefix of canonical_id)")
	planCmd.Flags().BoolP("json", "j", false, "Output as JSON")
}
