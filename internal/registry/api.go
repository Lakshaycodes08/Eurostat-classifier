// api.go defines types and HTTP calls for the registry (list integrations, get bundle, list workflows).
package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Integration represents a single integration from the list.
type Integration struct {
	ID                string   `json:"id"`
	Title             string   `json:"title"`
	AuthTypes         []string `json:"auth_types"`
	VerifiedWorkflows bool     `json:"verified_workflows"`
	LatestVersion     string   `json:"latest_version"`
}

// ListIntegrationsResponse is the response from GET /integrations.
type ListIntegrationsResponse struct {
	Integrations []Integration `json:"integrations"`
}

// IntegrationBundle represents the bundle response from GET /integrations/:id/bundle.
type IntegrationBundle struct {
	Integration string `json:"integration"`
	Version     string `json:"version"`
	Files       struct {
		Wreken struct {
			Format  string `json:"format"`
			Content string `json:"content"` // YAML content as string
		} `json:"wreken"`
		Manifest struct {
			Format  string                 `json:"format"`
			Content map[string]interface{} `json:"content"`
		} `json:"manifest"`
	} `json:"files"`
}

// Workflow represents a workflow from the list.
type Workflow struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Risk        struct {
		Writes bool   `json:"writes"`
		Auth   string `json:"auth"`
	} `json:"risk"`
}

// ListWorkflowsResponse is the response from GET /integrations/:id/workflows.
type ListWorkflowsResponse struct {
	Workflows []Workflow `json:"workflows"`
}

// WorkflowDefinition represents a verified workflow definition.
type WorkflowDefinition struct {
	WorkflowID      string                 `json:"workflow_id"`
	Version         string                 `json:"version"`
	ToolingFragment struct {
		Tools map[string]interface{} `json:"tools"`
	} `json:"tooling_fragment"`
}

// ErrorResponse represents an error response from the API.
type ErrorResponse struct {
	Error string `json:"error"`
}

// ListIntegrations fetches the list of available integrations.
func (c *Client) ListIntegrations(ctx context.Context) (*ListIntegrationsResponse, error) {
	resp, err := c.Get(ctx, "/integrations")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result ListIntegrationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// GetIntegrationBundle fetches the integration bundle (Wrekenfile + manifest).
func (c *Client) GetIntegrationBundle(ctx context.Context, integrationID string) (*IntegrationBundle, error) {
	path := fmt.Sprintf("/integrations/%s/bundle", integrationID)
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("integration %q not found", integrationID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var bundle IntegrationBundle
	if err := json.NewDecoder(resp.Body).Decode(&bundle); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &bundle, nil
}

// GetIntegrationBundleVersion fetches the integration bundle at an exact version (for bootstrap).
// Use this for deterministic installs; do not use "latest" at exec or bootstrap.
func (c *Client) GetIntegrationBundleVersion(ctx context.Context, integrationID, version string) (*IntegrationBundle, error) {
	path := fmt.Sprintf("/integrations/%s/bundle", integrationID)
	if version != "" {
		path += "?version=" + url.QueryEscape(version)
	}
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("integration %q version %q not found", integrationID, version)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var bundle IntegrationBundle
	if err := json.NewDecoder(resp.Body).Decode(&bundle); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &bundle, nil
}

// ListWorkflows fetches verified workflows for an integration.
func (c *Client) ListWorkflows(ctx context.Context, integrationID string) (*ListWorkflowsResponse, error) {
	path := fmt.Sprintf("/integrations/%s/workflows", integrationID)
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("integration %q not found", integrationID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result ListWorkflowsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// GetWorkflow fetches a verified workflow definition.
func (c *Client) GetWorkflow(ctx context.Context, workflowID string) (*WorkflowDefinition, error) {
	path := fmt.Sprintf("/workflows/%s", workflowID)
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("workflow %q not found", workflowID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var workflow WorkflowDefinition
	if err := json.NewDecoder(resp.Body).Decode(&workflow); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &workflow, nil
}

// handleError attempts to parse an error response from the API.
func (c *Client) handleError(resp *http.Response) error {
	var errResp ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Error)
	}
	return fmt.Errorf("API error: HTTP %d", resp.StatusCode)
}
