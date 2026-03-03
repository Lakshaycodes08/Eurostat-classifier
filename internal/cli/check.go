// check.go implements swytchcode check: queries the backend for integration update proposals
// and exits with code 1 if any major (breaking) changes are found.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/auth"
	"gitlab.com/swytchcode/shell/internal/commands"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for available integration updates",
	Long: `Queries the Swytchcode backend for integration update proposals detected by the agent pipeline.

Exits with code 1 if any major (breaking) proposals are found, 0 otherwise.

Required environment variables:
  SWYTCHCODE_PROJECT_UUID   Project UUID to check proposals for

Optional environment variables:
  SWYTCHCODE_API_URL        Backend base URL (default: http://localhost:80)
  SWYTCHCODE_TOKEN          Service token for API authentication (agents/CI)

Alternatively, log in with 'swytchcode login' to authenticate as a user.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiURL := os.Getenv("SWYTCHCODE_API_URL")
		if apiURL == "" {
			apiURL = "http://localhost:80"
		}
		projectUUID := os.Getenv("SWYTCHCODE_PROJECT_UUID")
		if projectUUID == "" {
			return fmt.Errorf("SWYTCHCODE_PROJECT_UUID is not set")
		}

		// Soft-fail auth: if neither service token nor session is available,
		// pass an empty token and let the server return 401 with a clean error.
		token, _, _ := auth.ResolveToken()

		hasBreaking, err := commands.RunCheck(commands.CheckConfig{
			APIURL:      apiURL,
			Token:       token,
			ProjectUUID: projectUUID,
		}, os.Stdout)
		if err != nil {
			return err
		}
		if hasBreaking {
			os.Exit(1)
		}
		return nil
	},
}
