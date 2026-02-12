// add.go implements the add command (adds tools to tooling.json by canonical_id).
// Searches methods.json and workflows.json files across all fetched integrations.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"gitlab.com/swytchcode/shell/internal/util"
)

// parseIntegrationSpec parses "project@library.version" and returns (project, library, version).
// Example: "weaviate@lyrid.v1" -> ("weaviate", "lyrid", "v1")
func parseIntegrationSpec(spec string) (project, library, version string) {
	// Split on @ to get project and rest
	parts := strings.SplitN(spec, "@", 2)
	if len(parts) != 2 {
		return "", "", ""
	}
	project = strings.TrimSpace(parts[0])
	rest := strings.TrimSpace(parts[1])
	
	// Split rest on . to get library and version
	// Last dot separates library from version
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
	project   string
	library   string
	version   string
	toolType  string // "method" or "workflow"
	canonicalID string
}

// findToolInAllIntegrations searches for a canonical_id across all fetched integrations.
// Searches both methods.json and workflows.json files.
// Returns list of matches with project, library, version, and tool type.
func findToolInAllIntegrations(projectRoot, canonicalID string) ([]toolMatch, error) {
	integrationsDir := filepath.Join(projectRoot, ".swytchcode", "integrations")
	
	// Check if integrations directory exists
	if _, err := os.Stat(integrationsDir); err != nil {
		return nil, fmt.Errorf("no integrations found. Run: swytchcode get <project>")
	}
	
	var matches []toolMatch
	
	// Walk through integrations directory structure: integrations/{project}/{library}/{version}/
	err := filepath.Walk(integrationsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}
		
		// Look for methods.json and workflows.json files
		if info.Name() == "methods.json" || info.Name() == "workflows.json" {
			// Extract project, library, version from path
			// Path format: integrations/{project}/{library}/{version}/methods.json
			relPath, err := filepath.Rel(integrationsDir, path)
			if err != nil {
				return nil
			}
			
			// Split path: project/library/version/methods.json
			parts := strings.Split(filepath.Dir(relPath), string(filepath.Separator))
			if len(parts) != 3 {
				return nil
			}
			
			project := parts[0]
			library := parts[1]
			version := parts[2]
			
			// Determine tool type from filename
			toolType := "method"
			if info.Name() == "workflows.json" {
				toolType = "workflow"
			}
			
			// Read and search the JSON file
			data, err := os.ReadFile(path)
			if err != nil {
				return nil // Skip if can't read
			}
			
			var result map[string]interface{}
			if err := json.Unmarshal(data, &result); err != nil {
				return nil // Skip if invalid JSON
			}
			
			// Search in methods or workflows array
			var items []interface{}
			if toolType == "method" {
				if methodsRaw, ok := result["methods"].([]interface{}); ok {
					items = methodsRaw
				}
			} else {
				if workflowsRaw, ok := result["workflows"].([]interface{}); ok {
					items = workflowsRaw
				}
			}
			
			// Search for canonical_id in items
			for _, itemRaw := range items {
				item, ok := itemRaw.(map[string]interface{})
				if !ok {
					continue
				}
				
				itemCanonicalID, ok := item["canonical_id"].(string)
				if !ok {
					continue
				}
				
				if itemCanonicalID == canonicalID {
					matches = append(matches, toolMatch{
						project:     project,
						library:     library,
						version:     version,
						toolType:    toolType,
						canonicalID: canonicalID,
					})
					break // Found in this file, no need to continue
				}
			}
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("search integrations: %w", err)
	}
	
	return matches, nil
}

// findMethodInWrekenfile reads a wrekenfile YAML and finds a method by canonical_id key.
// Returns the method entry as a map if found.
func findMethodInWrekenfile(wrekenPath, canonicalID string) (map[string]interface{}, error) {
	data, err := os.ReadFile(wrekenPath)
	if err != nil {
		return nil, err
	}
	
	var wreken map[string]interface{}
	if err := yaml.Unmarshal(data, &wreken); err != nil {
		return nil, fmt.Errorf("parse wrekenfile: %w", err)
	}
	
	// Look for METHODS section (wrekenfile uses METHODS, not INTERFACES)
	methodsRaw, ok := wreken["METHODS"]
	if !ok {
		return nil, fmt.Errorf("METHODS section not found in wrekenfile")
	}
	
	methods, ok := methodsRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("METHODS must be a map")
	}
	
	// Find method by canonical_id key
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

