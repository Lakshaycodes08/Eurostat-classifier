// api.go defines types and HTTP calls for the registry (list integrations, get bundle, list workflows, list methods).
package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Library represents a library within a project.
type Library struct {
	LibraryID      string `json:"library_id"`
	LibraryVersion string `json:"library_version"`
}

// Project represents a project with its libraries.
type Project struct {
	ProjectName string    `json:"project_name"`
	Libraries   []Library `json:"libraries"`
}

// ListIntegrationsResponse is the response from GET /integrations.
type ListIntegrationsResponse struct {
	Projects []Project `json:"projects"`
}

// IntegrationBundle represents a single bundle in the bundles array.
type IntegrationBundle struct {
	Integration        string `json:"integration"`
	Version           string `json:"version"`
	SandboxEndpoint   string `json:"sandbox_endpoint"`
	ProductionEndpoint string `json:"production_endpoint"`
	Files             struct {
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

// IntegrationBundlesResponse represents the response from GET /integrations/{project_name}/bundle.
type IntegrationBundlesResponse struct {
	ProjectName string             `json:"project_name"`
	Bundles     []IntegrationBundle `json:"bundles"`
}

// WorkflowStep represents a step in a workflow (used in workflows.json and downstream).
type WorkflowStep struct {
	Name        string `json:"name"`
	CanonicalID string `json:"canonical_id"`
}

// Workflow represents a workflow from the list (used in workflows.json and downstream).
type Workflow struct {
	Name        string        `json:"name"`
	CanonicalID string        `json:"canonical_id"`
	Steps       []WorkflowStep `json:"steps"`
}

// ListWorkflowsResponse is the response from GET /workflows?project_name=...
type ListWorkflowsResponse struct {
	Workflows []Workflow `json:"workflows"`
}

// listWorkflowsAPI is the raw API response: workflow uses "title", steps use "method_name"/"description".
type listWorkflowsAPI struct {
	Workflows []workflowAPI `json:"workflows"`
}
type workflowAPI struct {
	Title       string          `json:"title"`
	CanonicalID string          `json:"canonical_id"`
	Steps       []workflowStepAPI `json:"steps"`
}
type workflowStepAPI struct {
	Description string `json:"description"`
	MethodName  string `json:"method_name"`
	MethodUUID  string `json:"method_uuid"`
	CanonicalID string `json:"canonical_id"`
}

// FillEmptyWorkflowNames sets Name on any workflow or step that has an empty Name,
// using a fallback derived from CanonicalID (last segment after the final dot, or full id).
// Call this before writing workflows.json so the file always has readable names.
func FillEmptyWorkflowNames(resp *ListWorkflowsResponse) {
	if resp == nil {
		return
	}
	for i := range resp.Workflows {
		w := &resp.Workflows[i]
		if strings.TrimSpace(w.Name) == "" && w.CanonicalID != "" {
			w.Name = nameFromCanonicalID(w.CanonicalID)
		}
		for j := range w.Steps {
			s := &w.Steps[j]
			if strings.TrimSpace(s.Name) == "" && s.CanonicalID != "" {
				s.Name = nameFromCanonicalID(s.CanonicalID)
			}
		}
	}
}

func nameFromCanonicalID(canonicalID string) string {
	canonicalID = strings.TrimSpace(canonicalID)
	if canonicalID == "" {
		return ""
	}
	if i := strings.LastIndex(canonicalID, "."); i >= 0 && i < len(canonicalID)-1 {
		return canonicalID[i+1:]
	}
	return canonicalID
}

// WorkflowDefinition represents a verified workflow definition.
type WorkflowDefinition struct {
	WorkflowID string                 `json:"workflow_id"`
	Title      string                 `json:"title"`
	Version    string                 `json:"version"`
	Steps      []map[string]interface{} `json:"steps"`
}

// Method represents a method from the list.
type Method struct {
	MethodUUID  string `json:"method_uuid"`
	MethodName  string `json:"method_name"`
	CanonicalID string `json:"canonical_id"`
}

// ListMethodsResponse is the response from GET /methods?project_uuid=...
type ListMethodsResponse struct {
	Methods []Method `json:"methods"`
}

// MethodDefinition represents a method definition.
type MethodDefinition struct {
	MethodUUID string                 `json:"method_uuid"`
	LibraryUUID string                `json:"library_uuid"`
	MethodName string                 `json:"method_name"`
	Details    map[string]interface{} `json:"details"` // Can be null
	Tags       []string               `json:"tags"`
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

// GetIntegrationBundles fetches all integration bundles for a project.
// Route: GET /integrations/{project_name}/bundle
func (c *Client) GetIntegrationBundles(ctx context.Context, projectName string) (*IntegrationBundlesResponse, error) {
	path := fmt.Sprintf("/integrations/%s/bundle", url.PathEscape(projectName))
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("project %q not found", projectName)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result IntegrationBundlesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// GetIntegrationBundleVersion finds a specific integration bundle from the bundles response.
// This is a helper that filters GetIntegrationBundles by integration ID and version.
// If exact match fails, tries to match by integration ID only (ignoring version).
// If still no match and there's only one bundle, returns that bundle.
func (c *Client) GetIntegrationBundleVersion(ctx context.Context, projectName, integrationID, version string) (*IntegrationBundle, error) {
	bundlesResp, err := c.GetIntegrationBundles(ctx, projectName)
	if err != nil {
		return nil, err
	}

	if len(bundlesResp.Bundles) == 0 {
		return nil, fmt.Errorf("no bundles found for project %q", projectName)
	}

	// First try exact match: integration ID and version
	for _, bundle := range bundlesResp.Bundles {
		if bundle.Integration == integrationID && bundle.Version == version {
			return &bundle, nil
		}
	}

	// If no exact match, try matching by integration ID only
	for _, bundle := range bundlesResp.Bundles {
		if bundle.Integration == integrationID {
			return &bundle, nil
		}
	}

	// If still no match and there's only one bundle, use it (common case: project_name matches library name)
	if len(bundlesResp.Bundles) == 1 {
		bundle := bundlesResp.Bundles[0]
		// Verify the bundle has basic fields populated
		if bundle.Integration == "" && bundle.Version == "" {
			return nil, fmt.Errorf("bundle in response appears to be empty or malformed")
		}
		return &bundle, nil
	}

	return nil, fmt.Errorf("integration %q version %q not found in project %q (found %d bundles)", integrationID, version, projectName, len(bundlesResp.Bundles))
}

// ListWorkflows fetches verified workflows for a project.
// Route: GET /workflows?project_name=...
// The API returns workflow "title" and step "method_name"/"description"; we map these to Name for workflows.json.
func (c *Client) ListWorkflows(ctx context.Context, projectName string) (*ListWorkflowsResponse, error) {
	path := fmt.Sprintf("/workflows?project_name=%s", url.QueryEscape(projectName))
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var raw listWorkflowsAPI
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	result := &ListWorkflowsResponse{
		Workflows: make([]Workflow, 0, len(raw.Workflows)),
	}
	for _, w := range raw.Workflows {
		steps := make([]WorkflowStep, 0, len(w.Steps))
		for _, s := range w.Steps {
			name := strings.TrimSpace(s.MethodName)
			if name == "" {
				name = strings.TrimSpace(s.Description)
			}
			if name == "" && s.CanonicalID != "" {
				name = nameFromCanonicalID(s.CanonicalID)
			}
			steps = append(steps, WorkflowStep{
				Name:        name,
				CanonicalID: s.CanonicalID,
			})
		}
		name := strings.TrimSpace(w.Title)
		if name == "" && w.CanonicalID != "" {
			name = nameFromCanonicalID(w.CanonicalID)
		}
		result.Workflows = append(result.Workflows, Workflow{
			Name:        name,
			CanonicalID: w.CanonicalID,
			Steps:       steps,
		})
	}
	return result, nil
}

// GetWorkflow fetches a verified workflow definition by workflow_uuid.
// Route: GET /workflows/:workflow_uuid
func (c *Client) GetWorkflow(ctx context.Context, workflowUUID string) (*WorkflowDefinition, error) {
	path := fmt.Sprintf("/workflows/%s", url.PathEscape(workflowUUID))
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("workflow %q not found", workflowUUID)
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

// ListMethods fetches methods for a project.
// Route: GET /methods?project_name=...
func (c *Client) ListMethods(ctx context.Context, projectName string) (*ListMethodsResponse, error) {
	path := fmt.Sprintf("/methods?project_name=%s", url.QueryEscape(projectName))
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result ListMethodsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// GetMethod fetches a method definition by method_uuid.
// Route: GET /methods/:method_uuid
func (c *Client) GetMethod(ctx context.Context, methodUUID string) (*MethodDefinition, error) {
	path := fmt.Sprintf("/methods/%s", url.PathEscape(methodUUID))
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("method %q not found", methodUUID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var method MethodDefinition
	if err := json.NewDecoder(resp.Body).Decode(&method); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &method, nil
}

// handleError attempts to parse an error response from the API.
func (c *Client) handleError(resp *http.Response) error {
	var errResp ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Error)
	}
	return fmt.Errorf("API error: HTTP %d", resp.StatusCode)
}
