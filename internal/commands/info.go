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
	"gitlab.com/swytchcode/shell/internal/util"
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
	Inputs       interface{}            `json:"inputs,omitempty"`
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

	// For each match, read the wrekenfile and extract tool details
	var toolInfos []ToolInfo
	for _, match := range matches {
		wrekenPath := filepath.Join(projectRoot, ".swytchcode", "integrations", match.Project, match.Library, match.Version, "wrekenfile.yaml")
		
		toolEntry, err := findToolInWrekenfile(wrekenPath, canonicalID, match.ToolType)
		if err != nil {
			fmt.Fprintf(stderr, "Warning: failed to read wrekenfile for %s.%s@%s: %v\n", match.Project, match.Library, match.Version, err)
			continue
		}

		// Extract common fields
		summary := ""
		if summaryRaw, ok := toolEntry["SUMMARY"]; ok {
			if summaryStr, ok := summaryRaw.(string); ok {
				summary = summaryStr
			}
		}

		description := ""
		if descRaw, ok := toolEntry["DESC"]; ok {
			if descStr, ok := descRaw.(string); ok {
				description = descStr
			}
		}

		var inputs interface{}
		if inputsRaw, ok := toolEntry["INPUTS"]; ok {
			inputs = inputsRaw
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
		sectionName = "METHODS"
	} else if toolType == "workflow" {
		sectionName = "WORKFLOWS"
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
	}

	return nil
}
