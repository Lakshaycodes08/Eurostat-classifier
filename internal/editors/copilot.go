// copilot.go writes GitHub Copilot agent instructions: .github/instructions/swytchcode.md
package editors

import (
	"os"
	"path/filepath"
)

const copilotTemplatePath = "templates/copilot/swytchcode.md"

// WriteCopilotConfig writes .github/instructions/swytchcode.md (GitHub Copilot).
func WriteCopilotConfig(projectRoot string) error {
	copilotContent, err := templates.ReadFile(copilotTemplatePath)
	if err != nil {
		return err
	}
	instructionsDir := filepath.Join(projectRoot, ".github", "instructions")
	if err := os.MkdirAll(instructionsDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(instructionsDir, "swytchcode.md"), copilotContent, 0o644)
}
