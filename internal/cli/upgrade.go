// upgrade.go implements `swytchcode upgrade <library>`: approves a pending proposal.
// Requires user login (Firebase JWT) — service tokens are not accepted because the
// backend records who approved the change.
package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/auth"
	"gitlab.com/swytchcode/shell/internal/commands"
	"gitlab.com/swytchcode/shell/internal/telemetry"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade <library>",
	Short: "Approve a pending integration proposal",
	Long: `Approves the pending integration update proposal for the specified library.

Requires user login (run 'swytchcode login'). Service tokens are not accepted
because the backend records which user approved the change.

Example:
  swytchcode upgrade stripe
  swytchcode upgrade stripe.stripe`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		library := args[0]

		apiURL := auth.ResolveAPIURL()

		// upgrade requires a user session — no env-var service token fallback.
		session, err := auth.Load()
		if err != nil {
			return fmt.Errorf("not logged in — run `swytchcode login` (service tokens cannot approve upgrades)")
		}
		if err := auth.RefreshIfExpired(session, apiURL); err != nil {
			return err
		}

		confirm := func(prompt string) bool {
			fmt.Fprint(os.Stdout, prompt)
			scanner := bufio.NewScanner(os.Stdin)
			if !scanner.Scan() {
				return false
			}
			answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
			return answer == "y" || answer == "yes"
		}

		start := time.Now()
		err = commands.RunUpgrade(commands.UpgradeConfig{
			APIURL:  apiURL,
			Token:   session.AccessToken,
			Library: library,
		}, confirm, os.Stdout)
		opts := &telemetry.EventOpts{DurationMs: time.Since(start).Milliseconds()}
		telemetry.SendEvent(apiURL, session.AccessToken, true, "proposals_approve", library, err, opts)
		return err
	},
}
