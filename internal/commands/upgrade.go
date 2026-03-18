// upgrade.go implements the business logic for swytchcode upgrade: approves a pending
// integration proposal for the specified library.
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"gitlab.com/swytchcode/cli/internal/constants"
	"gitlab.com/swytchcode/cli/internal/util"
)

// UpgradeConfig holds configuration for an upgrade run.
type UpgradeConfig struct {
	APIURL      string    // base URL of the backend API
	Token       string    // Firebase JWT (user login required — service token not accepted)
	Library     string    // project.library name to upgrade
	Apply       bool      // if true, auto-run get + re-add all affected tools after approval
	ProjectRoot string    // path to project root (optional; detected automatically when empty)
	Stderr      io.Writer // stderr for apply sub-commands (falls back to stdout writer when nil)
}

// RunUpgrade finds the pending proposal for Library, confirms with the user,
// and calls PATCH /v2/app/proposals/:uuid/approve on confirmation.
// The confirm function is called with a prompt string and returns true if the
// user confirmed (allows testing without stdin).
func RunUpgrade(cfg UpgradeConfig, confirm func(prompt string) bool, w io.Writer) error {
	proposals, err := FetchProposals(CheckConfig{
		APIURL:  cfg.APIURL,
		Token:   cfg.Token,
		Library: cfg.Library,
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

	if cfg.Apply {
		stderr := cfg.Stderr
		if stderr == nil {
			stderr = w
		}
		if err := runApplyAfterUpgrade(context.Background(), cfg, found.LibraryName, w, stderr); err != nil {
			fmt.Fprintf(w, "⚠ apply failed: %v\n", err)
		}
	}
	return nil
}

// runApplyAfterUpgrade re-downloads the project bundle and re-adds all affected tools in tooling.json.
func runApplyAfterUpgrade(ctx context.Context, cfg UpgradeConfig, libraryName string, w, stderr io.Writer) error {
	projectRoot := cfg.ProjectRoot
	if projectRoot == "" {
		var err error
		projectRoot, err = util.ProjectRoot()
		if err != nil {
			return fmt.Errorf("detect project root: %w", err)
		}
	}

	projectName := strings.SplitN(libraryName, ".", 2)[0]
	fmt.Fprintf(w, "Refreshing bundles for project %q...\n", projectName)
	if err := RunGet(ctx, projectName, true, w, stderr); err != nil {
		return fmt.Errorf("get %s: %w", projectName, err)
	}

	// Re-add all tools in tooling.json whose integration belongs to this library.
	toolingPath := util.ToolingPath(projectRoot)
	data, err := os.ReadFile(toolingPath)
	if err != nil {
		return fmt.Errorf("read tooling.json: %w", err)
	}
	var tooling map[string]interface{}
	if err := json.Unmarshal(data, &tooling); err != nil {
		return fmt.Errorf("parse tooling.json: %w", err)
	}
	tools, _ := tooling["tools"].(map[string]interface{})

	prefix := libraryName + "@"
	var refreshed []string
	for canonicalID, raw := range tools {
		entry, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		integration, _ := entry["integration"].(string)
		if strings.HasPrefix(integration, prefix) {
			if err := RunAdd(ctx, canonicalID, "", false, w, stderr); err != nil {
				fmt.Fprintf(w, "  ⚠ re-add %s: %v\n", canonicalID, err)
			} else {
				refreshed = append(refreshed, canonicalID)
			}
		}
	}

	if len(refreshed) > 0 {
		fmt.Fprintf(w, "Re-added %d tool(s): %s\n", len(refreshed), strings.Join(refreshed, ", "))
	} else {
		fmt.Fprintf(w, "No tools for %s found in tooling.json to re-add.\n", libraryName)
	}
	return nil
}

func approveProposal(apiURL, token, proposalUUID string) error {
	endpoint := fmt.Sprintf("%s/v2/app/proposals/%s/approve", apiURL, proposalUUID)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPatch, endpoint, http.NoBody)
	if err != nil {
		return fmt.Errorf("build approve request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := constants.NewHTTPClient(constants.HTTPClientTimeout)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("approve request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("unauthorized — `swytchcode upgrade` requires user login, not a service token")
	}
	if resp.StatusCode == http.StatusBadRequest {
		return fmt.Errorf("proposal is not pending approval — it may already be approved or applied")
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("approve returned %d", resp.StatusCode)
	}
	return nil
}
