// inspect.go provides shared inspect logic used by both cli/inspect.go and mcp/tools.go.
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"gitlab.com/swytchcode/swytchcode-cli/internal/constants"
)

// ProposalDetail is the richer response from GET /v2/app/proposals/:uuid.
type ProposalDetail struct {
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

// FetchProposalDetail fetches full proposal details from GET /v2/app/proposals/:uuid.
func FetchProposalDetail(apiURL, token, proposalUUID string) (*ProposalDetail, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		apiURL+"/v2/app/proposals/"+proposalUUID, nil)
	if err != nil {
		return nil, fmt.Errorf("build inspect request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := constants.NewHTTPClient(constants.HTTPClientTimeout)
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
		Proposal ProposalDetail `json:"proposal"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode inspect response: %w", err)
	}
	return &result.Proposal, nil
}
