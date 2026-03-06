// inspect.go implements `swytchcode inspect <library>`: shows full proposal detail
// for a single library using the dedicated app endpoint.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/auth"
	"gitlab.com/swytchcode/shell/internal/commands"
	"gitlab.com/swytchcode/shell/internal/constants"
	"gitlab.com/swytchcode/shell/internal/telemetry"
)

// proposalDetail is the richer response from GET /v2/app/proposals/:uuid.
// Note: library_name is NOT returned by this endpoint (no JOIN to libraries table).
// Use the library name from Step 1 (FetchProposals) instead.
type proposalDetail struct {
	IntegrationProposalUUID string  `json:"integration_proposal_uuid"`
	CurrentVersion          string  `json:"current_version"`
	ProposedVersion         string  `json:"proposed_version"`
	Impact                  string  `json:"impact"`
	Confidence              float64 `json:"confidence"`
	Status                  string  `json:"status"`
	Summary                 string  `json:"summary"`
	ChangeSet               any     `json:"change_set"`
	SourcesEvidence         any     `json:"sources_evidence"`
	SpecPatch               any     `json:"spec_patch"`
}

var inspectProject string

var inspectCmd = &cobra.Command{
	Use:   "inspect <library>",
	Short: "Show full proposal detail for a library",
	Long: `Fetches the detailed integration update proposal for the named library.

Requires user login (run 'swytchcode login').

Example:
  swytchcode inspect stripe`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		library := args[0]

		apiURL := os.Getenv("SWYTCHCODE_API_URL")
		if apiURL == "" {
			apiURL = "https://api-v2.swytchcode.com"
		}
		projectUUID, err := auth.ResolveProjectUUID(inspectProject)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}

		// inspect requires a user session for the app endpoint.
		session, err := auth.Load()
		if err != nil {
			return fmt.Errorf("not logged in — run `swytchcode login`")
		}
		if err := auth.RefreshIfExpired(session, apiURL); err != nil {
			return err
		}

		// Step 1: find the proposal UUID by library name.
		proposals, err := commands.FetchProposals(commands.CheckConfig{
			APIURL:      apiURL,
			Token:       session.AccessToken,
			ProjectUUID: projectUUID,
		})
		if err != nil {
			telemetry.Send(apiURL, session.AccessToken, telemetry.Event{
				Command: "inspect", ProjectUUID: projectUUID, LibraryName: library,
				Outcome: "failure", CLIVersion: constants.Version,
			})
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
			telemetry.Send(apiURL, session.AccessToken, telemetry.Event{
				Command: "inspect", ProjectUUID: projectUUID, LibraryName: library,
				Outcome: "success", CLIVersion: constants.Version,
			})
			return nil
		}

		// Step 2: fetch full detail from the app endpoint.
		detail, err := fetchProposalDetail(apiURL, session.AccessToken, proposalUUID)
		if err != nil {
			telemetry.Send(apiURL, session.AccessToken, telemetry.Event{
				Command: "inspect", ProjectUUID: projectUUID, LibraryName: library,
				Outcome: "failure", CLIVersion: constants.Version,
			})
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

		telemetry.Send(apiURL, session.AccessToken, telemetry.Event{
			Command: "inspect", ProjectUUID: projectUUID, LibraryName: library,
			Outcome: "success", CLIVersion: constants.Version,
		})
		return nil
	},
}

func init() {
	inspectCmd.Flags().StringVar(&inspectProject, "project", "", "Project UUID (overrides SWYTCHCODE_PROJECT_UUID)")
}

func fetchProposalDetail(apiURL, token, proposalUUID string) (*proposalDetail, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		apiURL+"/v2/app/proposals/"+proposalUUID, nil)
	if err != nil {
		return nil, fmt.Errorf("build inspect request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("inspect request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized — run `swytchcode login`")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned %d", resp.StatusCode)
	}

	var result struct {
		Success  bool           `json:"success"`
		Proposal proposalDetail `json:"proposal"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode inspect response: %w", err)
	}
	return &result.Proposal, nil
}
