// resolver.go resolves a tool ID to tooling.json entry and integration bundle.
package kernel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Tool represents a tool entry from tooling.json.
type Tool struct {
	CanonicalID string
	Type        string // "method" or "workflow"
	Integration string // "project.library@version" format
	Summary     string
	Desc        string
	Inputs      interface{} // Input schema
	Mode        string       // "production" or "sandbox" from tooling.json
}

// ResolveTool resolves a canonical_id to a Tool from tooling.json.
// For raw methods (isRaw=true), bypasses tooling.json and returns minimal Tool.
func ResolveTool(projectRoot, canonicalID string, isRaw bool) (*Tool, error) {
	if isRaw {
		// Raw methods bypass tooling.json
		// Parse "raw.project.library.operation" format
		parts := strings.Split(canonicalID, ".")
		if len(parts) < 3 {
			return nil, fmt.Errorf("invalid raw method format: %q (expected raw.project.library.operation)", canonicalID)
		}
		// For raw, we'll need to search integrations to find version
		// For now, return error - raw method resolution needs integration discovery
		return nil, fmt.Errorf("raw method resolution not yet implemented")
	}

	// Load tooling.json
	toolingPath := filepath.Join(projectRoot, ".swytchcode", "tooling.json")
	data, err := os.ReadFile(toolingPath)
	if err != nil {
		return nil, fmt.Errorf("tooling.json not found; run 'swytchcode init' first")
	}

	var tooling map[string]interface{}
	if err := json.Unmarshal(data, &tooling); err != nil {
		return nil, fmt.Errorf("failed to parse tooling.json: %w", err)
	}

	// Get mode from tooling.json (defaults to "production")
	mode := "production"
	if modeRaw, ok := tooling["mode"].(string); ok {
		mode = modeRaw
	}

	// Find tool in tools map
	tools, ok := tooling["tools"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("tool %q not found in tooling.json. Run: swytchcode add %s", canonicalID, canonicalID)
	}

	toolRaw, ok := tools[canonicalID]
	if !ok {
		return nil, fmt.Errorf("tool %q not found in tooling.json. Run: swytchcode add %s", canonicalID, canonicalID)
	}

	toolMap, ok := toolRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid tool entry format in tooling.json")
	}

	// Extract tool fields
	tool := &Tool{
		CanonicalID: canonicalID,
		Mode:        mode,
	}

	if integrationRaw, ok := toolMap["integration"].(string); ok {
		tool.Integration = integrationRaw
	} else {
		return nil, fmt.Errorf("tool %q missing integration field", canonicalID)
	}

	if typeRaw, ok := toolMap["type"].(string); ok {
		tool.Type = typeRaw
	}

	if summaryRaw, ok := toolMap["summary"].(string); ok {
		tool.Summary = summaryRaw
	}

	if descRaw, ok := toolMap["desc"].(string); ok {
		tool.Desc = descRaw
	}

	if inputsRaw, ok := toolMap["inputs"]; ok {
		tool.Inputs = inputsRaw
	}

	return tool, nil
}
