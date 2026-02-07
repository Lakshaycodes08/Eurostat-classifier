// config_cmd.go implements swytchcode config: prints effective configuration (registry URL and source) without mutating anything.
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/registry"
	"gitlab.com/swytchcode/shell/internal/util"
)

// configCmd implements `swytchcode config`: show effective config so env overrides are visible.
// Environment variable overrides must not be silently persisted; this command makes them visible.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show effective configuration (registry_url source: env vs tooling)",
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

		var tooling map[string]interface{}
		if err := json.Unmarshal(data, &tooling); err != nil {
			return fmt.Errorf("parse tooling.json: %w", err)
		}

		effectiveURL, source := registry.RegistryURLEffectiveAndSource(projectRoot)
		out := map[string]interface{}{
			"registry_url": map[string]string{
				"effective": effectiveURL,
				"source":    source,
			},
		}
		if v, ok := tooling["mode"]; ok {
			out["mode"] = v
		}
		if v, ok := tooling["version"]; ok {
			out["version"] = v
		}

		return util.WriteJSON(os.Stdout, out)
	},
}
