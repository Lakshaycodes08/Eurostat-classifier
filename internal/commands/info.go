// info.go provides shared info command logic for CLI and MCP.
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"gitlab.com/swytchcode/cli/internal/constants"
	"gitlab.com/swytchcode/cli/internal/output"
	"gitlab.com/swytchcode/cli/internal/util"
)

// ToolInfo represents information about a tool (method or workflow).
type ToolInfo struct {
	CanonicalID  string                 `json:"canonical_id"`
	Type         string                 `json:"type"` // "method" or "workflow"
	Project      string                 `json:"project"`
	Library      string                 `json:"library"`
	Version      string                 `json:"version"`
	Integration  string                 `json:"integration"` // "project.library@version"
	Summary      string                 `json:"summary,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Inputs       interface{}            `json:"inputs,omitempty"`  // Resolved to scalars when STRUCTS available
	Output       interface{}            `json:"output,omitempty"`   // Resolved return schema when RETURNS/STRUCTS available
	Wrekenfile   map[string]interface{} `json:"wrekenfile,omitempty"` // Full entry from wrekenfile
}

// RunInfo runs the info command: search for a canonical_id and return its information.
// Returns all matches found across integrations.
func RunInfo(ctx context.Context, canonicalID string, stdout, stderr io.Writer) ([]ToolInfo, error) {
	projectRoot, err := util.ProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("detect project root: %w", err)
	}

	// Search for the tool in all integrations
	matches, err := FindToolInAllIntegrations(projectRoot, canonicalID)
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("canonical ID %q not found in any fetched integrations", canonicalID)
	}

	// For each match, get tool details from wrekenfile (methods) or workflows.json (workflows)
	var toolInfos []ToolInfo
	for _, match := range matches {
		integrationsBase := util.IntegrationVersionDir(projectRoot, match.Project, match.Library, match.Version)
		var toolEntry map[string]interface{}
		var err error

		if match.ToolType == "workflow" {
			workflowsPath := filepath.Join(integrationsBase, constants.WorkflowsJSONFile)
			toolEntry, err = findWorkflowEntryInWorkflowsJSON(workflowsPath, canonicalID)
		} else {
			wrekenPath := filepath.Join(integrationsBase, constants.WrekenfileYAMLFile)
			toolEntry, err = findToolInWrekenfile(wrekenPath, canonicalID, match.ToolType)
		}

		if err != nil {
			output.Warn(stderr, fmt.Sprintf("failed to read tool data for %s.%s@%s: %v", match.Project, match.Library, match.Version, err))
			continue
		}

		// Extract common fields (wrekenfile uses SUMMARY/DESC/INPUTS; workflows.json uses summary/desc/steps)
		summary := ""
		for _, key := range []string{constants.WrekenSummary, "summary"} {
			if summaryRaw, ok := toolEntry[key]; ok {
				if summaryStr, ok := summaryRaw.(string); ok {
					summary = summaryStr
					break
				}
			}
		}

		description := ""
		for _, key := range []string{constants.WrekenDesc, "desc", "description"} {
			if descRaw, ok := toolEntry[key]; ok {
				if descStr, ok := descRaw.(string); ok {
					description = descStr
					break
				}
			}
		}

		var inputs interface{}
		var toolOutput interface{}
		if match.ToolType == "method" {
			wrekenPath := filepath.Join(integrationsBase, constants.WrekenfileYAMLFile)
			wreken, loadErr := LoadWrekenfile(wrekenPath)
			if loadErr == nil {
				if inputsRaw, ok := toolEntry[constants.WrekenInputs]; ok {
					if resolved, err := ResolveInputs(wreken, inputsRaw); err != nil {
						output.Warn(stderr, fmt.Sprintf("resolve STRUCTs for inputs: %v (showing raw inputs)", err))
						inputs = inputsRaw
					} else if resolved != nil {
						inputs = resolved
					} else {
						inputs = inputsRaw
					}
				}
				if returnsRaw, ok := toolEntry[constants.WrekenReturns]; ok {
					if resolved, err := ResolveReturns(wreken, returnsRaw); err != nil {
						output.Warn(stderr, fmt.Sprintf("resolve STRUCTs for returns: %v (output omitted)", err))
					} else if resolved != nil {
						toolOutput = resolved
					}
				}
			}
			if inputs == nil && toolEntry[constants.WrekenInputs] != nil {
				inputs = toolEntry[constants.WrekenInputs]
			}
		} else {
			if inputsRaw, ok := toolEntry[constants.WrekenInputs]; ok {
				inputs = inputsRaw
			} else if stepsRaw, ok := toolEntry["steps"]; ok {
				inputs = stepsRaw
			}
		}

		projectLibrary := fmt.Sprintf("%s.%s", match.Project, match.Library)
		integration := fmt.Sprintf("%s@%s", projectLibrary, match.Version)

		toolInfo := ToolInfo{
			CanonicalID: canonicalID,
			Type:        match.ToolType,
			Project:     match.Project,
			Library:     match.Library,
			Version:     match.Version,
			Integration: integration,
			Summary:     summary,
			Description: description,
			Inputs:      inputs,
			Output:      toolOutput,
			Wrekenfile:  toolEntry,
		}

		toolInfos = append(toolInfos, toolInfo)
	}

	return toolInfos, nil
}

// findToolInWrekenfile reads a wrekenfile YAML and finds a tool (method or workflow) by canonical_id key.
func findToolInWrekenfile(wrekenPath, canonicalID, toolType string) (map[string]interface{}, error) {
	data, err := os.ReadFile(wrekenPath)
	if err != nil {
		return nil, fmt.Errorf("read wrekenfile: %w", err)
	}

	var wreken map[string]interface{}
	if err := yaml.Unmarshal(data, &wreken); err != nil {
		return nil, fmt.Errorf("parse wrekenfile: %w", err)
	}

	var sectionName string
	if toolType == "method" {
		sectionName = constants.WrekenMethods
	} else if toolType == "workflow" {
		sectionName = constants.WrekenWorkflows
	} else {
		return nil, fmt.Errorf("unknown tool type: %q", toolType)
	}

	sectionRaw, ok := wreken[sectionName]
	if !ok {
		return nil, fmt.Errorf("%s section not found in wrekenfile", sectionName)
	}

	section, ok := sectionRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%s must be a map", sectionName)
	}

	toolEntry, ok := section[canonicalID]
	if !ok {
		return nil, fmt.Errorf("%s %q not found in wrekenfile", toolType, canonicalID)
	}

	toolMap, ok := toolEntry.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%s entry must be a map", toolType)
	}

	return toolMap, nil
}

// findWorkflowEntryInWorkflowsJSON reads workflows.json and returns the workflow entry as a map.
// Used for "info" when the tool is a workflow, so we do not depend on a WORKFLOWS section in the wrekenfile.
func findWorkflowEntryInWorkflowsJSON(workflowsPath, canonicalID string) (map[string]interface{}, error) {
	data, err := os.ReadFile(workflowsPath)
	if err != nil {
		return nil, fmt.Errorf("read workflows.json: %w", err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parse workflows.json: %w", err)
	}

	workflowsRaw, ok := out["workflows"]
	if !ok {
		return nil, fmt.Errorf("workflows section not found in workflows.json")
	}

	workflows, ok := workflowsRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("workflows must be an array")
	}

	for _, wRaw := range workflows {
		wMap, ok := wRaw.(map[string]interface{})
		if !ok {
			continue
		}
		if id, ok := wMap["canonical_id"].(string); ok && id == canonicalID {
			return wMap, nil
		}
	}

	return nil, fmt.Errorf("workflow %q not found in workflows.json", canonicalID)
}

// FormatInfoOutput formats tool info for display.
func FormatInfoOutput(toolInfos []ToolInfo, jsonOutput bool, stdout io.Writer) error {
	if jsonOutput {
		// Output as JSON array
		return json.NewEncoder(stdout).Encode(toolInfos)
	}

	// Human-readable output
	for i, info := range toolInfos {
		if i > 0 {
			fmt.Fprintln(stdout)
		}
		fmt.Fprintf(stdout, "Canonical ID: %s\n", info.CanonicalID)
		fmt.Fprintf(stdout, "Type: %s\n", info.Type)
		fmt.Fprintf(stdout, "Integration: %s\n", info.Integration)
		if info.Summary != "" {
			fmt.Fprintf(stdout, "Summary: %s\n", info.Summary)
		}
		if info.Description != "" {
			fmt.Fprintf(stdout, "Description: %s\n", info.Description)
		}
		if info.Inputs != nil {
			inputsJSON, err := json.MarshalIndent(info.Inputs, "", "  ")
			if err == nil {
				fmt.Fprintf(stdout, "Inputs:\n%s\n", inputsJSON)
			}
		}
		if info.Output != nil {
			outputJSON, err := json.MarshalIndent(info.Output, "", "  ")
			if err == nil {
				fmt.Fprintf(stdout, "Output:\n%s\n", outputJSON)
			}
		}
	}

	return nil
}
