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

		// Use library argument as project_name for API call
		projectName := library

		// Fetch all bundles for this project
		bundlesResp, err := regClient.GetIntegrationBundles(ctx, projectName)
		if err != nil {
			return fmt.Errorf("fetch integration bundles: %w", err)
		}

		if bundlesResp == nil || len(bundlesResp.Bundles) == 0 {
			return fmt.Errorf("no bundles found for project %q", projectName)
		}

		// Create project-specific directory: wrekenfiles/{project_name}/
		swytchDir := filepath.Join(projectRoot, ".swytchcode")
		projectWrekenDir := filepath.Join(swytchDir, "wrekenfiles", projectName)
		if err := util.EnsureDir(projectWrekenDir, 0o755); err != nil {
			return fmt.Errorf("create wrekenfiles directory: %w", err)
		}

		// Save each bundle to wrekenfiles/{project_name}/{integration}.yaml
		savedCount := 0
		for _, bundle := range bundlesResp.Bundles {
			if bundle.Integration == "" {
				return fmt.Errorf("bundle has empty integration name")
			}
			if bundle.Version == "" {
				return fmt.Errorf("bundle for integration %q has empty version", bundle.Integration)
			}

			wrekenPath := filepath.Join(projectWrekenDir, bundle.Integration+".yaml")

			// Check if file already exists
			exists := false
			if _, err := os.Stat(wrekenPath); err == nil {
				exists = true
				if !getAutoYes {
					if interactive {
						// TODO: Implement interactive prompt for overwrite confirmation
						return fmt.Errorf("Wrekenfile %q already exists; use --yes to overwrite (interactive confirmation not yet implemented)", wrekenPath)
					} else {
						return fmt.Errorf("Wrekenfile %q already exists; use --yes to overwrite", wrekenPath)
					}
				}
			}

			// Validate content is not empty
			contentStr := bundle.Files.Wreken.Content
			if contentStr == "" {
				return fmt.Errorf("bundle for integration %q has empty Wrekenfile content (version: %q)", bundle.Integration, bundle.Version)
			}

			// Decode base64 content and write to file
			wrekenBytes := util.DecodeBase64OrRaw(contentStr)
			if len(wrekenBytes) == 0 {
				return fmt.Errorf("decoded Wrekenfile content is empty for integration %q (original content length: %d)", bundle.Integration, len(contentStr))
			}

			if err := os.WriteFile(wrekenPath, wrekenBytes, 0o644); err != nil {
				return fmt.Errorf("write Wrekenfile %q: %w", wrekenPath, err)
			}

			// Record installed version in manifest
			if err := updateWrekenManifest(projectRoot, bundle.Integration, bundle.Version); err != nil {
				return fmt.Errorf("update wreken manifest: %w", err)
			}

			savedCount++

			if interactive {
				if exists {
					fmt.Printf("Updated Wrekenfile for %s/%s (version %s)\n", projectName, bundle.Integration, bundle.Version)
				} else {
					fmt.Printf("Added Wrekenfile for %s/%s (version %s)\n", projectName, bundle.Integration, bundle.Version)
				}
			}
		}

		if interactive {
			fmt.Printf("Saved %d bundle(s) for project %q\n", savedCount, projectName)
		}

		return nil
	},
}

func init() {
	getCmd.Flags().BoolVar(&getAutoYes, "yes", false, "auto-confirm overwrite in non-interactive mode")
	getCmd.Flags().BoolVar(&getNonInteractive, "non-interactive", false, "disable prompts; suitable for CI")
}

