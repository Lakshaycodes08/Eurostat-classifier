// cursor.go writes Cursor editor rules so the editor delegates execution to swytchcode exec (init-time only).
package editors

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gitlab.com/swytchcode/swytchcode-cli/internal/constants"
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

// WriteCursorMCPConfig merges the swytchcode MCP server entry into ~/.cursor/mcp.json.
// Existing servers are preserved; only the "swytchcode" key is added or overwritten.
func WriteCursorMCPConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, ".cursor", "mcp.json")

	config := map[string]interface{}{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &config)
	}

	servers, _ := config["mcpServers"].(map[string]interface{})
	if servers == nil {
		servers = map[string]interface{}{}
	}
	servers["swytchcode"] = map[string]interface{}{
		"url": fmt.Sprintf("http://localhost:%d/sse", constants.MCPDefaultPort),
	}
	config["mcpServers"] = servers

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

