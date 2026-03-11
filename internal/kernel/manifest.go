// manifest.go resolves base URL from manifest.json based on mode.
package kernel

import (
	"fmt"
	"os"
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
	// Optional: warn when using a plain http://localhost base URL, which usually implies
	// a local backend. This is a best-effort hint only and does not affect execution.
	if endpoint == "http://localhost" {
		fmt.Fprintln(os.Stderr, "[swytchcode] warning: integration", projectLibrary, "is using base URL http://localhost – this expects a local backend and is unrelated to the MCP HTTP server address")
	}
	return endpoint, nil
}
