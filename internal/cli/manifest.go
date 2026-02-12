// manifest.go manages .swytchcode/manifest.json (registry manifest with project.library entries).
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ManifestEntry represents a single project.library entry in manifest.json.
type ManifestEntry struct {
	Version           string                 `json:"version"`
	SandboxEndpoint   string                 `json:"sandbox_endpoint"`
	ProductionEndpoint string                `json:"production_endpoint"`
	Methods           int                    `json:"methods"`
	Workflows         int                    `json:"workflows"`
	Auth              map[string]interface{} `json:"auth,omitempty"`
}

// manifestPath returns the path to .swytchcode/integrations/manifest.json.
func manifestPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".swytchcode", "integrations", "manifest.json")
}

// readManifest reads the manifest.json file.
// Returns nil map if file is missing or empty.
func readManifest(projectRoot string) (map[string]ManifestEntry, error) {
	data, err := os.ReadFile(manifestPath(projectRoot))
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]ManifestEntry), nil
		}
		return nil, err
	}
	var m map[string]ManifestEntry
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = make(map[string]ManifestEntry)
	}
	return m, nil
}

// writeManifest writes the manifest map to manifest.json.
func writeManifest(projectRoot string, manifest map[string]ManifestEntry) error {
	if manifest == nil {
		manifest = make(map[string]ManifestEntry)
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	p := manifestPath(projectRoot)
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// updateManifestEntry updates or creates a manifest entry for project.library.
func updateManifestEntry(projectRoot, projectLibrary, version, sandboxEndpoint, productionEndpoint string, methodsCount, workflowsCount int, auth map[string]interface{}) error {
	if projectLibrary == "" {
		return fmt.Errorf("project.library name cannot be empty")
	}
	if version == "" {
		return fmt.Errorf("version cannot be empty for %q", projectLibrary)
	}
	
	manifest, err := readManifest(projectRoot)
	if err != nil {
		return err
	}
	
	manifest[projectLibrary] = ManifestEntry{
		Version:           version,
		SandboxEndpoint:   sandboxEndpoint,
		ProductionEndpoint: productionEndpoint,
		Methods:           methodsCount,
		Workflows:         workflowsCount,
		Auth:              auth,
	}
	
	return writeManifest(projectRoot, manifest)
}
