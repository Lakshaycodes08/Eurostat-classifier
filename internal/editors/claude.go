// claude.go writes CLAUDE.md at repo root so Claude uses swytchcode exec (init-time only).
package editors

import (
	"os"
	"path/filepath"
)

const claudeTemplatePath = "templates/claude/CLAUDE.md"

// WriteClaudeConfig writes root/CLAUDE.md from the embedded template.
func WriteClaudeConfig(projectRoot string) error {
	content, err := templates.ReadFile(claudeTemplatePath)
	if err != nil {
		return err
	}
	dest := filepath.Join(projectRoot, "CLAUDE.md")
	return os.WriteFile(dest, content, 0o644)
}

