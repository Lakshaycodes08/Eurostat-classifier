// manifest.go manages .swytchcode/integrations/manifest.json (shared by cli and mcp).
package manifest

import (
	"encoding/json"
	"fmt"
	"os"

	"gitlab.com/swytchcode/cli/internal/util"
)

// Entry represents a single project.library entry in manifest.json.
type Entry struct {
	Version           string                 `json:"version"`
	SandboxEndpoint   string                 `json:"sandbox_endpoint"`
	ProductionEndpoint string                `json:"production_endpoint"`
	Methods           int                    `json:"methods"`
	Workflows         int                    `json:"workflows"`
	Auth              map[string]interface{} `json:"auth,omitempty"`
}

// Path returns the path to .swytchcode/integrations/manifest.json.
func Path(projectRoot string) string {
	return util.ManifestPath(projectRoot)
}

// Read reads the manifest.json file.
// Returns nil map if file is missing or empty.
func Read(projectRoot string) (map[string]Entry, error) {
	data, err := os.ReadFile(Path(projectRoot))
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]Entry), nil
		}
		return nil, err
	}
	var m map[string]Entry
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = make(map[string]Entry)
	}
	return m, nil
}

// Write writes the manifest map to manifest.json.
func Write(projectRoot string, manifest map[string]Entry) error {
	if manifest == nil {
		manifest = make(map[string]Entry)
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	p := Path(projectRoot)
	if err := os.MkdirAll(util.IntegrationsDir(projectRoot), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// UpdateEntry updates or creates a manifest entry for project.library.
func UpdateEntry(projectRoot, projectLibrary, version, sandboxEndpoint, productionEndpoint string, methodsCount, workflowsCount int, auth map[string]interface{}) error {
	if projectLibrary == "" {
		return fmt.Errorf("project.library name cannot be empty")
	}
	if version == "" {
		return fmt.Errorf("version cannot be empty for %q", projectLibrary)
	}

	manifest, err := Read(projectRoot)
	if err != nil {
		return err
	}

	manifest[projectLibrary] = Entry{
		Version:           version,
		SandboxEndpoint:   sandboxEndpoint,
		ProductionEndpoint: productionEndpoint,
		Methods:           methodsCount,
		Workflows:         workflowsCount,
		Auth:              auth,
	}

	return Write(projectRoot, manifest)
}
