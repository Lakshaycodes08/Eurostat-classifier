// bootstrap.go implements swytchcode bootstrap: installs exact integration versions from tooling.json into the local cache.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/registry"
	"gitlab.com/swytchcode/shell/internal/util"
)

// bootstrapCmd implements `swytchcode bootstrap`: install exact integration versions from tooling.json.
//
// Invariant: tooling.json pins what is trusted. The registry supplies how it works.
// bootstrap reconciles the two. exec only executes.
//
// Never installs "latest"; never mutates tooling.json; never runs during exec.
var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Install exact integration versions declared in tooling.json",
	Long:  "Reads integration versions from tooling.json, downloads exact versions from the registry. Fails if installed version does not match. Never modifies tooling.json.",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return fmt.Errorf("detect project root: %w", err)
		}

		toolingPath := filepath.Join(projectRoot, ".swytchcode", "tooling.json")
		data, err := os.ReadFile(toolingPath)
		if err != nil {
			return fmt.Errorf("tooling.json not found; run 'swytchcode init' first: %w", err)
		}

		var tooling struct {
			Integrations map[string]struct {
				Version string `json:"version"`
			} `json:"integrations"`
		}
		if err := json.Unmarshal(data, &tooling); err != nil {
			return fmt.Errorf("parse tooling.json: %w", err)
		}
		if tooling.Integrations == nil {
			return nil // no integrations to bootstrap
		}

		installed, err := readWrekenManifest(projectRoot)
		if err != nil {
			return fmt.Errorf("read wreken manifest: %w", err)
		}
		if installed == nil {
			installed = make(map[string]string)
		}

		regClient := registry.NewClient(registry.ConfigFromProjectRoot(projectRoot))
		ctx := context.Background()
		wrekenDir := filepath.Join(projectRoot, ".swytchcode", "wrekenfiles")
		if err := os.MkdirAll(wrekenDir, 0o755); err != nil {
			return fmt.Errorf("create wrekenfiles directory: %w", err)
		}

		for name, spec := range tooling.Integrations {
			required := spec.Version
			if required == "" {
				writeBootstrapError(os.Stderr, fmt.Sprintf("Integration %q in tooling.json has no version", name))
				return fmt.Errorf("integration %q has no version in tooling.json", name)
			}

			current, ok := installed[name]
			if !ok {
				// Not installed: fetch exact version from registry
				bundle, err := regClient.GetIntegrationBundleVersion(ctx, name, required)
				if err != nil {
					writeBootstrapError(os.Stderr, fmt.Sprintf("Integration %q@%s not installed. Run `swytchcode bootstrap`.", name, required))
					return fmt.Errorf("fetch %s@%s: %w", name, required, err)
				}
				wrekenPath := filepath.Join(wrekenDir, name+".yaml")
				wrekenBytes := util.DecodeBase64OrRaw(bundle.Files.Wreken.Content)
				if err := os.WriteFile(wrekenPath, wrekenBytes, 0o644); err != nil {
					return fmt.Errorf("write Wrekenfile for %s: %w", name, err)
				}
				if err := updateWrekenManifest(projectRoot, name, bundle.Version); err != nil {
					return fmt.Errorf("update manifest: %w", err)
				}
				installed[name] = bundle.Version
				continue
			}

			if current != required {
				writeBootstrapError(os.Stderr, fmt.Sprintf("Installed %s@%s does not match tooling.json (%s).", name, current, required))
				return fmt.Errorf("installed %s@%s does not match tooling.json (%s)", name, current, required)
			}
		}

		return nil
	},
}

func writeBootstrapError(w *os.File, msg string) {
	_ = util.WriteJSON(w, map[string]string{"error": msg})
}
