// get.go implements swytchcode get: fetches and installs a Wrekenfile for an integration (exploratory only; does not modify tooling.json).
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

		// Show spinner while fetching bundles
		spinner := util.NewSpinner(fmt.Sprintf("Fetching %s...", projectName))
		spinner.Start()

		// Fetch all bundles for this project
		bundlesResp, err := regClient.GetIntegrationBundles(ctx, projectName)
		if err != nil {
			spinner.StopWithMessage(fmt.Sprintf("✗ Failed to fetch integration bundles: %v", err))
			return fmt.Errorf("fetch integration bundles: %w", err)
		}

		if bundlesResp == nil || len(bundlesResp.Bundles) == 0 {
			spinner.StopWithMessage(fmt.Sprintf("✗ No bundles found for project %q", projectName))
			return fmt.Errorf("no bundles found for project %q", projectName)
		}

		spinner.StopWithMessage(fmt.Sprintf("✓ Found %d bundle(s)", len(bundlesResp.Bundles)))

		// Create base .swytchcode directory and integrations subdirectory
		swytchDir := filepath.Join(projectRoot, ".swytchcode")
		if err := util.EnsureDir(swytchDir, 0o755); err != nil {
			return fmt.Errorf("create .swytchcode directory: %w", err)
		}
		integrationsDir := filepath.Join(swytchDir, "integrations")
		if err := util.EnsureDir(integrationsDir, 0o755); err != nil {
			return fmt.Errorf("create integrations directory: %w", err)
		}

		// Fetch workflows and methods for this project (needed for counts and manifest)
		spinner = util.NewSpinner("Fetching workflows and methods...")
		spinner.Start()

		workflowsResp, err := regClient.ListWorkflows(ctx, projectName)
		if err != nil {
			spinner.StopWithMessage(fmt.Sprintf("✗ Failed to fetch workflows: %v", err))
			return fmt.Errorf("fetch workflows for project %q: %w", projectName, err)
		}
		methodsResp, err := regClient.ListMethods(ctx, projectName)
		if err != nil {
			spinner.StopWithMessage(fmt.Sprintf("✗ Failed to fetch methods: %v", err))
			return fmt.Errorf("fetch methods for project %q: %w", projectName, err)
		}

		spinner.Stop()

		methodsCount := 0
		workflowsCount := 0
		if methodsResp != nil {
			methodsCount = len(methodsResp.Methods)
		}
		if workflowsResp != nil {
			workflowsCount = len(workflowsResp.Workflows)
		}

		// Save each bundle to .swytchcode/integrations/{project}/{library}/{version}/ structure
		savedCount := 0
		for i, bundle := range bundlesResp.Bundles {
			bundleSpinner := util.NewSpinner(fmt.Sprintf("Saving bundle %d/%d (%s@%s)...", i+1, len(bundlesResp.Bundles), bundle.Integration, bundle.Version))
			bundleSpinner.Start()

			if bundle.Integration == "" {
				bundleSpinner.StopWithMessage(fmt.Sprintf("✗ Bundle %d has empty integration name", i+1))
				return fmt.Errorf("bundle has empty integration name")
			}
			if bundle.Version == "" {
				bundleSpinner.StopWithMessage(fmt.Sprintf("✗ Bundle %q has empty version", bundle.Integration))
				return fmt.Errorf("bundle for integration %q has empty version", bundle.Integration)
			}

			// Create versioned directory: .swytchcode/integrations/{project}/{library}/{version}/
			versionedDir := filepath.Join(integrationsDir, projectName, bundle.Integration, bundle.Version)
			if err := util.EnsureDir(versionedDir, 0o755); err != nil {
				bundleSpinner.StopWithMessage(fmt.Sprintf("✗ Failed to create directory: %v", err))
				return fmt.Errorf("create versioned directory: %w", err)
			}

			wrekenPath := filepath.Join(versionedDir, "wrekenfile.yaml")
			methodsPath := filepath.Join(versionedDir, "methods.json")
			workflowsPath := filepath.Join(versionedDir, "workflows.json")

			// Check if directory already exists
			exists := false
			if _, err := os.Stat(versionedDir); err == nil {
				if _, err := os.Stat(wrekenPath); err == nil {
					exists = true
					if !getAutoYes {
						bundleSpinner.Stop()
						if interactive {
							return fmt.Errorf("Version %q for %s/%s already exists; use --yes to overwrite (interactive confirmation not yet implemented)", bundle.Version, projectName, bundle.Integration)
						} else {
							return fmt.Errorf("Version %q for %s/%s already exists; use --yes to overwrite", bundle.Version, projectName, bundle.Integration)
						}
					}
				}
			}

			// Validate and write Wrekenfile
			contentStr := bundle.Files.Wreken.Content
			if contentStr == "" {
				bundleSpinner.StopWithMessage(fmt.Sprintf("✗ Bundle %q@%q has empty Wrekenfile content", bundle.Integration, bundle.Version))
				return fmt.Errorf("bundle for integration %q has empty Wrekenfile content (version: %q)", bundle.Integration, bundle.Version)
			}

			wrekenBytes := util.DecodeBase64OrRaw(contentStr)
			if len(wrekenBytes) == 0 {
				bundleSpinner.StopWithMessage(fmt.Sprintf("✗ Decoded Wrekenfile content is empty for %q@%q", bundle.Integration, bundle.Version))
				return fmt.Errorf("decoded Wrekenfile content is empty for integration %q (original content length: %d)", bundle.Integration, len(contentStr))
			}

			if err := os.WriteFile(wrekenPath, wrekenBytes, 0o644); err != nil {
				bundleSpinner.StopWithMessage(fmt.Sprintf("✗ Failed to write Wrekenfile: %v", err))
				return fmt.Errorf("write Wrekenfile %q: %w", wrekenPath, err)
			}

			// Write methods.json (only for this library's version)
			// Filter methods by library if needed, or save all methods for the project
			// For now, save all methods for the project (they're project-scoped)
			if methodsResp != nil {
				data, err := json.MarshalIndent(methodsResp, "", "  ")
				if err != nil {
					bundleSpinner.StopWithMessage(fmt.Sprintf("✗ Failed to marshal methods: %v", err))
					return fmt.Errorf("marshal methods response: %w", err)
				}
				if err := os.WriteFile(methodsPath, data, 0o644); err != nil {
					bundleSpinner.StopWithMessage(fmt.Sprintf("✗ Failed to write methods.json: %v", err))
					return fmt.Errorf("write methods file %q: %w", methodsPath, err)
				}
			}

			// Write workflows.json
			if workflowsResp != nil {
				data, err := json.MarshalIndent(workflowsResp, "", "  ")
				if err != nil {
					bundleSpinner.StopWithMessage(fmt.Sprintf("✗ Failed to marshal workflows: %v", err))
					return fmt.Errorf("marshal workflows response: %w", err)
				}
				if err := os.WriteFile(workflowsPath, data, 0o644); err != nil {
					bundleSpinner.StopWithMessage(fmt.Sprintf("✗ Failed to write workflows.json: %v", err))
					return fmt.Errorf("write workflows file %q: %w", workflowsPath, err)
				}
			}

			// Update manifest.json with project.library entry
			projectLibrary := fmt.Sprintf("%s.%s", projectName, bundle.Integration)
			
			// Use endpoints directly from bundle (use http://localhost if empty)
			sandboxEndpoint := bundle.SandboxEndpoint
			productionEndpoint := bundle.ProductionEndpoint
			if sandboxEndpoint == "" {
				sandboxEndpoint = "http://localhost"
			}
			if productionEndpoint == "" {
				productionEndpoint = "http://localhost"
			}
			
			// TODO: Extract auth info from bundle or API - for now use empty
			auth := make(map[string]interface{})
			if err := updateManifestEntry(projectRoot, projectLibrary, bundle.Version, sandboxEndpoint, productionEndpoint, methodsCount, workflowsCount, auth); err != nil {
				bundleSpinner.StopWithMessage(fmt.Sprintf("✗ Failed to update manifest: %v", err))
				return fmt.Errorf("update manifest: %w", err)
			}

			savedCount++
			bundleSpinner.StopWithMessage(fmt.Sprintf("✓ Saved %s/%s@%s", projectName, bundle.Integration, bundle.Version))

			if interactive {
				if exists {
					fmt.Printf("Updated %s/%s@%s (wrekenfile.yaml, methods.json, workflows.json)\n", projectName, bundle.Integration, bundle.Version)
				} else {
					fmt.Printf("Added %s/%s@%s (wrekenfile.yaml, methods.json, workflows.json)\n", projectName, bundle.Integration, bundle.Version)
				}
			}
		}

		if interactive {
			if savedCount > 0 {
				fmt.Printf("Saved %d bundle(s) for project %q\n", savedCount, projectName)
			}
		}

		return nil
	},
}

func init() {
	getCmd.Flags().BoolVar(&getAutoYes, "yes", false, "auto-confirm overwrite in non-interactive mode")
	getCmd.Flags().BoolVar(&getNonInteractive, "non-interactive", false, "disable prompts; suitable for CI")
}
