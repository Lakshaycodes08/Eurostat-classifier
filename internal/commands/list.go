// list.go provides shared list command logic for CLI and MCP (local state only).
package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ListResult represents the result of listing local state.
type ListResult struct {
	Methods      []string `json:"methods,omitempty"`
	Workflows    []string `json:"workflows,omitempty"`
	Integrations []string `json:"integrations,omitempty"`
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

	if jsonOutput {
		if err := json.NewEncoder(stdout).Encode(result); err != nil {
			return nil, fmt.Errorf("encode JSON: %w", err)
		}
	} else {
		// Human-readable output
		if filter == "" || filter == "methods" {
			if len(methods) > 0 {
				fmt.Fprintln(stdout, "Methods:")
				for _, m := range methods {
					fmt.Fprintf(stdout, "  %s\n", m)
				}
				fmt.Fprintln(stdout)
			}
		}
		if filter == "" || filter == "workflows" {
			if len(workflows) > 0 {
				fmt.Fprintln(stdout, "Workflows:")
				for _, w := range workflows {
					fmt.Fprintf(stdout, "  %s\n", w)
				}
				fmt.Fprintln(stdout)
			}
		}
		if filter == "" || filter == "integrations" {
			if len(integrations) > 0 {
				fmt.Fprintln(stdout, "Integrations:")
				for _, i := range integrations {
					fmt.Fprintf(stdout, "  %s\n", i)
				}
			}
		}
	}

	return result, nil
}

// getLocalState reads local state from tooling.json and integrations directory.
func getLocalState(projectRoot, filter, prefix string) (methods []string, workflows []string, integrations []string, err error) {
	// Read tooling.json for methods and workflows
	toolingPath := filepath.Join(projectRoot, ".swytchcode", "tooling.json")
	if data, err := os.ReadFile(toolingPath); err == nil {
		var tooling map[string]interface{}
		if err := json.Unmarshal(data, &tooling); err == nil {
			if toolsRaw, ok := tooling["tools"].(map[string]interface{}); ok {
				for canonicalID, toolRaw := range toolsRaw {
					tool, ok := toolRaw.(map[string]interface{})
					if !ok {
						continue
					}

					toolType, _ := tool["type"].(string)
					integration, _ := tool["integration"].(string)

					// Check prefix filter (project name from integration)
					if prefix != "" {
						// Extract project name from integration (e.g., "stripe.stripe@v1" -> "stripe")
						project := extractProjectFromIntegration(integration)
						if project != prefix {
							continue
						}
					}

					if toolType == "method" && (filter == "" || filter == "methods") {
						methods = append(methods, canonicalID)
					} else if toolType == "workflow" && (filter == "" || filter == "workflows") {
						workflows = append(workflows, canonicalID)
					}
				}
			}
		}
	}

	// Read integrations directory for fetched integrations
	if filter == "" || filter == "integrations" {
		integrationsDir := filepath.Join(projectRoot, ".swytchcode", "integrations")
		if entries, err := os.ReadDir(integrationsDir); err == nil {
			integrationMap := make(map[string]bool)
			for _, projectEntry := range entries {
				if !projectEntry.IsDir() {
					continue
				}
				projectName := projectEntry.Name()
				if prefix != "" && projectName != prefix {
					continue
				}

				projectPath := filepath.Join(integrationsDir, projectName)
				if libraryEntries, err := os.ReadDir(projectPath); err == nil {
					for _, libraryEntry := range libraryEntries {
						if !libraryEntry.IsDir() {
							continue
						}
						libraryName := libraryEntry.Name()
						libraryPath := filepath.Join(projectPath, libraryName)
						if versionEntries, err := os.ReadDir(libraryPath); err == nil {
							for _, versionEntry := range versionEntries {
								if !versionEntry.IsDir() {
									continue
								}
								version := versionEntry.Name()
								// Check if wrekenfile exists (valid integration)
								wrekenPath := filepath.Join(libraryPath, version, "wrekenfile.yaml")
								if _, err := os.Stat(wrekenPath); err == nil {
									integration := fmt.Sprintf("%s.%s@%s", projectName, libraryName, version)
									integrationMap[integration] = true
								}
							}
						}
					}
				}
			}
			for integration := range integrationMap {
				integrations = append(integrations, integration)
			}
		}
	}

	return methods, workflows, integrations, nil
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
