// claude.go writes CLAUDE.md at repo root so Claude uses swytchcode exec (init-time only).
package editors

import (
	"os"
	"path/filepath"
)

const claudeTemplatePath = "templates/claude/CLAUDE.md"

// WriteClaudeConfig writes root/CLAUDE.md from the embedded template.
// If CLAUDE.md already exists and contains the Swytchcode contract, it is left unchanged.
// If CLAUDE.md exists but does not contain the contract, the contract is appended.
// If CLAUDE.md does not exist, it is created fresh.
func WriteClaudeConfig(projectRoot string) error {
	return writeTemplateWithContractCheck(claudeTemplatePath, filepath.Join(projectRoot, "CLAUDE.md"))
}

// WriteClaudeMCPConfig merges the swytchcode MCP server entry into ~/.claude/settings.json.
// Existing settings are preserved; only the "swytchcode" key under mcpServers is added or overwritten.
// Uses stdio transport so Claude Code spawns the server as a subprocess (no daemon required).
func WriteClaudeMCPConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return mergeMCPServerEntry(
		filepath.Join(home, ".claude", "settings.json"),
		"mcpServers",
		stdioMCPEntry,
	)
}
