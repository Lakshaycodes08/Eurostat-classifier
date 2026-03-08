// add.go provides shared add command logic for CLI and MCP.
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
	"gitlab.com/swytchcode/cli/internal/registry"
	"gitlab.com/swytchcode/cli/internal/util"
)

// ToolMatch represents a found tool (method or workflow) in an integration.
// Used by CLI for interactive disambiguation when a canonical_id appears in multiple integrations.
type ToolMatch struct {
	Project     string
	Library     string
	Version     string
	ToolType    string
	CanonicalID string
}

// FindToolInAllIntegrations searches for a canonical_id across all fetched integrations.
// Returns matches so the caller can disambiguate (e.g. interactive prompt) or pass an explicit integration spec to RunAdd.
func FindToolInAllIntegrations(projectRoot, canonicalID string) ([]ToolMatch, error) {
	integrationsDir := filepath.Join(projectRoot, ".swytchcode", "integrations")

	if _, err := os.Stat(integrationsDir); err != nil {
		return nil, fmt.Errorf("no integrations found. Run: swytchcode get <project>")
	}

	var matches []ToolMatch

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
						matches = append(matches, ToolMatch{
							Project:     project,
							Library:     library,
							Version:     version,
							ToolType:    toolType,
							CanonicalID: canonicalID,
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

// RunAdd runs the add command: add a tool (method or workflow) to tooling.json by canonical_id.
// If integrationSpec is empty, searches all fetched integrations (errors if ambiguous).
// If integrationSpec is set (e.g. "project@library.version"), uses that integration.
func RunAdd(ctx context.Context, canonicalID, integrationSpec string, stdout, stderr io.Writer) error {
	projectRoot, err := util.ProjectRoot()
	if err != nil {
		return fmt.Errorf("detect project root: %w", err)
	}

	var project, library, version, toolType string
	mode1 := integrationSpec == ""

	if mode1 {
		// Mode 1: Search all fetched integrations
		matches, err := FindToolInAllIntegrations(projectRoot, canonicalID)
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
				options = append(options, fmt.Sprintf("%s@%s.%s (%s)", m.Project, m.Library, m.Version, m.ToolType))
			}
			return fmt.Errorf("ambiguous canonical ID. Found in %d integrations:\n  %s\nUse: swytchcode add <integration@version> %s", len(matches), strings.Join(options, "\n  "), canonicalID)
		}

		// Single match
		project = matches[0].Project
		library = matches[0].Library
		version = matches[0].Version
		toolType = matches[0].ToolType
	} else {
		// Mode 2: Parse explicit integration spec
		project, library, version = ParseIntegrationSpec(integrationSpec)
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
		if findToolTypeInFile(methodsPath, "methods", canonicalID) {
			toolType = "method"
		} else if findToolTypeInFile(workflowsPath, "workflows", canonicalID) {
			toolType = "workflow"
		}

		if toolType == "" {
			return fmt.Errorf("canonical ID %q not found in integration %s", canonicalID, integrationSpec)
		}
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

	projectLibrary := fmt.Sprintf("%s.%s", project, library)
	wrekenPath := filepath.Join(projectRoot, ".swytchcode", "integrations", project, library, version, "wrekenfile.yaml")
	wreken, err := LoadWrekenfile(wrekenPath)
	if err != nil {
		return fmt.Errorf("load wrekenfile: %w", err)
	}

	if toolType == "workflow" {
		// Handle workflow: define steps within the workflow entry (no top-level method entries for steps)
		workflowsPath := filepath.Join(projectRoot, ".swytchcode", "integrations", project, library, version, "workflows.json")
		entry, err := findWorkflowEntryInWorkflowsJSON(workflowsPath, canonicalID)
		if err != nil {
			return fmt.Errorf("workflow %q not found in workflows.json: %w", canonicalID, err)
		}
		workflow, err := workflowFromEntry(entry, canonicalID)
		if err != nil {
			return err
		}

		integrationStr := fmt.Sprintf("%s@%s", projectLibrary, version)

		// Build steps as array of step definitions (full method details nested inside workflow)
		stepsArray := make([]interface{}, 0, len(workflow.Steps))
		for index, step := range workflow.Steps {
			methodEntry, err := findMethodInWrekenfile(wrekenPath, step.CanonicalID)
			if err != nil {
				fmt.Fprintf(stderr, "Warning: method %q from workflow step not found in wrekenfile: %v\n", step.CanonicalID, err)
				continue
			}

			summary := ""
			desc := ""
			var inputs interface{}
			if summaryRaw, ok := methodEntry["SUMMARY"]; ok {
				if summaryStr, ok := summaryRaw.(string); ok {
					summary = summaryStr
				}
			}
			if descRaw, ok := methodEntry["DESC"]; ok {
				if descStr, ok := descRaw.(string); ok {
					desc = descStr
				}
			}
			if inputsRaw, ok := methodEntry["INPUTS"]; ok {
				inputs = inputsRaw
				if resolved, err := ResolveInputs(wreken, inputsRaw); err != nil {
					fmt.Fprintf(stderr, "Warning: resolve STRUCTs for step %q: %v (using raw inputs)\n", step.CanonicalID, err)
				} else if resolved != nil {
					inputs = resolved
				}
			}

			stepDef := map[string]interface{}{
				"canonical_id": step.CanonicalID,
				"name":        step.Name,
				"summary":     summary,
				"desc":        desc,
				"inputs":      inputs,
				"integration": integrationStr,
				"index":       index,
			}
			if returnsRaw, ok := methodEntry["RETURNS"]; ok {
				if output, err := ResolveReturns(wreken, returnsRaw); err != nil {
					fmt.Fprintf(stderr, "Warning: resolve STRUCTs for step %q returns: %v (output omitted)\n", step.CanonicalID, err)
				} else if output != nil {
					stepDef["output"] = output
				}
			}
			stepsArray = append(stepsArray, stepDef)
		}

		// Add single workflow entry with steps defined inside it
		workflowEntryMap := map[string]interface{}{
			"name":        workflow.Name,
			"integration": integrationStr,
			"type":        "workflow",
			"steps":       stepsArray,
		}

		tools[canonicalID] = workflowEntryMap
		fmt.Fprintf(stdout, "Added workflow %q with %d step(s) to tooling.json (integration: %s)\n", canonicalID, len(stepsArray), projectLibrary)
	} else {
		// Handle method: read from wrekenfile and add
		toolEntry, err := findMethodInWrekenfile(wrekenPath, canonicalID)
		if err != nil {
			return fmt.Errorf("%s %q not found in wrekenfile: %w", toolType, canonicalID, err)
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
			if resolved, err := ResolveInputs(wreken, inputsRaw); err != nil {
				return fmt.Errorf("resolve STRUCTs in INPUTS: %w", err)
			} else if resolved != nil {
				inputs = resolved
			}
		}

		var output interface{}
		if returnsRaw, ok := toolEntry["RETURNS"]; ok {
			resolved, err := ResolveReturns(wreken, returnsRaw)
			if err != nil {
				return fmt.Errorf("resolve STRUCTs in RETURNS: %w", err)
			}
			output = resolved
		}

		// Build tool entry
		toolEntryMap := map[string]interface{}{
			"summary":     summary,
			"integration": fmt.Sprintf("%s@%s", projectLibrary, version),
			"type":        toolType,
			"desc":        desc,
			"inputs":      inputs,
		}
		if output != nil {
			toolEntryMap["output"] = output
		}

		tools[canonicalID] = toolEntryMap
		fmt.Fprintf(stdout, "Added %s %q to tooling.json (integration: %s)\n", toolType, canonicalID, projectLibrary)
	}

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

	return nil
}

// findToolTypeInFile reports whether canonicalID exists in the named key of a JSON file.
// key is "methods" or "workflows".
func findToolTypeInFile(path, key, canonicalID string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var resp map[string]interface{}
	if json.Unmarshal(data, &resp) != nil {
		return false
	}
	items, ok := resp[key].([]interface{})
	if !ok {
		return false
	}
	for _, item := range items {
		if m, ok := item.(map[string]interface{}); ok {
			if id, ok := m["canonical_id"].(string); ok && id == canonicalID {
				return true
			}
		}
	}
	return false
}

// ParseIntegrationSpec parses "project@library.version" and returns (project, library, version).
func ParseIntegrationSpec(spec string) (project, library, version string) {
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

// workflowFromEntry converts a workflow entry map (from workflows.json) into registry.Workflow.
func workflowFromEntry(wMap map[string]interface{}, canonicalID string) (*registry.Workflow, error) {
	name, _ := wMap["name"].(string)
	stepsRaw, ok := wMap["steps"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("workflow steps must be an array")
	}
	var steps []registry.WorkflowStep
	for _, sRaw := range stepsRaw {
		sMap, ok := sRaw.(map[string]interface{})
		if !ok {
			continue
		}
		stepName, _ := sMap["name"].(string)
		stepID, _ := sMap["canonical_id"].(string)
		steps = append(steps, registry.WorkflowStep{
			Name:        stepName,
			CanonicalID: stepID,
		})
	}
	return &registry.Workflow{
		Name:        name,
		CanonicalID: canonicalID,
		Steps:       steps,
	}, nil
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
