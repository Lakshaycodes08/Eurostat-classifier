// api.go defines types and HTTP calls for the registry (list integrations, get bundle, list workflows, list methods).
package registry

import (
	"bytes"
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
	Version            string `json:"version"`
	LibraryUUID        string `json:"library_uuid"`
	SandboxEndpoint    string `json:"sandbox_endpoint"`
	ProductionEndpoint string `json:"production_endpoint"`
	Files              struct {
		Wreken struct {
			Format  string `json:"format"`
			Content string `json:"content"` // YAML content as string (base64)
		} `json:"wreken"`
		Manifest struct {
			Format  string                 `json:"format"`
			Content map[string]interface{} `json:"content"`
		} `json:"manifest"`
	} `json:"files"`
}

// IntegrationBundlesResponse represents the response from GET /integrations/{project_name}/bundle.
type IntegrationBundlesResponse struct {
	ProjectName string              `json:"project_name"`
	Bundles     []IntegrationBundle `json:"bundles"`
}

// WorkflowStep represents a step in a workflow (used in workflows.json and downstream).
type WorkflowStep struct {
	Name        string `json:"name"`
	CanonicalID string `json:"canonical_id"`
	LibraryUUID string `json:"library_uuid"`
}

// Workflow represents a workflow from the list (used in workflows.json and downstream).
type Workflow struct {
	Name        string         `json:"name"`
	CanonicalID string         `json:"canonical_id"`
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
	Title       string            `json:"title"`
	CanonicalID string            `json:"canonical_id"`
	Steps       []workflowStepAPI `json:"steps"`
}
type workflowStepAPI struct {
	Description string `json:"description"`
	MethodName  string `json:"method_name"`
	MethodUUID  string `json:"method_uuid"`
	CanonicalID string `json:"canonical_id"`
	LibraryUUID string `json:"library_uuid"`
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

// GetWorkflowBundles fetches all integration bundles required for a specific workflow.
// Route: GET /integrations/{project_name}/workflow/{canonical_id}/bundles
// The backend resolves which libraries the workflow uses and returns only those bundles.
func (c *Client) GetWorkflowBundles(ctx context.Context, projectName, canonicalID string) (*IntegrationBundlesResponse, error) {
	path := fmt.Sprintf("/integrations/%s/workflow/%s/bundles", url.PathEscape(projectName), url.PathEscape(canonicalID))
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("workflow %q not found for project %q", canonicalID, projectName)
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
				LibraryUUID: s.LibraryUUID,
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

// DiscoveryCapability represents a single result from semantic capability discovery.
type DiscoveryCapability struct {
	CanonicalID  string   `json:"canonical_id"`
	Type         string   `json:"type"` // "api" | "sdk" | "workflow"
	Summary      string   `json:"summary"`
	Library      string   `json:"library"`
	LibGroup     string   `json:"lib_group"`
	Version      string   `json:"lib_version"`
	LibraryUUID  string   `json:"library_uuid"`
	LibraryUUIDs []string `json:"library_uuids"`
	Distance     float64  `json:"distance"`
	Tags         []string `json:"tags"`
}

// DiscoverResponse is the response from POST /discover.
type DiscoverResponse struct {
	Capabilities        []DiscoveryCapability `json:"capabilities"`
	RecommendedWorkflow *DiscoveryCapability  `json:"recommended_workflow"`
}

// WorkflowDetail is a single workflow returned by GET /workflows/:canonical_id.
type WorkflowDetail struct {
	Title       string         `json:"title"`
	CanonicalID string         `json:"canonical_id"`
	Steps       []WorkflowStep `json:"steps"`
}

// DiscoverCapabilities calls POST /discover with the given intent.
// projectName and libraryName are optional — pass empty strings to search across all projects/libraries.
func (c *Client) DiscoverCapabilities(ctx context.Context, intent, projectName, libraryName string, topK int) (*DiscoverResponse, error) {
	payload := map[string]interface{}{
		"intent": intent,
		"top_k":  topK,
	}
	if projectName != "" {
		payload["project_name"] = projectName
	}
	if libraryName != "" {
		payload["library_name"] = libraryName
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.Post(ctx, "/discover", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result DiscoverResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// GetWorkflow fetches a single workflow by project name and canonical ID.
// Route: GET /workflows/{canonical_id}?project_name={project_name}
func (c *Client) GetWorkflow(ctx context.Context, projectName, canonicalID string) (*WorkflowDetail, error) {
	path := fmt.Sprintf("/workflows/%s?project_name=%s", url.PathEscape(canonicalID), url.QueryEscape(projectName))
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("workflow %q not found for project %q", canonicalID, projectName)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var raw workflowAPI
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	steps := make([]WorkflowStep, 0, len(raw.Steps))
	for _, s := range raw.Steps {
		name := strings.TrimSpace(s.MethodName)
		if name == "" {
			name = strings.TrimSpace(s.Description)
		}
		if name == "" && s.CanonicalID != "" {
			name = nameFromCanonicalID(s.CanonicalID)
		}
		steps = append(steps, WorkflowStep{Name: name, CanonicalID: s.CanonicalID, LibraryUUID: s.LibraryUUID})
	}

	title := strings.TrimSpace(raw.Title)
	if title == "" && raw.CanonicalID != "" {
		title = nameFromCanonicalID(raw.CanonicalID)
	}

	return &WorkflowDetail{
		Title:       title,
		CanonicalID: raw.CanonicalID,
		Steps:       steps,
	}, nil
}

// VersionMethodsResponse is the response from GET /integrations/{project}/{library}/{version}/methods.
type VersionMethodsResponse struct {
	Methods    []string `json:"methods"`
	Deprecated []string `json:"deprecated"`
}

// GetVersionMethods fetches the list of canonical IDs available in a specific integration version.
// Route: GET /integrations/{project}/{library}/{version}/methods
// Returns nil error and nil response when the endpoint is not yet available (graceful degradation).
func (c *Client) GetVersionMethods(ctx context.Context, project, library, version string) (*VersionMethodsResponse, error) {
	path := fmt.Sprintf("/integrations/%s/%s/%s/methods",
		url.PathEscape(project), url.PathEscape(library), url.PathEscape(version))
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // endpoint or version not found; caller skips validation
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result VersionMethodsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// CanonicalIDResolution is the response from GET /canonical-id/resolve.
type CanonicalIDResolution struct {
	Status          string `json:"status"` // "active" | "renamed" | "removed"
	NewID           string `json:"new_id,omitempty"`
	DeprecatedSince string `json:"deprecated_since,omitempty"`
}

// ResolveCanonicalID checks whether a canonical ID has been renamed or removed.
// Route: GET /canonical-id/resolve?id=<id>
// Returns nil, nil when the endpoint is not yet implemented (graceful degradation).
func (c *Client) ResolveCanonicalID(ctx context.Context, id string) (*CanonicalIDResolution, error) {
	path := fmt.Sprintf("/canonical-id/resolve?id=%s", url.QueryEscape(id))
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // endpoint not implemented yet
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result CanonicalIDResolution
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// InputField describes a single input parameter on a method.
type InputField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

// InputChange describes a change to an existing input field.
type InputChange struct {
	Name    string `json:"name"`
	OldType string `json:"old_type"`
	NewType string `json:"new_type"`
}

// MethodChange describes signature-level changes to a single method.
type MethodChange struct {
	CanonicalID   string        `json:"canonical_id"`
	AddedInputs   []InputField  `json:"added_inputs"`
	RemovedInputs []InputField  `json:"removed_inputs"`
	ChangedInputs []InputChange `json:"changed_inputs"`
}

// ProposalDiff is the response from GET /proposals/diff.
type ProposalDiff struct {
	Library         string         `json:"library"`
	CurrentVersion  string         `json:"current_version"`
	ProposedVersion string         `json:"proposed_version"`
	Added           []string       `json:"added"`
	Removed         []string       `json:"removed"`
	Changed         []MethodChange `json:"changed"`
}

// GetProposalDiff fetches the method-level diff between the installed version and the pending proposal.
// Route: GET /proposals/diff?library=<library>
// Returns nil, nil when no proposal exists or the endpoint is not yet available.
func (c *Client) GetProposalDiff(ctx context.Context, library string) (*ProposalDiff, error) {
	path := fmt.Sprintf("/proposals/diff?library=%s", url.QueryEscape(library))
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // no pending proposal for this library
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result ProposalDiff
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// handleError attempts to parse an error response from the API.
func (c *Client) handleError(resp *http.Response) error {
	var errResp ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Error)
	}
	return fmt.Errorf("API error: HTTP %d", resp.StatusCode)
}
