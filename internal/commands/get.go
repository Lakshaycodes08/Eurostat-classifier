// get.go provides shared get command logic for CLI and MCP.
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"gitlab.com/swytchcode/swytchcode-cli/internal/constants"
	"gitlab.com/swytchcode/swytchcode-cli/internal/manifest"
	"gitlab.com/swytchcode/swytchcode-cli/internal/output"
	"gitlab.com/swytchcode/swytchcode-cli/internal/registry"
	"gitlab.com/swytchcode/swytchcode-cli/internal/util"
)

// filterMethodsByWreken returns a new ListMethodsResponse containing only methods whose
// CanonicalID appears in the METHODS section of the given wrekenfile bytes.
// This ensures each library's methods.json only contains its own methods, not the full project list.
func filterMethodsByWreken(wrekenBytes []byte, all *registry.ListMethodsResponse) *registry.ListMethodsResponse {
	if all == nil {
		return nil
	}
	var wreken map[string]interface{}
	if err := yaml.Unmarshal(wrekenBytes, &wreken); err != nil {
		return all // can't parse — return unfiltered
	}
	methodsSection, ok := wreken[constants.WrekenMethods].(map[string]interface{})
	if !ok {
		return all // no METHODS section — return unfiltered
	}
	allowed := make(map[string]bool, len(methodsSection))
	for id := range methodsSection {
		allowed[id] = true
	}
	filtered := make([]registry.Method, 0, len(all.Methods))
	for _, m := range all.Methods {
		if allowed[m.CanonicalID] {
			filtered = append(filtered, m)
		}
	}
	return &registry.ListMethodsResponse{Methods: filtered}
}

