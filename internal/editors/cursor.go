// cursor.go writes Cursor editor rules so the editor delegates execution to swytchcode exec (init-time only).
package editors

import (
	"os"
	"path/filepath"
)

const cursorTemplatePath = "templates/cursor/swytchcode.mdc"

// WriteCursorRules creates .cursor/rules/swytchcode.mdc from the embedded template.
func WriteCursorRules(projectRoot string) error {
	rulesPath := filepath.Join(projectRoot, ".cursor", "rules", "swytchcode.mdc")
	return writeTemplateWithContractCheck(cursorTemplatePath, rulesPath)
}

// WriteCursorMCPConfig merges the swytchcode MCP server entry into ~/.cursor/mcp.json.
// Existing servers are preserved; only the "swytchcode" key is added or overwritten.
// Uses stdio transport so Cursor spawns the server as a subprocess (no daemon required).
func WriteCursorMCPConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	// Cursor stdio entry omits "type" — command+args is sufficient for Cursor's MCP format.
	cursorEntry := map[string]interface{}{
		"command": "swytchcode",
		"args":    []string{"mcp", "serve"},
	}
	return mergeMCPServerEntry(
		filepath.Join(home, ".cursor", "mcp.json"),
		"mcpServers",
		cursorEntry,
	)
}
