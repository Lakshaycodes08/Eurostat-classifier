// add.go extracts add command logic for MCP use.
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"gitlab.com/swytchcode/shell/internal/util"
)

// RunAdd runs the add command logic.
func RunAdd(ctx context.Context, canonicalID, integrationSpec string, stdout, stderr io.Writer) error {
	projectRoot, err := util.ProjectRoot()
	if err != nil {
		return fmt.Errorf("detect project root: %w", err)
	}

	var project, library, version, toolType string
	mode1 := integrationSpec == ""

	if mode1 {
		// Mode 1: Search all fetched integrations
		matches, err := findToolInAllIntegrations(projectRoot, canonicalID)
		if err != nil {
			return err
		}

		if len(matches) == 0 {
			return fmt.Errorf("canonical ID %q not found in any fetched integrations.\nRun: swytchcode get <project>", canonicalID)
		}

		if len(matches) > 1 {
			// Ambiguity - error in non-interactive mode
			var options []string
			for _, m := range matches {
				options = append(options, fmt.Sprintf("%s@%s.%s (%s)", m.project, m.library, m.version, m.toolType))
			}
			return fmt.Errorf("ambiguous canonical ID. Found in %d integrations:\n  %s\nUse: swytchcode add <integration@version> %s", len(matches), strings.Join(options, "\n  "), canonicalID)
		}

		// Single match
		project = matches[0].project
		library = matches[0].library
		version = matches[0].version
		toolType = matches[0].toolType
	} else {
		// Mode 2: Parse explicit integration spec
		project, library, version = parseIntegrationSpec(integrationSpec)
		if project == "" || library == "" || version == "" {
			return fmt.Errorf("invalid integration spec format: %q (expected: project@library.version)", integrationSpec)
		}

		// Verify integration exists
		integrationsDir := filepath.Join(projectRoot, ".swytchcode", "integrations", project, library, version)
		if _, err := os.Stat(integrationsDir); err != nil {
			return fmt.Errorf("Integration %s not installed. Run: swytchcode get %s", integrationSpec, project)
		}

		// Determine tool type by checking methods.json and workflows.json
		methodsPath := filepath.Join(integrationsDir, "methods.json")
		workflowsPath := filepath.Join(integrationsDir, "workflows.json")

		toolType = ""
		if data, err := os.ReadFile(methodsPath); err == nil {
			var methodsResp map[string]interface{}
			if json.Unmarshal(data, &methodsResp) == nil {
				if methods, ok := methodsResp["methods"].([]interface{}); ok {
					for _, m := range methods {
						if mMap, ok := m.(map[string]interface{}); ok {
							if id, ok := mMap["canonical_id"].(string); ok && id == canonicalID {
								toolType = "method"
								break
							}
						}
					}
				}
			}
		}

		if toolType == "" {
			if data, err := os.ReadFile(workflowsPath); err == nil {
				var workflowsResp map[string]interface{}
				if json.Unmarshal(data, &workflowsResp) == nil {
					if workflows, ok := workflowsResp["workflows"].([]interface{}); ok {
						for _, w := range workflows {
							if wMap, ok := w.(map[string]interface{}); ok {
								if id, ok := wMap["canonical_id"].(string); ok && id == canonicalID {
									toolType = "workflow"
									break
								}
							}
						}
					}
				}
			}
		}

		if toolType == "" {
			return fmt.Errorf("canonical ID %q not found in integration %s", canonicalID, integrationSpec)
		}
	}

	// Read wrekenfile and extract tool details
	wrekenPath := filepath.Join(projectRoot, ".swytchcode", "integrations", project, library, version, "wrekenfile.yaml")
	toolEntry, err := findMethodInWrekenfile(wrekenPath, canonicalID)
	if err != nil {
		return fmt.Errorf("%s %q not found in wrekenfile: %w", toolType, canonicalID, err)
	}

	// Load tooling.json
	toolingPath := filepath.Join(projectRoot, ".swytchcode", "tooling.json")
	data, err := os.ReadFile(toolingPath)
	if err != nil {
		return fmt.Errorf("tooling.json not found; run 'swytchcode init' first: %w", err)
	}

	var tooling map[string]interface{}
	if err := json.Unmarshal(data, &tooling); err != nil {
		return fmt.Errorf("parse tooling.json: %w", err)
	}

	// Ensure tools map exists
	tools, ok := tooling["tools"].(map[string]interface{})
	if !ok {
		tools = make(map[string]interface{})
		tooling["tools"] = tools
	}

	// Extract tool details from wrekenfile entry
	summary := ""
	desc := ""
	var inputs interface{}

	if summaryRaw, ok := toolEntry["SUMMARY"]; ok {
		if summaryStr, ok := summaryRaw.(string); ok {
			summary = summaryStr
		}
	}
	if descRaw, ok := toolEntry["DESC"]; ok {
		if descStr, ok := descRaw.(string); ok {
			desc = descStr
		}
	}
	if inputsRaw, ok := toolEntry["INPUTS"]; ok {
		inputs = inputsRaw
	}

	// Build tool entry
	projectLibrary := fmt.Sprintf("%s.%s", project, library)
	toolEntryMap := map[string]interface{}{
		"summary":     summary,
		"integration": fmt.Sprintf("%s@%s", projectLibrary, version),
		"type":        toolType,
		"desc":        desc,
		"inputs":      inputs,
	}

	tools[canonicalID] = toolEntryMap

	// Ensure integration is tracked in tooling.json (key is project.library)
	integrations, ok := tooling["integrations"].(map[string]interface{})
	if !ok {
		integrations = make(map[string]interface{})
		tooling["integrations"] = integrations
	}

	if _, exists := integrations[projectLibrary]; !exists {
		integrations[projectLibrary] = map[string]interface{}{
			"version": version,
		}
	}

	// Write updated tooling.json
	data, err = json.MarshalIndent(tooling, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tooling.json: %w", err)
	}
	if err := os.WriteFile(toolingPath, data, 0o644); err != nil {
		return fmt.Errorf("write tooling.json: %w", err)
	}

	fmt.Fprintf(stdout, "Added %s %q to tooling.json (integration: %s)\n", toolType, canonicalID, projectLibrary)
	return nil
}

