// manifest.go resolves base URL from manifest.json based on mode.
package kernel

import (
	"fmt"
	"strings"

	"gitlab.com/swytchcode/cli/internal/manifest"
)

// GetBaseURL gets the base URL for an integration from manifest.json based on mode.
// integration format: "project.library@version"
// mode: "production" or "sandbox"
func GetBaseURL(projectRoot, integration, mode string) (string, error) {
	parts := strings.Split(integration, "@")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid integration format: %q", integration)
	}
	projectLibrary := parts[0]

	m, err := manifest.Read(projectRoot)
	if err != nil {
		return "", fmt.Errorf("read manifest: %w", err)
	}
	entry, ok := m[projectLibrary]
	if !ok {
		return "", fmt.Errorf("integration %q not found in manifest.json", projectLibrary)
	}

	var endpoint string
	if mode == "sandbox" {
		endpoint = entry.SandboxEndpoint
	} else {
		endpoint = entry.ProductionEndpoint
	}
	if endpoint == "" {
		return "", fmt.Errorf("no %s endpoint found for integration %q in manifest.json", mode, projectLibrary)
	}
	return endpoint, nil
}
