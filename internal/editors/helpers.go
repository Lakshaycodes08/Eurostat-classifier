// helpers.go provides shared utilities used by the editor config writers.
package editors

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// stdioMCPEntry is the standard stdio MCP server entry written by all editor configs.
var stdioMCPEntry = map[string]interface{}{
	"type":    "stdio",
	"command": "swytchcode",
	"args":    []string{"mcp", "serve"},
}

// writeTemplateWithContractCheck writes templatePath content to destPath.
//   - If destPath exists and already contains "# Swytchcode Agent Contract", it is left unchanged.
//   - If destPath exists but lacks the marker, the template is appended with a separator.
//   - If destPath does not exist, it is created fresh (parent directories are created automatically).
func writeTemplateWithContractCheck(templatePath, destPath string) error {
	content, err := templates.ReadFile(templatePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	existing, err := os.ReadFile(destPath)
	if err == nil {
		if strings.Contains(string(existing), "# Swytchcode Agent Contract") {
			return nil
		}
		separator := []byte("\n\n---\n\n")
		return os.WriteFile(destPath, append(existing, append(separator, content...)...), 0o644)
	}
	return os.WriteFile(destPath, content, 0o644)
}

// mergeMCPServerEntry reads configPath as JSON, sets config[serversKey]["swytchcode"] = entry,
// and writes the result back. Parent directories are created if needed.
// If the file does not exist, a fresh config object is created.
func mergeMCPServerEntry(configPath, serversKey string, entry map[string]interface{}) error {
	config := map[string]interface{}{}
	if data, err := os.ReadFile(configPath); err == nil {
		_ = json.Unmarshal(data, &config)
	}
	servers, _ := config[serversKey].(map[string]interface{})
	if servers == nil {
		servers = map[string]interface{}{}
	}
	servers["swytchcode"] = entry
	config[serversKey] = servers
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0o644)
}
