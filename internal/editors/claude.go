// claude.go writes CLAUDE.md at repo root so Claude uses swytchcode exec (init-time only).
package editors

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/swytchcode/cli/internal/constants"
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

// WriteClaudeMCPConfig merges the swytchcode MCP server entry into ~/.claude/settings.json.
// Existing settings are preserved; only the "swytchcode" key under mcpServers is added or overwritten.
func WriteClaudeMCPConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, ".claude", "settings.json")

	config := map[string]interface{}{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &config)
	}

	servers, _ := config["mcpServers"].(map[string]interface{})
	if servers == nil {
		servers = map[string]interface{}{}
	}
	servers["swytchcode"] = map[string]interface{}{
		"type": "sse",
		"url":  fmt.Sprintf("http://localhost:%d/sse", constants.MCPDefaultPort),
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
