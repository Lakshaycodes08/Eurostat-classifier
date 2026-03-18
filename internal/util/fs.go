// fs.go provides directory creation, project-root resolution, and swytchcode path helpers.
package util

import (
	"os"
	"path/filepath"

	"gitlab.com/swytchcode/cli/internal/constants"
)

// EnsureDir ensures that the given directory exists, creating it
// (and any missing parents) with the provided permissions if needed.
func EnsureDir(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// ProjectRoot returns the current working directory.
// In the future this can be extended to walk up for markers
// (e.g. go.mod, .git, .swytchcode) if needed.
func ProjectRoot() (string, error) {
	return os.Getwd()
}

// Join is a thin wrapper around filepath.Join for convenience.
func Join(elem ...string) string {
	return filepath.Join(elem...)
}

// SwytchDir returns the path to the .swytchcode directory for the given project root.
func SwytchDir(projectRoot string) string {
	return filepath.Join(projectRoot, constants.SwytchDirName)
}

// ToolingPath returns the path to tooling.json for the given project root.
func ToolingPath(projectRoot string) string {
	return filepath.Join(SwytchDir(projectRoot), constants.ToolingJSONFile)
}

// IntegrationsDir returns the path to the integrations directory for the given project root.
func IntegrationsDir(projectRoot string) string {
	return filepath.Join(SwytchDir(projectRoot), constants.IntegrationsDirName)
}

// IntegrationVersionDir returns the versioned integration bundle directory path.
func IntegrationVersionDir(projectRoot, project, library, version string) string {
	return filepath.Join(IntegrationsDir(projectRoot), project, library, version)
}

// ManifestPath returns the path to integrations/manifest.json for the given project root.
func ManifestPath(projectRoot string) string {
	return filepath.Join(IntegrationsDir(projectRoot), constants.ManifestJSONFile)
}

// MCPPIDPath returns the path to the MCP daemon PID file for the given project root.
func MCPPIDPath(projectRoot string) string {
	return filepath.Join(SwytchDir(projectRoot), constants.MCPPIDFile)
}

