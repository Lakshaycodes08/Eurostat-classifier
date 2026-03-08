// inspect.go implements `swytchcode inspect <library>`: shows full proposal detail
// for a single library using the dedicated app endpoint.
package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/auth"
	"gitlab.com/swytchcode/shell/internal/commands"
	"gitlab.com/swytchcode/shell/internal/telemetry"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect <library>",
	Short: "Show full proposal detail for a library",
	Long: `Fetches the detailed integration update proposal for the named library.

Requires user login (run 'swytchcode login').

Example:
  swytchcode inspect stripe
  swytchcode inspect stripe.stripe`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		library := args[0]

		apiURL := auth.ResolveAPIURL()

		// inspect requires a user session for the app endpoint.
		session, err := auth.Load()
		if err != nil {
			return fmt.Errorf("not logged in — run `swytchcode login`")
		}
		if err := auth.RefreshIfExpired(session, apiURL); err != nil {
			return err
		}

		start := time.Now()
		// Step 1: find the proposal UUID by library name.
		proposals, err := commands.FetchProposals(commands.CheckConfig{
			APIURL:  apiURL,
			Token:   session.AccessToken,
			Library: library,
		})
		if err != nil {
			opts := &telemetry.EventOpts{DurationMs: time.Since(start).Milliseconds()}
			telemetry.SendEvent(apiURL, session.AccessToken, true, "inspect", library, err, opts)
			return err
		}

		var proposalUUID string
		for _, p := range proposals {
			if strings.EqualFold(p.LibraryName, library) {
				proposalUUID = p.ProposalUUID
				break
			}
		}
		if proposalUUID == "" {
			fmt.Fprintf(os.Stdout, "No proposals found for %s\n", library)
			opts := &telemetry.EventOpts{DurationMs: time.Since(start).Milliseconds()}
			telemetry.SendEvent(apiURL, session.AccessToken, true, "inspect", library, nil, opts)
			return nil
		}

		// Step 2: fetch full detail from the app endpoint.
		detail, err := commands.FetchProposalDetail(apiURL, session.AccessToken, proposalUUID)
		if err != nil {
			opts := &telemetry.EventOpts{DurationMs: time.Since(start).Milliseconds()}
			telemetry.SendEvent(apiURL, session.AccessToken, true, "inspect", library, err, opts)
			return err
		}

		from := detail.CurrentVersion
		if from == "" {
			from = "unknown"
		}
		to := detail.ProposedVersion
		if to == "" {
			to = "unknown"
		}
		header := fmt.Sprintf("%s   %s -> %s   [%s]   confidence: %.2f",
			library, from, to, detail.Impact, detail.Confidence)
		divider := strings.Repeat("─", len(header))
		fmt.Fprintf(os.Stdout, "%s\n%s\n", header, divider)
		if detail.Summary != "" {
			fmt.Fprintf(os.Stdout, "Summary:  %s\n", detail.Summary)
		}
		if detail.ChangeSet != nil {
			fmt.Fprintf(os.Stdout, "Status:   %s\n", detail.Status)
		}

		opts := &telemetry.EventOpts{DurationMs: time.Since(start).Milliseconds()}
		telemetry.SendEvent(apiURL, session.AccessToken, true, "inspect", library, nil, opts)
		return nil
	},
}
