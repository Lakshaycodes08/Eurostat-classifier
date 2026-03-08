// claude.go writes CLAUDE.md at repo root so Claude uses swytchcode exec (init-time only).
package editors

import (
	"os"
	"path/filepath"
	"strings"
)

const claudeTemplatePath = "templates/claude/CLAUDE.md"

// WriteClaudeConfig writes root/CLAUDE.md from the embedded template.
// If CLAUDE.md already exists and contains the Swytchcode contract, it is left unchanged.
// If CLAUDE.md exists but does not contain the contract, the contract is appended.
// If CLAUDE.md does not exist, it is created fresh.
func WriteClaudeConfig(projectRoot string) error {
	content, err := templates.ReadFile(claudeTemplatePath)
	if err != nil {
		return err
	}
	dest := filepath.Join(projectRoot, "CLAUDE.md")

	existing, err := os.ReadFile(dest)
	if err == nil {
		// File exists — skip if swytchcode contract already present
		if strings.Contains(string(existing), "# Swytchcode Agent Contract") {
			return nil
		}
		// Append with separator
		separator := []byte("\n\n---\n\n")
		return os.WriteFile(dest, append(existing, append(separator, content...)...), 0o644)
	}
	// File doesn't exist — write fresh
	return os.WriteFile(dest, content, 0o644)
}
