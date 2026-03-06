// check.go implements swytchcode check: queries the backend for integration update proposals
// and exits with code 1 if any major (breaking) changes are found.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/auth"
	"gitlab.com/swytchcode/shell/internal/commands"
	"gitlab.com/swytchcode/shell/internal/constants"
	"gitlab.com/swytchcode/shell/internal/telemetry"
)

var checkProject string

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for available integration updates",
	Long: `Queries the Swytchcode backend for integration update proposals detected by the agent pipeline.

Exits with code 1 if any major (breaking) proposals are found, 0 otherwise.

Project UUID resolution order:
  1. --project flag
  2. SWYTCHCODE_PROJECT_UUID environment variable

Optional environment variables:
  SWYTCHCODE_API_URL   Backend base URL (default: https://api-v2.swytchcode.com)
  SWYTCHCODE_TOKEN     Service token for API authentication (agents/CI)

Alternatively, log in with 'swytchcode login' to authenticate as a user.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiURL := os.Getenv("SWYTCHCODE_API_URL")
		if apiURL == "" {
			apiURL = "https://api-v2.swytchcode.com"
		}
		projectUUID, err := auth.ResolveProjectUUID(checkProject)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}

		// Soft-fail auth: if neither service token nor session is available,
		// pass an empty token and let the server return 401 with a clean error.
		token, _, _ := auth.ResolveToken()

		outcome := "success"
		hasBreaking, err := commands.RunCheck(commands.CheckConfig{
			APIURL:      apiURL,
			Token:       token,
			ProjectUUID: projectUUID,
		}, os.Stdout)
		if err != nil {
			outcome = "failure"
			telemetry.Send(apiURL, token, telemetry.Event{
				Command:     "check",
				ProjectUUID: projectUUID,
				Outcome:     outcome,
				CLIVersion:  constants.Version,
			})
			var limitErr *commands.ExecLimitError
			if errors.As(err, &limitErr) {
				fmt.Fprintln(os.Stderr, "Error:", limitErr.Error())
				os.Exit(2)
			}
			// auth / network / server errors → exit 2
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}
		telemetry.Send(apiURL, token, telemetry.Event{
			Command:     "check",
			ProjectUUID: projectUUID,
			Outcome:     outcome,
			CLIVersion:  constants.Version,
		})
		if hasBreaking {
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	checkCmd.Flags().StringVar(&checkProject, "project", "", "Project UUID (overrides SWYTCHCODE_PROJECT_UUID)")
}