// addCmd is the main add command that accepts canonical_id directly.
// Usage:
//   - swytchcode add <canonical_id> (searches all integrations)
//   - swytchcode add <integration_spec> <canonical_id> (explicit scoped)
var addCmd = &cobra.Command{
	Use:   "add [integration_spec] <canonical_id>",
	Short: "Add a tool (method or workflow) to tooling.json by canonical_id",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 || len(args) > 2 {
			return errors.New("usage: swytchcode add [integration_spec] <canonical_id>")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return fmt.Errorf("detect project root: %w", err)
		}
		
		var canonicalID string
		var integrationSpec string
		var mode1 bool
		
		if len(args) == 1 {
			// Mode 1: canonical_id only
			canonicalID = args[0]
			mode1 = true
		} else {
			// Mode 2: explicit scoped
			integrationSpec = args[0]
			canonicalID = args[1]
			mode1 = false
		}
		
		var project, library, version, toolType string
		
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
				// Ambiguity - prompt user if interactive, else error
				interactive := util.IsInteractive()
				if !interactive {
					var options []string
					for _, m := range matches {
						options = append(options, fmt.Sprintf("%s@%s.%s (%s)", m.project, m.library, m.version, m.toolType))
					}
					return fmt.Errorf("ambiguous canonical ID. Found in %d integrations:\n  %s\nUse: swytchcode add <integration@version> %s", len(matches), strings.Join(options, "\n  "), canonicalID)
				}
				
				// Interactive prompt
				options := make([]string, len(matches))
				for i, m := range matches {
					options[i] = fmt.Sprintf("%s@%s.%s (%s)", m.project, m.library, m.version, m.toolType)
				}
				
				fmt.Printf("\nCanonical ID %q found in multiple integrations:\n\n", canonicalID)
				for i, opt := range options {
					fmt.Printf("  %d) %s\n", i+1, opt)
				}
				fmt.Println()
				
				_, selected := util.SelectWithRetry("Select integration:", options)
				if selected == "" {
					return errors.New("no integration selected")
				}
				
				// Parse selected integration spec (remove tool type suffix)
				selectedParts := strings.Split(selected, " (")
				if len(selectedParts) > 0 {
					selected = selectedParts[0]
				}
				
				project, library, version = parseIntegrationSpec(selected)
				if project == "" || library == "" || version == "" {
					return fmt.Errorf("invalid integration spec format: %q", selected)
				}
				
				// Find tool type from matches
				for _, m := range matches {
					if m.project == project && m.library == library && m.version == version {
						toolType = m.toolType
						break
					}
				}
			} else {
				// Single match
				project = matches[0].project
				library = matches[0].library
				version = matches[0].version
				toolType = matches[0].toolType
			}
		} else {
			// Mode 2: Parse explicit integration spec
			project, library, version = parseIntegrationSpec(integrationSpec)
			if project == "" || library == "" || version == "" {
				return fmt.Errorf("invalid integration spec format: %q (expected: project@library.version)", integrationSpec)
			}
			
			// Verify integration exists
			integrationsDir := filepath.Join(projectRoot, ".swytchcode", "integrations", project, library, version)
			if _, err := os.Stat(integrationsDir); err != nil {
				return fmt.Errorf("integration %s@%s.%s not found.\nRun: swytchcode get %s", project, library, version, project)
			}
			
			// Search for canonical_id in this specific integration
			matches, err := findToolInAllIntegrations(projectRoot, canonicalID)
			if err != nil {
				return err
			}
			
			// Filter matches to this specific integration
			var foundMatch *toolMatch
			for _, m := range matches {
				if m.project == project && m.library == library && m.version == version {
					foundMatch = &m
					break
				}
			}
			
			if foundMatch == nil {
				return fmt.Errorf("canonical ID %q not found in %s@%s.%s", canonicalID, project, library, version)
			}
			
			toolType = foundMatch.toolType
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
		
		// Ensure integrations map exists
		integrations, _ := tooling["integrations"].(map[string]interface{})
		if integrations == nil {
			integrations = make(map[string]interface{})
			tooling["integrations"] = integrations
		}
		
		// Ensure integration is tracked in tooling.json (key is project.library)
		projectLibrary := fmt.Sprintf("%s.%s", project, library)
		if _, exists := integrations[projectLibrary]; !exists {
			integrations[projectLibrary] = map[string]interface{}{
				"version": version,
			}
		}
		
		// Ensure tools map exists
		tools, _ := tooling["tools"].(map[string]interface{})
		if tools == nil {
			tools = make(map[string]interface{})
			tooling["tools"] = tools
		}
		
		// Build integration string (project.library@version format)
		integration := fmt.Sprintf("%s@%s", projectLibrary, version)
		
		// Extract summary and desc from wrekenfile entry
		summary := ""
		if summaryRaw, ok := toolEntry["SUMMARY"]; ok {
			if summaryStr, ok := summaryRaw.(string); ok {
				summary = summaryStr
			}
		}
		
		desc := ""
		if descRaw, ok := toolEntry["DESC"]; ok {
			if descStr, ok := descRaw.(string); ok {
				desc = descStr
			}
		}
		
		// Extract inputs from INPUTS array
		var inputs interface{}
		if inputsRaw, ok := toolEntry["INPUTS"]; ok {
			inputs = inputsRaw // Keep as-is (array/list)
		}
		
		// Add tool to tools map (keyed by canonical_id)
		// Include: summary, integration, type, desc, inputs
		// Note: endpoint is NOT stored here - it comes from Wreken HTTP.ENDPOINT
		// Base URL comes from manifest.json based on mode in tooling.json
		toolDef := make(map[string]interface{})
		toolDef["summary"] = summary
		toolDef["integration"] = integration
		toolDef["type"] = toolType
		toolDef["desc"] = desc
		if inputs != nil {
			toolDef["inputs"] = inputs
		}
		
		tools[canonicalID] = toolDef
		
		// Write updated tooling.json
		data, err = json.MarshalIndent(tooling, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal tooling.json: %w", err)
		}
		
		if err := os.WriteFile(toolingPath, data, 0o644); err != nil {
			return fmt.Errorf("write tooling.json: %w", err)
		}
		
		if util.IsInteractive() {
			fmt.Printf("Added %s %q to tooling.json (integration: %s)\n", toolType, canonicalID, projectLibrary)
		}
		
		return nil
	},
}

