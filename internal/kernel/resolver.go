// resolver.go resolves a tool ID to tooling.json entry and integration bundle.
package kernel

import (
	"fmt"

	"gitlab.com/swytchcode/swytchcode-cli/internal/util"
)

// LocalWorkflowStep is a workflow step read from the local tooling.json.
type LocalWorkflowStep struct {
	CanonicalID string
	Integration string // "project.library@version" format
	Name        string
}

// Tool represents a tool entry from tooling.json.
type Tool struct {
	CanonicalID string
	Type        string // "method" or "workflow"
	Integration string // "project.library@version" format
	Summary     string
	Desc        string
	Inputs      interface{}         // Input schema
	Mode        string              // "production" or "sandbox" from tooling.json
	Steps       []LocalWorkflowStep // populated for workflow tools
}

// ErrToolNotFound is returned when a canonical ID is not present in tooling.json.
// It is distinct from other resolver errors (parse failures, missing fields) so
// executor.go can safely fall back to demo mode only for the not-found case.
type ErrToolNotFound struct {
	CanonicalID string
}

func (e *ErrToolNotFound) Error() string {
	return fmt.Sprintf("tool %q not found in tooling.json — run 'swytchcode add %s' to add it, or 'swytchcode search' to find it", e.CanonicalID, e.CanonicalID)
}

// ResolveTool resolves a canonical_id to a Tool from tooling.json.
func ResolveTool(projectRoot, canonicalID string, isRaw bool) (*Tool, error) {
	tooling, err := util.LoadToolingJSON(projectRoot)
	if err != nil {
		return nil, err
	}

	// Get mode from tooling.json (defaults to "production")
	mode := "production"
	if modeRaw, ok := tooling["mode"].(string); ok {
		mode = modeRaw
	}

	// Find tool in tools map
	tools, ok := tooling["tools"].(map[string]interface{})
	if !ok {
		return nil, &ErrToolNotFound{CanonicalID: canonicalID}
	}

	toolRaw, ok := tools[canonicalID]
	if !ok {
		return nil, &ErrToolNotFound{CanonicalID: canonicalID}
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

	// Read workflow steps if present
	if stepsRaw, ok := toolMap["steps"].([]interface{}); ok {
		for _, sRaw := range stepsRaw {
			sMap, ok := sRaw.(map[string]interface{})
			if !ok {
				continue
			}
			step := LocalWorkflowStep{}
			if v, ok := sMap["canonical_id"].(string); ok {
				step.CanonicalID = v
			}
			if v, ok := sMap["integration"].(string); ok {
				step.Integration = v
			}
			if v, ok := sMap["name"].(string); ok {
				step.Name = v
			}
			if step.CanonicalID != "" && step.Integration != "" {
				tool.Steps = append(tool.Steps, step)
			}
		}
	}

	return tool, nil
}