// RunGet runs the get command: fetch bundles for a project and write wrekenfile, methods, workflows, manifest.
func RunGet(ctx context.Context, projectName string, yes bool, stdout, stderr io.Writer) error {
	projectRoot, err := util.ProjectRoot()
	if err != nil {
		return fmt.Errorf("detect project root: %w", err)
	}

	regClient := registry.NewClient(registry.DefaultConfig())

	// Show spinner while fetching bundles
	spinner := util.NewSpinner(fmt.Sprintf("Fetching %s...", projectName))
	spinner.Start()

	// Fetch all bundles for this project
	bundlesResp, err := regClient.GetIntegrationBundles(ctx, projectName)
	if err != nil {
		spinner.Stop()
		output.Error(stderr, fmt.Sprintf("Failed to fetch integration bundles: %v", err))
		output.Hint(stderr, "check your network connection or run 'swytchcode login'")
		return fmt.Errorf("fetch integration bundles: %w", err)
	}

	if bundlesResp == nil || len(bundlesResp.Bundles) == 0 {
		spinner.Stop()
		output.Error(stderr, fmt.Sprintf("No bundles found for project %q", projectName))
		return fmt.Errorf("no bundles found for project %q", projectName)
	}

	spinner.StopWithMessage(fmt.Sprintf("✓ Found %d bundle(s)\n", len(bundlesResp.Bundles)))

	// Create base .swytchcode directory and integrations subdirectory
	if err := util.EnsureDir(util.SwytchDir(projectRoot), 0o755); err != nil {
		return fmt.Errorf("create .swytchcode directory: %w", err)
	}
	if err := util.EnsureDir(util.IntegrationsDir(projectRoot), 0o755); err != nil {
		return fmt.Errorf("create integrations directory: %w", err)
	}

	// Fetch workflows and methods for this project (needed for counts and manifest)
	spinner = util.NewSpinner("Fetching workflows and methods...")
	spinner.Start()

	workflowsResp, err := regClient.ListWorkflows(ctx, projectName)
	if err != nil {
		spinner.Stop()
		output.Error(stderr, fmt.Sprintf("Failed to fetch workflows: %v", err))
		return fmt.Errorf("fetch workflows for project %q: %w", projectName, err)
	}
	registry.FillEmptyWorkflowNames(workflowsResp)
	methodsResp, err := regClient.ListMethods(ctx, projectName)
	if err != nil {
		spinner.Stop()
		output.Error(stderr, fmt.Sprintf("Failed to fetch methods: %v", err))
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
			bundleSpinner.StopWithMessage(fmt.Sprintf("✗ Bundle %d has empty integration name\n", i+1))
			return fmt.Errorf("bundle has empty integration name")
		}
		if bundle.Version == "" {
			bundleSpinner.StopWithMessage(fmt.Sprintf("✗ Bundle %q has empty version\n", bundle.Integration))
			return fmt.Errorf("bundle for integration %q has empty version", bundle.Integration)
		}

		// Create versioned directory: .swytchcode/integrations/{project}/{library}/{version}/
		versionedDir := util.IntegrationVersionDir(projectRoot, projectName, bundle.Integration, bundle.Version)
		if err := util.EnsureDir(versionedDir, 0o755); err != nil {
			bundleSpinner.StopWithMessage(fmt.Sprintf("✗ Failed to create directory: %v\n", err))
			return fmt.Errorf("create versioned directory: %w", err)
		}

		wrekenPath := filepath.Join(versionedDir, constants.WrekenfileYAMLFile)
		methodsPath := filepath.Join(versionedDir, constants.MethodsJSONFile)
		workflowsPath := filepath.Join(versionedDir, constants.WorkflowsJSONFile)

		// Check if directory already exists
		if _, err := os.Stat(versionedDir); err == nil {
			if _, err := os.Stat(wrekenPath); err == nil {
				if !yes {
					bundleSpinner.Stop()
					return fmt.Errorf("Version %q for %s/%s already exists; use --yes to overwrite", bundle.Version, projectName, bundle.Integration)
				}
			}
		}

		// Validate and write Wrekenfile
		contentStr := bundle.Files.Wreken.Content
		if contentStr == "" {
			bundleSpinner.StopWithMessage(fmt.Sprintf("✗ Bundle %q@%q has empty Wrekenfile content\n", bundle.Integration, bundle.Version))
			return fmt.Errorf("bundle for integration %q has empty Wrekenfile content (version: %q)", bundle.Integration, bundle.Version)
		}

		wrekenBytes := util.DecodeBase64OrRaw(contentStr)
		if len(wrekenBytes) == 0 {
			bundleSpinner.StopWithMessage(fmt.Sprintf("✗ Decoded Wrekenfile content is empty for %q@%q\n", bundle.Integration, bundle.Version))
			return fmt.Errorf("decoded Wrekenfile content is empty for integration %q (original content length: %d)", bundle.Integration, len(contentStr))
		}

		if err := os.WriteFile(wrekenPath, wrekenBytes, 0o644); err != nil {
			bundleSpinner.StopWithMessage(fmt.Sprintf("✗ Failed to write Wrekenfile: %v\n", err))
			return fmt.Errorf("write Wrekenfile %q: %w", wrekenPath, err)
		}

		// Write methods.json — filtered to only this library's methods (by wrekenfile METHODS keys)
		if methodsResp != nil {
			data, err := json.MarshalIndent(filterMethodsByWreken(wrekenBytes, methodsResp), "", "  ")
			if err != nil {
				bundleSpinner.StopWithMessage(fmt.Sprintf("✗ Failed to marshal methods: %v\n", err))
				return fmt.Errorf("marshal methods response: %w", err)
			}
			if err := os.WriteFile(methodsPath, data, 0o644); err != nil {
				bundleSpinner.StopWithMessage(fmt.Sprintf("✗ Failed to write methods.json: %v\n", err))
				return fmt.Errorf("write methods file %q: %w", methodsPath, err)
			}
		}

		// Write workflows.json
		if workflowsResp != nil {
			data, err := json.MarshalIndent(workflowsResp, "", "  ")
			if err != nil {
				bundleSpinner.StopWithMessage(fmt.Sprintf("✗ Failed to marshal workflows: %v\n", err))
				return fmt.Errorf("marshal workflows response: %w", err)
			}
			if err := os.WriteFile(workflowsPath, data, 0o644); err != nil {
				bundleSpinner.StopWithMessage(fmt.Sprintf("✗ Failed to write workflows.json: %v\n", err))
				return fmt.Errorf("write workflows file %q: %w", workflowsPath, err)
			}
		}

		// Update manifest.json
		projectLibrary := fmt.Sprintf("%s.%s", projectName, bundle.Integration)

		// Use endpoints directly from bundle (fall back to DefaultLocalEndpoint if empty)
		sandboxEndpoint := bundle.SandboxEndpoint
		productionEndpoint := bundle.ProductionEndpoint
		if sandboxEndpoint == "" {
			sandboxEndpoint = constants.DefaultLocalEndpoint
		}
		if productionEndpoint == "" {
			productionEndpoint = constants.DefaultLocalEndpoint
		}

		auth := make(map[string]interface{})
		if err := manifest.UpdateEntry(projectRoot, projectLibrary, bundle.Version, sandboxEndpoint, productionEndpoint, methodsCount, workflowsCount, auth); err != nil {
			bundleSpinner.StopWithMessage(fmt.Sprintf("✗ Failed to update manifest: %v\n", err))
			return fmt.Errorf("update manifest: %w", err)
		}

		savedCount++
		bundleSpinner.StopWithMessage(fmt.Sprintf("✓ Saved %s/%s@%s\n", projectName, bundle.Integration, bundle.Version))
	}

	if savedCount > 0 {
		fmt.Fprintf(stdout, "Saved %d bundle(s) for project %q\n", savedCount, projectName)
	}

	return nil
}
