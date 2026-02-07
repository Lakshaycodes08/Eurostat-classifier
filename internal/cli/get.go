// get.go implements swytchcode get: fetches and installs a Wrekenfile for an integration (exploratory only; does not modify tooling.json).
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/registry"
	"gitlab.com/swytchcode/shell/internal/util"
)

var (
	getAutoYes        bool
	getNonInteractive bool
)

// getCmd implements `swytchcode get`.
//
// Interaction rules:
//   - On a TTY without --non-interactive, get MAY prompt (library selection,
//     overwrite confirmation).
//   - In non-interactive mode (--non-interactive or no TTY), it must not
//     prompt and should rely on flags such as --yes.
var getCmd = &cobra.Command{
	Use:   "get [library]",
	Short: "Fetch Wrekenfile for a library",
	Args: func(cmd *cobra.Command, args []string) error {
		interactive := util.IsInteractive() && !getNonInteractive
		if len(args) == 0 && !interactive {
			// Non-interactive usage requires an explicit library name.
			return errors.New("library name required when running non-interactively")
		}
		// When interactive, zero args are allowed; the actual prompt is handled in RunE.
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		interactive := util.IsInteractive() && !getNonInteractive

		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return fmt.Errorf("detect project root: %w", err)
		}
		regClient := registry.NewClient(registry.ConfigFromProjectRoot(projectRoot))
		ctx := context.Background()

		var library string
		if len(args) > 0 {
			library = args[0]
		} else if interactive {
			// Interactive selection: fetch available integrations from registry

			listResp, err := regClient.ListIntegrations(ctx)
			if err != nil {
				return fmt.Errorf("fetch available integrations: %w", err)
			}

			if len(listResp.Integrations) == 0 {
				return errors.New("no integrations available")
			}

			options := make([]string, len(listResp.Integrations))
			for i, integration := range listResp.Integrations {
				options[i] = integration.ID
			}

			fmt.Println()
			_, library = util.SelectWithRetry("Which library do you want to add?", options)
		}

		if library == "" {
			return errors.New("library name required")
		}

		swytchDir := filepath.Join(projectRoot, ".swytchcode")
		wrekenDir := filepath.Join(swytchDir, "wrekenfiles")
		if err := util.EnsureDir(wrekenDir, 0o755); err != nil {
			return fmt.Errorf("create wrekenfiles directory: %w", err)
		}

		wrekenPath := filepath.Join(wrekenDir, library+".yaml")

		// Check if file already exists
		exists := false
		if _, err := os.Stat(wrekenPath); err == nil {
			exists = true
			if !getAutoYes {
				if interactive {
					// TODO: Implement interactive prompt for overwrite confirmation
					return errors.New("Wrekenfile already exists; use --yes to overwrite (interactive confirmation not yet implemented)")
				} else {
					return errors.New("Wrekenfile already exists; use --yes to overwrite")
				}
			}
		}

		// Fetch bundle from registry
		bundle, err := regClient.GetIntegrationBundle(ctx, library)
		if err != nil {
			return fmt.Errorf("fetch integration bundle: %w", err)
		}

		// Write Wrekenfile (YAML); decode base64 if the API returned encoded content
		wrekenBytes := util.DecodeBase64OrRaw(bundle.Files.Wreken.Content)
		if err := os.WriteFile(wrekenPath, wrekenBytes, 0o644); err != nil {
			return fmt.Errorf("write Wrekenfile: %w", err)
		}

		// Record installed version in manifest (get is for exploration; tooling.json pins are set via add integration / apply)
		if err := updateWrekenManifest(projectRoot, library, bundle.Version); err != nil {
			return fmt.Errorf("update wreken manifest: %w", err)
		}

		// TODO: Validate schema via internal/wreken (once validator is implemented)

		if interactive {
			if exists {
				fmt.Printf("Updated Wrekenfile for %s (version %s)\n", library, bundle.Version)
			} else {
				fmt.Printf("Added Wrekenfile for %s (version %s)\n", library, bundle.Version)
			}
		}

		return nil
	},
}

func init() {
	getCmd.Flags().BoolVar(&getAutoYes, "yes", false, "auto-confirm overwrite in non-interactive mode")
	getCmd.Flags().BoolVar(&getNonInteractive, "non-interactive", false, "disable prompts; suitable for CI")
}

