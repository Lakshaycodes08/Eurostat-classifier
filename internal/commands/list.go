// list.go provides shared list command logic for CLI and MCP (local state only).
package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/swytchcode/cli/internal/constants"
	"gitlab.com/swytchcode/cli/internal/util"
)

// ListEntry is a tool (method or workflow) with its integration for identification.
type ListEntry struct {
	CanonicalID  string `json:"canonical_id"`
	Integration  string `json:"integration"` // project.library@version
}

// ListResult represents the result of listing local state.
type ListResult struct {
	Methods      []ListEntry `json:"methods,omitempty"`
	Workflows    []ListEntry `json:"workflows,omitempty"`
	Integrations []string    `json:"integrations,omitempty"`
}

// printEntries writes a slice of ListEntry to stdout, one per line.
func printEntries(stdout io.Writer, entries []ListEntry) {
	for _, e := range entries {
		if e.Integration != "" {
			fmt.Fprintf(stdout, "  %s  %s\n", e.CanonicalID, e.Integration)
		} else {
			fmt.Fprintf(stdout, "  %s\n", e.CanonicalID)
		}
	}
}

// RunList lists locally available tools and integrations (no registry calls).
func RunList(projectRoot, filter, prefix string, jsonOutput bool, stdout io.Writer) (*ListResult, error) {
	methods, workflows, integrations, err := getLocalState(projectRoot, filter, prefix)
	if err != nil {
		return nil, err
	}

	result := &ListResult{}
	if filter == "" || filter == "methods" {
		result.Methods = methods
	}
	if filter == "" || filter == "workflows" {
		result.Workflows = workflows
	}
	if filter == "" || filter == "integrations" {
		result.Integrations = integrations
	}
	if filter == "tooling" {
		result.Methods = methods
		result.Workflows = workflows
	}

	if jsonOutput {
		if err := json.NewEncoder(stdout).Encode(result); err != nil {
			return nil, fmt.Errorf("encode JSON: %w", err)
		}
	} else {
		// Human-readable output: show canonical_id and project.library@version for identification
		if filter == "" || filter == "methods" {
			fmt.Fprintln(stdout, "Methods:")
			printEntries(stdout, methods)
			fmt.Fprintln(stdout)
		}
		if filter == "" || filter == "workflows" {
			fmt.Fprintln(stdout, "Workflows:")
			printEntries(stdout, workflows)
			fmt.Fprintln(stdout)
		}
		if filter == "" || filter == "integrations" {
			fmt.Fprintln(stdout, "Integrations:")
			for _, i := range integrations {
				fmt.Fprintf(stdout, "  %s\n", i)
			}
		}
		if filter == "tooling" {
			fmt.Fprintln(stdout, "Enabled Methods (tooling.json):")
			printEntries(stdout, methods)
			fmt.Fprintln(stdout)
			fmt.Fprintln(stdout, "Enabled Workflows (tooling.json):")
			printEntries(stdout, workflows)
		}
	}

	return result, nil
}