// parseIntegrationSpec parses "project@library.version" and returns (project, library, version).
func parseIntegrationSpec(spec string) (project, library, version string) {
	parts := strings.SplitN(spec, "@", 2)
	if len(parts) != 2 {
		return "", "", ""
	}
	project = strings.TrimSpace(parts[0])
	rest := strings.TrimSpace(parts[1])

	lastDot := strings.LastIndex(rest, ".")
	if lastDot < 0 {
		return "", "", ""
	}
	library = strings.TrimSpace(rest[:lastDot])
	version = strings.TrimSpace(rest[lastDot+1:])

	return project, library, version
}

// toolMatch represents a found tool (method or workflow) in an integration.
type toolMatch struct {
	project     string
	library     string
	version     string
	toolType    string
	canonicalID string
}

// findToolInAllIntegrations searches for a canonical_id across all fetched integrations.
func findToolInAllIntegrations(projectRoot, canonicalID string) ([]toolMatch, error) {
	integrationsDir := filepath.Join(projectRoot, ".swytchcode", "integrations")

	if _, err := os.Stat(integrationsDir); err != nil {
		return nil, fmt.Errorf("no integrations found. Run: swytchcode get <project>")
	}

	var matches []toolMatch

	err := filepath.Walk(integrationsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.Name() == "methods.json" || info.Name() == "workflows.json" {
			relPath, err := filepath.Rel(integrationsDir, path)
			if err != nil {
				return nil
			}

			parts := strings.Split(filepath.Dir(relPath), string(filepath.Separator))
			if len(parts) != 3 {
				return nil
			}

			project := parts[0]
			library := parts[1]
			version := parts[2]

			toolType := "method"
			if info.Name() == "workflows.json" {
				toolType = "workflow"
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			var result map[string]interface{}
			if err := json.Unmarshal(data, &result); err != nil {
				return nil
			}

			var items []interface{}
			if toolType == "method" {
				if methods, ok := result["methods"].([]interface{}); ok {
					items = methods
				}
			} else {
				if workflows, ok := result["workflows"].([]interface{}); ok {
					items = workflows
				}
			}

			for _, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if id, ok := itemMap["canonical_id"].(string); ok && id == canonicalID {
						matches = append(matches, toolMatch{
							project:     project,
							library:     library,
							version:     version,
							toolType:    toolType,
							canonicalID: canonicalID,
						})
						break
					}
				}
			}
		}
		return nil
	})

	return matches, err
}

// findMethodInWrekenfile reads a wrekenfile YAML and finds a method by canonical_id key.
func findMethodInWrekenfile(wrekenPath, canonicalID string) (map[string]interface{}, error) {
	data, err := os.ReadFile(wrekenPath)
	if err != nil {
		return nil, fmt.Errorf("read wrekenfile: %w", err)
	}

	var wreken map[string]interface{}
	if err := yaml.Unmarshal(data, &wreken); err != nil {
		return nil, fmt.Errorf("parse wrekenfile: %w", err)
	}

	methodsRaw, ok := wreken["METHODS"]
	if !ok {
		return nil, fmt.Errorf("METHODS section not found in wrekenfile")
	}

	methods, ok := methodsRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("METHODS must be a map")
	}

	methodEntry, ok := methods[canonicalID]
	if !ok {
		return nil, fmt.Errorf("method %q not found in wrekenfile", canonicalID)
	}

	methodMap, ok := methodEntry.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("method entry must be a map")
	}

	return methodMap, nil
}
