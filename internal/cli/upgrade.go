// upgrade.go implements `swytchcode upgrade <library>`: approves a pending proposal.
// Requires user login (Firebase JWT) — service tokens are not accepted because the
// backend records who approved the change.
package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/auth"
	"gitlab.com/swytchcode/shell/internal/commands"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade <library>",
	Short: "Approve a pending integration proposal",
	Long: `Approves the pending integration update proposal for the specified library.

Requires user login (run 'swytchcode login'). Service tokens are not accepted
because the backend records which user approved the change.

Example:
  swytchcode upgrade stripe`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		library := args[0]

		apiURL := os.Getenv("SWYTCHCODE_API_URL")
		if apiURL == "" {
			apiURL = "http://localhost:80"
		}
		projectUUID := os.Getenv("SWYTCHCODE_PROJECT_UUID")
		if projectUUID == "" {
			return fmt.Errorf("SWYTCHCODE_PROJECT_UUID is not set")
		}

		// upgrade requires a user session — no env-var service token fallback.
		session, err := auth.Load()
		if err != nil {
			return fmt.Errorf("not logged in — run `swytchcode login` (service tokens cannot approve upgrades)")
		}
		if session.IsExpired() {
			return fmt.Errorf("session expired — run `swytchcode login`")
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

		return commands.RunUpgrade(commands.UpgradeConfig{
			APIURL:      apiURL,
			Token:       session.AccessToken,
			ProjectUUID: projectUUID,
			Library:     library,
		}, confirm, os.Stdout)
	},
}