// getLocalState reads local state by scanning .swytchcode/integrations recursively.
// Methods and workflows are discovered from methods.json and workflows.json in each integration.
// When filter is "tooling", reads from tooling.json instead to show what has been enabled via swytchcode add.
// prefix is used as a filter pattern: match canonical_id or project name (case-insensitive substring/project match).
func getLocalState(projectRoot, filter, prefix string) (methods []ListEntry, workflows []ListEntry, integrations []string, err error) {
	// Tooling filter: read from tooling.json to show what the user has explicitly added.
	if filter == "tooling" {
		toolingPath := util.Join(projectRoot, constants.SwytchDirName, constants.ToolingJSONFile)
		data, readErr := os.ReadFile(toolingPath)
		if readErr != nil {
			return nil, nil, nil, fmt.Errorf("tooling.json not found: run 'swytchcode init' first")
		}
		var tooling map[string]interface{}
		if jsonErr := json.Unmarshal(data, &tooling); jsonErr != nil {
			return nil, nil, nil, fmt.Errorf("failed to parse tooling.json: %w", jsonErr)
		}
		tools, _ := tooling["tools"].(map[string]interface{})
		for canonicalID, entryRaw := range tools {
			entry, _ := entryRaw.(map[string]interface{})
			integration, _ := entry["integration"].(string)
			toolType, _ := entry["type"].(string)
			if !matchesPattern(canonicalID, integration, "", prefix) {
				continue
			}
			if toolType == "workflow" {
				workflows = append(workflows, ListEntry{CanonicalID: canonicalID, Integration: integration})
			} else {
				methods = append(methods, ListEntry{CanonicalID: canonicalID, Integration: integration})
			}
		}
		return methods, workflows, nil, nil
	}

	integrationsDir := util.IntegrationsDir(projectRoot)
	if _, statErr := os.Stat(integrationsDir); statErr != nil {
		if filter == "methods" || filter == "workflows" {
			return nil, nil, nil, fmt.Errorf("integrations directory not found at %s: run 'swytchcode get <project>' first", integrationsDir)
		}
		return nil, nil, nil, nil
	}

	// Walk project/library/version and collect methods, workflows, and integration names
	integrationSet := make(map[string]bool)
	// workflowIntegrations deduplicates workflows by canonical_id across libraries.
	// Value is the ordered list of integrations the workflow was found in.
	workflowIntegrations := make(map[string][]string)

	projectEntries, _ := os.ReadDir(integrationsDir)
	for _, projectEntry := range projectEntries {
		if !projectEntry.IsDir() {
			continue
		}
		projectName := projectEntry.Name()
		projectPath := filepath.Join(integrationsDir, projectName)

		libraryEntries, _ := os.ReadDir(projectPath)
		for _, libraryEntry := range libraryEntries {
			if !libraryEntry.IsDir() {
				continue
			}
			libraryName := libraryEntry.Name()
			libraryPath := filepath.Join(projectPath, libraryName)

			versionEntries, _ := os.ReadDir(libraryPath)
			for _, versionEntry := range versionEntries {
				if !versionEntry.IsDir() {
					continue
				}
				version := versionEntry.Name()
				versionPath := filepath.Join(libraryPath, version)
				wrekenPath := filepath.Join(versionPath, constants.WrekenfileYAMLFile)
				if _, err := os.Stat(wrekenPath); err != nil {
					continue
				}
				integration := fmt.Sprintf("%s.%s@%s", projectName, libraryName, version)
				integrationSet[integration] = true

				// Methods from methods.json
				if filter == "" || filter == "methods" {
					methodsPath := filepath.Join(versionPath, constants.MethodsJSONFile)
					if data, readErr := os.ReadFile(methodsPath); readErr == nil {
						var out map[string]interface{}
						if json.Unmarshal(data, &out) == nil {
							if methodsList, ok := out["methods"].([]interface{}); ok {
								for _, mRaw := range methodsList {
									mMap, ok := mRaw.(map[string]interface{})
									if !ok {
										continue
									}
									canonicalID, _ := mMap["canonical_id"].(string)
									if canonicalID == "" {
										continue
									}
									if !matchesPattern(canonicalID, integration, projectName, prefix) {
										continue
									}
									methods = append(methods, ListEntry{CanonicalID: canonicalID, Integration: integration})
								}
							}
						}
					}
				}

				// Workflows from workflows.json
				if filter == "" || filter == "workflows" {
					workflowsPath := filepath.Join(versionPath, constants.WorkflowsJSONFile)
					if data, readErr := os.ReadFile(workflowsPath); readErr == nil {
						var out map[string]interface{}
						if json.Unmarshal(data, &out) == nil {
							if workflowsList, ok := out["workflows"].([]interface{}); ok {
								for _, wRaw := range workflowsList {
									wMap, ok := wRaw.(map[string]interface{})
									if !ok {
										continue
									}
									canonicalID, _ := wMap["canonical_id"].(string)
									if canonicalID == "" {
										continue
									}
									if !matchesPattern(canonicalID, integration, projectName, prefix) {
										continue
									}
									workflowIntegrations[canonicalID] = append(workflowIntegrations[canonicalID], integration)
								}
							}
						}
					}
				}
			}
		}
	}

	// Convert deduplicated workflow map to []ListEntry
	for canonicalID, integs := range workflowIntegrations {
		integration := integs[0]
		if len(integs) > 1 {
			integration = fmt.Sprintf("%s +%d more", integs[0], len(integs)-1)
		}
		workflows = append(workflows, ListEntry{CanonicalID: canonicalID, Integration: integration})
	}

	// Build integrations list (optionally filtered by prefix as project name)
	if filter == "" || filter == "integrations" {
		for integration := range integrationSet {
			if prefix != "" {
				project := extractProjectFromIntegration(integration)
				if !strings.EqualFold(project, prefix) && !strings.Contains(strings.ToLower(integration), strings.ToLower(prefix)) {
					continue
				}
			}
			integrations = append(integrations, integration)
		}
	}

	return methods, workflows, integrations, nil
}

// matchesPattern returns true if pattern is empty, or if canonical_id or project name matches (case-insensitive).
func matchesPattern(canonicalID, integration, projectName, pattern string) bool {
	if pattern == "" {
		return true
	}
	pattern = strings.ToLower(pattern)
	if strings.Contains(strings.ToLower(canonicalID), pattern) {
		return true
	}
	if strings.Contains(strings.ToLower(projectName), pattern) {
		return true
	}
	if strings.Contains(strings.ToLower(integration), pattern) {
		return true
	}
	return false
}

// extractProjectFromIntegration extracts project name from integration string (e.g., "stripe.stripe@v1" -> "stripe").
func extractProjectFromIntegration(integration string) string {
	parts := strings.Split(integration, "@")
	if len(parts) == 0 {
		return ""
	}
	projectLibrary := parts[0]
	projectParts := strings.Split(projectLibrary, ".")
	if len(projectParts) == 0 {
		return ""
	}
	return projectParts[0]
}
