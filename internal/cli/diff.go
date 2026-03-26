// diff.go implements `swytchcode diff <library>`: shows what would change in a pending upgrade.
package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/swytchcode-cli/internal/auth"
	"gitlab.com/swytchcode/swytchcode-cli/internal/commands"
	"gitlab.com/swytchcode/swytchcode-cli/internal/telemetry"
)

var diffCmd = &cobra.Command{
	Use:   "diff <library>",
	Short: "Show method-level changes in a pending integration upgrade",
	Long: `Displays a human-readable diff of what would change in a pending TinyFish proposal.
Shows added, removed, and changed methods and their input signatures.

Requires login or SWYTCHCODE_TOKEN.

Examples:
  swytchcode diff stripe
  swytchcode diff stripe.payments`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		library := args[0]

		apiURL := auth.ResolveAPIURL()
		token, fromSession, err := auth.ResolveToken()
		if err != nil || token == "" {
			return fmt.Errorf("not authenticated — run `swytchcode login` or set SWYTCHCODE_TOKEN")
		}

		ctx := context.Background()
		start := time.Now()
		runErr := commands.RunDiff(ctx, library, token, os.Stdout, os.Stderr)
		opts := &telemetry.EventOpts{DurationMs: time.Since(start).Milliseconds()}
		telemetry.SendEvent(apiURL, token, fromSession, "diff", library, runErr, opts)
		return runErr
	},
}
