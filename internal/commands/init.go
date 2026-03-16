// init.go provides shared init command logic for CLI and MCP.
package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gitlab.com/swytchcode/cli/internal/constants"
	"gitlab.com/swytchcode/cli/internal/editors"
	"gitlab.com/swytchcode/cli/internal/output"
	"gitlab.com/swytchcode/cli/internal/util"
)

// RunInit runs the init command: creates .swytchcode/, tooling.json, and editor-specific config.
// editor and mode are required (non-interactive mode).
func RunInit(projectRoot, editor, mode string, stdout, stderr io.Writer) error {
	// Accumulate validation errors so both are shown at once
	var validationErrs []string
	if mode != "production" && mode != "sandbox" {
		validationErrs = append(validationErrs, fmt.Sprintf("invalid mode %q (expected production or sandbox)", mode))
	}
	validEditors := map[string]bool{"cursor": true, "claude": true, "none": true}
	if editor != "" && !validEditors[editor] {
		validationErrs = append(validationErrs, fmt.Sprintf("unknown editor %q (expected cursor|claude|none)", editor))
	}
	if len(validationErrs) > 0 {
		output.ValidationErrors(stderr, validationErrs)
		return fmt.Errorf("validation failed")
	}

	swytchDir := filepath.Join(projectRoot, ".swytchcode")
	if err := util.EnsureDir(swytchDir, 0o755); err != nil {
		return fmt.Errorf("create .swytchcode directory: %w", err)
	}
	// Create integrations directory where all integration data (wrekenfile, methods, workflows)
	// will be stored by swytchcode get.
	if err := util.EnsureDir(filepath.Join(swytchDir, "integrations"), 0o755); err != nil {
		return fmt.Errorf("create integrations directory: %w", err)
	}

	// Create or update tooling.json with mode
	toolingPath := filepath.Join(swytchDir, "tooling.json")
	var tooling map[string]interface{}

	if data, err := os.ReadFile(toolingPath); err == nil {
		// Load existing tooling.json
		if err := json.Unmarshal(data, &tooling); err != nil {
			// If invalid, start fresh
			tooling = make(map[string]interface{})
		}
	} else {
		// Create new tooling.json
		tooling = make(map[string]interface{})
	}

	// Ensure tools and integrations maps exist (integrations = pinned versions for determinism)
	if _, ok := tooling["tools"]; !ok {
		tooling["tools"] = make(map[string]interface{})
	}
	if _, ok := tooling["integrations"]; !ok {
		tooling["integrations"] = make(map[string]interface{})
	}

	// Set mode and version so they are visible and project-specific.
	// version is owned by the kernel: only set when absent (never overwrite on re-init).
	tooling["mode"] = mode
	if _, hasVersion := tooling["version"]; !hasVersion {
		tooling["version"] = constants.Version
	}

	// Write tooling.json
	data, err := json.MarshalIndent(tooling, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tooling.json: %w", err)
	}
	if err := os.WriteFile(toolingPath, data, 0o644); err != nil {
		return fmt.Errorf("write tooling.json: %w", err)
	}

	// Write editor config if editor is specified
	if editor != "" && editor != "none" {
		switch editor {
		case "cursor":
			if err := editors.WriteCursorRules(projectRoot); err != nil {
				return fmt.Errorf("write Cursor rules: %w", err)
			}
		case "claude":
			if err := editors.WriteClaudeConfig(projectRoot); err != nil {
				return fmt.Errorf("write Claude config: %w", err)
			}
		}
	}

	fmt.Fprintf(stdout, "Swytchcode initialized for project at %s\n", projectRoot)
	return nil
}
