// cursor.go writes Cursor editor rules so the editor delegates execution to swytchcode exec (init-time only).
package editors

import (
	"os"
	"path/filepath"
)

const cursorTemplatePath = "templates/cursor/swytchcode.mdc"

// WriteCursorRules creates .cursor/rules/swytchcode.mdc from the embedded template.
func WriteCursorRules(projectRoot string) error {
	content, err := templates.ReadFile(cursorTemplatePath)
	if err != nil {
		return err
	}
	rulesDir := filepath.Join(projectRoot, ".cursor", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		return err
	}
	rulesPath := filepath.Join(rulesDir, "swytchcode.mdc")
	return os.WriteFile(rulesPath, content, 0o644)
}

