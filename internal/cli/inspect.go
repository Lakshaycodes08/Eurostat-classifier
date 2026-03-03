// inspect.go implements `swytchcode inspect <library>`: shows full proposal detail
// for a single library.
package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/auth"
	"gitlab.com/swytchcode/shell/internal/commands"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect <library>",
	Short: "Show full proposal detail for a library",
	Long: `Fetches integration update proposals and shows the full detail for
the specified library (summary, versions, impact, confidence).

Example:
  swytchcode inspect stripe`,
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

		token, _, _ := auth.ResolveToken()

		proposals, err := commands.FetchProposals(commands.CheckConfig{
			APIURL:      apiURL,
			Token:       token,
			ProjectUUID: projectUUID,
		})
		if err != nil {
			return err
		}

		for _, p := range proposals {
			if !strings.EqualFold(p.LibraryName, library) {
				continue
			}

			from := p.CurrentVersion
			if from == "" {
				from = "unknown"
			}
			to := p.ProposedVersion
			if to == "" {
				to = "unknown"
			}

			header := fmt.Sprintf("%s   %s -> %s   [%s]   confidence: %.2f",
				p.LibraryName, from, to, p.Impact, p.Confidence)
			divider := strings.Repeat("─", len(header))

			fmt.Fprintf(os.Stdout, "%s\n%s\n", header, divider)
			if p.Summary != "" {
				fmt.Fprintf(os.Stdout, "Summary:  %s\n", p.Summary)
			}
			return nil
		}

		fmt.Fprintf(os.Stdout, "No proposals found for %s\n", library)
		return nil
	},
}
