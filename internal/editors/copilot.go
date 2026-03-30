// copilot.go writes GitHub Copilot instructions and VS Code MCP config (init-time only).
package editors

import (
	"path/filepath"
)

const copilotTemplatePath = "templates/copilot/copilot-instructions.md"

// WriteGitHubCopilotConfig writes .github/copilot-instructions.md from the embedded template.
// If the file already exists and contains the Swytchcode contract, it is left unchanged.
// If the file exists but does not contain the contract, the contract is appended.
// If the file does not exist, it is created fresh.
func WriteGitHubCopilotConfig(projectRoot string) error {
	destPath := filepath.Join(projectRoot, ".github", "copilot-instructions.md")
	return writeTemplateWithContractCheck(copilotTemplatePath, destPath)
}

// WriteVSCodeMCPConfig merges the swytchcode MCP server entry into .vscode/mcp.json.
// This enables GitHub Copilot Chat (and other VS Code MCP clients) to call swytchcode tools.
// Existing servers are preserved; only the "swytchcode" key is added or overwritten.
func WriteVSCodeMCPConfig(projectRoot string) error {
	configPath := filepath.Join(projectRoot, ".vscode", "mcp.json")
	return mergeMCPServerEntry(configPath, "servers", stdioMCPEntry)
}

