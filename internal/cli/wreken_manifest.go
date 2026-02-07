// wreken_manifest.go manages .swytchcode/wrekenfiles/manifest.json (index of installed Wrekenfiles for list/describe).
package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const wrekenManifestName = "manifest.json"

// wrekenManifestPath returns the path to .swytchcode/wrekenfiles/manifest.json.
func wrekenManifestPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".swytchcode", "wrekenfiles", wrekenManifestName)
}

// readWrekenManifest reads installed integration versions from .swytchcode/wrekenfiles/manifest.json.
// Returns nil map if file is missing or empty (no integrations installed).
func readWrekenManifest(projectRoot string) (map[string]string, error) {
	data, err := os.ReadFile(wrekenManifestPath(projectRoot))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = make(map[string]string)
	}
	return m, nil
}

// writeWrekenManifest writes the installed-versions map to manifest.json.
func writeWrekenManifest(projectRoot string, installed map[string]string) error {
	if installed == nil {
		installed = make(map[string]string)
	}
	data, err := json.MarshalIndent(installed, "", "  ")
	if err != nil {
		return err
	}
	p := wrekenManifestPath(projectRoot)
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// updateWrekenManifest sets one integration's installed version and saves the manifest.
func updateWrekenManifest(projectRoot, integration, version string) error {
	m, err := readWrekenManifest(projectRoot)
	if err != nil {
		return err
	}
	if m == nil {
		m = make(map[string]string)
	}
	m[integration] = version
	return writeWrekenManifest(projectRoot, m)
}
