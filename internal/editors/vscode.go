// vscode.go writes VS Code agent instructions: Copilot (.github/instructions) and Claude (CLAUDE.md).
package editors

import (
	"os"
	"path/filepath"
)

const (
	copilotTemplatePath = "templates/vscode/copilot/swytchcode.md"
)

// WriteVSCodeConfig writes .github/instructions/swytchcode.md (Copilot) and root/CLAUDE.md (Claude in VS Code).
func WriteVSCodeConfig(projectRoot string) error {
	// Copilot: .github/instructions/swytchcode.md
	copilotContent, err := templates.ReadFile(copilotTemplatePath)
	if err != nil {
		return err
	}
	instructionsDir := filepath.Join(projectRoot, ".github", "instructions")
	if err := os.MkdirAll(instructionsDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(instructionsDir, "swytchcode.md"), copilotContent, 0o644); err != nil {
		return err
	}
	// Claude in VS Code: same CLAUDE.md at repo root
	return WriteClaudeConfig(projectRoot)
}

