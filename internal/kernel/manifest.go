// manifest.go resolves base URL from manifest.json based on mode.
package kernel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetBaseURL gets the base URL for an integration from manifest.json based on mode.
// integration format: "project.library@version"
// mode: "production" or "sandbox"
func GetBaseURL(projectRoot, integration, mode string) (string, error) {
	// Parse integration to get project.library
	parts := strings.Split(integration, "@")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid integration format: %q", integration)
	}
	projectLibrary := parts[0]

	// Load manifest.json
	manifestPath := filepath.Join(projectRoot, ".swytchcode", "integrations", "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", fmt.Errorf("manifest.json not found. Run: swytchcode get <project>")
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", fmt.Errorf("failed to parse manifest.json: %w", err)
	}

	// Find entry for project.library
	entryRaw, ok := manifest[projectLibrary]
	if !ok {
		return "", fmt.Errorf("integration %q not found in manifest.json", projectLibrary)
	}

	entry, ok := entryRaw.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid manifest entry format for %q", projectLibrary)
	}

	// Get endpoint based on mode
	var endpoint string
	if mode == "sandbox" {
		if sandboxRaw, ok := entry["sandbox_endpoint"].(string); ok {
			endpoint = sandboxRaw
		}
	} else {
		// Default to production
		if prodRaw, ok := entry["production_endpoint"].(string); ok {
			endpoint = prodRaw
		}
	}

	if endpoint == "" {
		return "", fmt.Errorf("no %s endpoint found for integration %q in manifest.json", mode, projectLibrary)
	}

	return endpoint, nil
}