// addIntegrationCmd adds an integration version to tooling.json (does not fetch).
// Usage: swytchcode add integration weaviate@lyrid.v1
var addIntegrationCmd = &cobra.Command{
	Use:   "integration <project@library.version>",
	Short: "Add an integration version to tooling.json (explicit, reviewable; does not fetch)",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("integration spec required (e.g. weaviate@lyrid.v1)")
		}
		project, library, version := parseIntegrationSpec(args[0])
		if project == "" || library == "" || version == "" {
			return errors.New("integration must be <project@library.version> (e.g. weaviate@lyrid.v1)")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		project, library, version := parseIntegrationSpec(args[0])
		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return fmt.Errorf("detect project root: %w", err)
		}
		toolingPath := filepath.Join(projectRoot, ".swytchcode", "tooling.json")
		var tooling map[string]interface{}
		if data, err := os.ReadFile(toolingPath); err == nil {
			if err := json.Unmarshal(data, &tooling); err != nil {
				return fmt.Errorf("parse tooling.json: %w", err)
			}
		} else {
			return fmt.Errorf("tooling.json not found; run 'swytchcode init' first: %w", err)
		}
		integrations, _ := tooling["integrations"].(map[string]interface{})
		if integrations == nil {
			integrations = make(map[string]interface{})
			tooling["integrations"] = integrations
		}
		// Use project.library as key
		projectLibrary := fmt.Sprintf("%s.%s", project, library)
		integrations[projectLibrary] = map[string]interface{}{"version": version}
		data, err := json.MarshalIndent(tooling, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal tooling.json: %w", err)
		}
		if err := os.WriteFile(toolingPath, data, 0o644); err != nil {
			return fmt.Errorf("write tooling.json: %w", err)
		}
		if util.IsInteractive() {
			fmt.Printf("Added %s@%s.%s to tooling.json. Run 'swytchcode bootstrap' to install.\n", project, library, version)
		}
		return nil
	},
}

func init() {
	addCmd.AddCommand(addIntegrationCmd)
}
