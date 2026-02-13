// manifest.go manages .swytchcode/manifest.json (registry manifest with project.library entries).
// This file is kept for backward compatibility. New code should use internal/manifest package.
package cli

import (
	"gitlab.com/swytchcode/shell/internal/manifest"
)

// UpdateManifestEntry updates or creates a manifest entry for project.library.
// Deprecated: Use manifest.UpdateEntry instead.
func UpdateManifestEntry(projectRoot, projectLibrary, version, sandboxEndpoint, productionEndpoint string, methodsCount, workflowsCount int, auth map[string]interface{}) error {
	return manifest.UpdateEntry(projectRoot, projectLibrary, version, sandboxEndpoint, productionEndpoint, methodsCount, workflowsCount, auth)
}
