// check.go implements swytchcode check: queries the backend for integration update proposals
// and exits with code 1 if any major (breaking) changes are found.
package cli

import (
	"errors"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/cli/internal/auth"
	"gitlab.com/swytchcode/cli/internal/commands"
	"gitlab.com/swytchcode/cli/internal/output"
	"gitlab.com/swytchcode/cli/internal/telemetry"
)

var checkCmd = &cobra.Command{
	Use:   "check [project_or_library]",
	Short: "Check for available integration updates",
	Long: `Queries the Swytchcode backend for integration update proposals detected by the agent pipeline.

Exits with code 1 if any major (breaking) proposals are found, 0 otherwise.

Optionally filter by project name or project.library:
  swytchcode check                  # all proposals for the authed user
  swytchcode check weaviate         # filter by project name
  swytchcode check weaviate.lyrid   # filter by project.library

Optional environment variables:
  SWYTCHCODE_API_URL   Backend base URL (default: https://api-v2.swytchcode.com)
  SWYTCHCODE_TOKEN     Service token for API authentication (agents/CI)

Alternatively, log in with 'swytchcode login' to authenticate as a user.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiURL := auth.ResolveAPIURL()

		var library string
		if len(args) == 1 {
			library = args[0]
		}

		// Soft-fail auth: if neither service token nor session is available,
		// pass an empty token and let the server return 401 with a clean error.
		token, fromSession, _ := auth.ResolveToken()
		if token == "" {
			telemetry.MaybeHintNoAuth()
		}

		start := time.Now()
		hasBreaking, err := commands.RunCheck(commands.CheckConfig{
			APIURL:  apiURL,
			Token:   token,
			Library: library,
		}, os.Stdout)
		opts := &telemetry.EventOpts{DurationMs: time.Since(start).Milliseconds()}
		telemetry.SendEvent(apiURL, token, fromSession, "proposals_check", library, err, opts)
		if err != nil {
			var limitErr *commands.ExecLimitError
			if errors.As(err, &limitErr) {
				output.Error(os.Stderr, limitErr.Error())
				os.Exit(2)
			}
			// auth / network / server errors → exit 2
			output.Error(os.Stderr, err.Error())
			output.Hint(os.Stderr, "run 'swytchcode login' or set SWYTCHCODE_TOKEN")
			os.Exit(2)
		}
		if hasBreaking {
			os.Exit(1)
		}
		return nil
	},
}
