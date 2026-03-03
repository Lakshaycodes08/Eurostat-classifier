// upgrade.go implements the business logic for swytchcode upgrade: approves a pending
// integration proposal for the specified library.
package commands

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"gitlab.com/swytchcode/shell/internal/constants"
)

// UpgradeConfig holds configuration for an upgrade run.
type UpgradeConfig struct {
	APIURL      string // base URL of the backend API
	Token       string // Firebase JWT (user login required — service token not accepted)
	ProjectUUID string // project to check proposals for
	Library     string // library name to upgrade
}

// RunUpgrade finds the pending proposal for Library, confirms with the user,
// and calls PATCH /v2/app/proposals/:uuid/approve on confirmation.
// The confirm function is called with a prompt string and returns true if the
// user confirmed (allows testing without stdin).
func RunUpgrade(cfg UpgradeConfig, confirm func(prompt string) bool, w io.Writer) error {
	proposals, err := FetchProposals(CheckConfig{
		APIURL:      cfg.APIURL,
		Token:       cfg.Token,
		ProjectUUID: cfg.ProjectUUID,
	})
	if err != nil {
		return err
	}

	var found *Proposal
	for i := range proposals {
		if strings.EqualFold(proposals[i].LibraryName, cfg.Library) {
			found = &proposals[i]
			break
		}
	}
	if found == nil {
		fmt.Fprintf(w, "No pending proposal for %s\n", cfg.Library)
		return nil
	}

	from := found.CurrentVersion
	if from == "" {
		from = "unknown"
	}
	to := found.ProposedVersion
	if to == "" {
		to = "unknown"
	}

	var prompt string
	if found.Impact == "major" {
		prompt = fmt.Sprintf(
			"\033[1;31mApprove %s %s → %s [%s]? This is a BREAKING change. (y/N)\033[0m ",
			found.LibraryName, from, to, found.Impact)
	} else {
		prompt = fmt.Sprintf(
			"Approve %s %s → %s [%s]? (y/N) ",
			found.LibraryName, from, to, found.Impact)
	}

	if !confirm(prompt) {
		fmt.Fprintln(w, "Aborted.")
		return nil
	}

	if err := approveProposal(cfg.APIURL, cfg.Token, found.ProposalUUID); err != nil {
		return err
	}

	fmt.Fprintf(w, "Upgrade submitted for %s — spec fetch and library reprocessing started\n", found.LibraryName)
	return nil
}

func approveProposal(apiURL, token, proposalUUID string) error {
	endpoint := fmt.Sprintf("%s/v2/app/proposals/%s/approve", apiURL, proposalUUID)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPatch, endpoint, http.NoBody)
	if err != nil {
		return fmt.Errorf("build approve request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: constants.HTTPClientTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("approve request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("unauthorized — `swytchcode upgrade` requires user login, not a service token")
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("approve returned %d", resp.StatusCode)
	}
	return nil
}
