// upgrade.go implements swytchcode upgrade: refreshes a Wrekenfile from the registry (local convenience only; not for CI).
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/registry"
	"gitlab.com/swytchcode/shell/internal/util"
)

var (
	upgradeAutoYes        bool
	upgradeNonInteractive bool
)

// upgradeCmd implements `swytchcode upgrade <library>`, which refreshes
// an existing Wrekenfile spec from the registry.
//
// Semantics:
//   - Logically equivalent to "get, but require that the spec already
//     exists and treat the operation as an explicit upgrade".
//
// Interaction rules:
//   - On a TTY without --non-interactive, upgrade MAY prompt for
//     confirmation before overwriting. Until prompts are implemented,
//     we require --yes to proceed.
//   - In non-interactive mode (--non-interactive or no TTY), upgrade
//     must not prompt and requires --yes to overwrite.
var upgradeCmd = &cobra.Command{
	Use:   "upgrade <library>",
	Short: "Upgrade a Wrekenfile spec for a library",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("library name required")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		library := args[0]
		_ = util.IsInteractive() // reserved for future prompt behavior

		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return fmt.Errorf("detect project root: %w", err)
		}

		wrekenPath := filepath.Join(projectRoot, ".swytchcode", "wrekenfiles", library+".yaml")

		// Verify existing spec exists
		if _, err := os.Stat(wrekenPath); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("no Wrekenfile found for %q; use 'swytchcode get %s' to fetch it first", library, library)
		} else if err != nil {
			return fmt.Errorf("stat Wrekenfile: %w", err)
		}

		interactive := util.IsInteractive() && !upgradeNonInteractive
		if !upgradeAutoYes {
			if interactive {
				// TODO: Implement interactive prompt for overwrite confirmation
				return errors.New("upgrade requires --yes (interactive confirmation not yet implemented)")
			} else {
				return errors.New("upgrade requires --yes")
			}
		}

		// Load project_name from tooling.json, or use library name as fallback
		toolingPath := filepath.Join(projectRoot, ".swytchcode", "tooling.json")
		var tooling map[string]interface{}
		var projectName string
		if data, err := os.ReadFile(toolingPath); err == nil {
			if err := json.Unmarshal(data, &tooling); err == nil {
				if name, ok := tooling["project_name"].(string); ok && name != "" {
					projectName = name
				}
			}
		}
		// Fallback to library name if project_name not set (common case: project_name matches library name)
		if projectName == "" {
			projectName = library
		}

		// Fetch latest bundle from registry (base URL from tooling.json or env)
		regClient := registry.NewClient(registry.ConfigFromProjectRoot(projectRoot))
		ctx := context.Background()

		// Fetch latest version from integrations list, then fetch bundle
		listResp, err := regClient.ListIntegrations(ctx)
		if err != nil {
			return fmt.Errorf("fetch available integrations: %w", err)
		}
		
		var latestVersion string
		for _, integration := range listResp.Integrations {
			if integration.ID == library {
				latestVersion = integration.LatestVersion
				break
			}
		}
		if latestVersion == "" {
			return fmt.Errorf("integration %q not found", library)
		}

		bundle, err := regClient.GetIntegrationBundleVersion(ctx, projectName, library, latestVersion)
		if err != nil {
			return fmt.Errorf("fetch integration bundle: %w", err)
		}

		// Atomic overwrite: write to temp file, then rename (decode base64 if API returned encoded content)
		tmpPath := wrekenPath + ".tmp"
		wrekenBytes := util.DecodeBase64OrRaw(bundle.Files.Wreken.Content)
		if err := os.WriteFile(tmpPath, wrekenBytes, 0o644); err != nil {
			return fmt.Errorf("write Wrekenfile: %w", err)
		}

		if err := os.Rename(tmpPath, wrekenPath); err != nil {
			os.Remove(tmpPath) // Clean up on failure
			return fmt.Errorf("replace Wrekenfile: %w", err)
		}

		if err := updateWrekenManifest(projectRoot, library, bundle.Version); err != nil {
			return fmt.Errorf("update wreken manifest: %w", err)
		}

		// Remove proposals for this library so they are not stale (user can create a new proposal against new version).
		if n, errProposals := removeProposalsForLibrary(projectRoot, library); errProposals != nil {
			// Non-fatal: upgrade succeeded; log and continue
			if interactive {
				fmt.Printf("Upgraded Wrekenfile for %s to version %s (could not remove %d proposal(s): %v)\n", library, bundle.Version, n, errProposals)
			}
		} else if n > 0 && interactive {
			fmt.Printf("Removed %d stale proposal(s) for %s\n", n, library)
		}

		// TODO: Validate schema via internal/wreken (once validator is implemented)

		if interactive {
			fmt.Printf("Upgraded Wrekenfile for %s to version %s\n", library, bundle.Version)
		}

		return nil
	},
}

func init() {
	upgradeCmd.Flags().BoolVar(&upgradeAutoYes, "yes", false, "auto-confirm overwrite during upgrade")
	upgradeCmd.Flags().BoolVar(&upgradeNonInteractive, "non-interactive", false, "disable prompts; suitable for CI")
}

