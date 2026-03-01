// check.go implements the business logic for swytchcode check: fetches integration update
// proposals from the backend API and reports any breaking changes.
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"gitlab.com/swytchcode/shell/internal/constants"
)

// ANSI color codes for terminal output.
const (
	colorReset = "\033[0m"
	colorRed   = "\033[1;31m" // bold red — breaking changes
	colorGreen = "\033[32m"   // green    — all up to date
)

// Proposal represents a single integration update proposal returned by the backend.
type Proposal struct {
	ProposalUUID    string  `json:"integration_proposal_uuid"`
	LibraryName     string  `json:"library_name"`
	CurrentVersion  string  `json:"current_version"`
	ProposedVersion string  `json:"proposed_version"`
	Impact          string  `json:"impact"`    // "breaking" | "non_breaking" | "unknown"
	Status          string  `json:"status"`    // "pending_approval" | "applied"
	Confidence      float64 `json:"confidence"`
	Summary         string  `json:"summary"`
}

// CheckResponse is the response envelope from GET /v2/shell/proposals/check.
type CheckResponse struct {
	Success   bool       `json:"success"`
	Proposals []Proposal `json:"proposals"`
}

// CheckConfig holds the configuration for a check run.
type CheckConfig struct {
	APIURL      string // base URL of the backend API
	Token       string // bearer token (INTERNAL_AGENT_TOKEN)
	ProjectUUID string // project to query proposals for
}

// RunCheck calls the backend proposals endpoint, prints any pending/applied proposals,
// and returns hasBreaking=true if any proposal has impact "breaking".
func RunCheck(cfg CheckConfig, w io.Writer) (hasBreaking bool, err error) {
	endpoint, err := url.Parse(cfg.APIURL)
	if err != nil {
		return false, fmt.Errorf("invalid SWYTCHCODE_API_URL %q: %w", cfg.APIURL, err)
	}
	endpoint.Path = "/v2/shell/proposals/check"
	q := endpoint.Query()
	q.Set("project_uuid", cfg.ProjectUUID)
	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return false, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Token)
	}

	httpClient := &http.Client{Timeout: constants.HTTPClientTimeout}
	resp, err := httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return false, fmt.Errorf("unauthorized: check SWYTCHCODE_TOKEN")
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("API returned %d", resp.StatusCode)
	}

	var result CheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Proposals) == 0 {
		fmt.Fprintln(w, colorGreen+"All integrations up to date"+colorReset)
		return false, nil
	}

	for _, p := range result.Proposals {
		from := p.CurrentVersion
		if from == "" {
			from = "unknown"
		}
		to := p.ProposedVersion
		if to == "" {
			to = "unknown"
		}
		if p.Impact == "breaking" {
			fmt.Fprintf(w, colorRed+"[!] %-12s %s -> %s   (%s)"+colorReset+"\n", p.LibraryName, from, to, p.Impact)
			hasBreaking = true
		} else {
			fmt.Fprintf(w, "[!] %-12s %s -> %s   (%s)\n", p.LibraryName, from, to, p.Impact)
		}
	}

	return hasBreaking, nil
}
