// bootstrap.go provides shared bootstrap command logic for CLI and MCP.
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/swytchcode/shell/internal/manifest"
	"gitlab.com/swytchcode/shell/internal/registry"
	"gitlab.com/swytchcode/shell/internal/util"
)

// RunBootstrap runs the bootstrap command: fetches all integrations declared in tooling.json.
func RunBootstrap(ctx context.Context, projectRoot string, stdout, stderr io.Writer) error {
	// Load tooling.json
	toolingPath := filepath.Join(projectRoot, ".swytchcode", "tooling.json")
	data, err := os.ReadFile(toolingPath)
	if err != nil {
		return fmt.Errorf("tooling.json not found; run 'swytchcode init' first: %w", err)
	}

	var tooling map[string]interface{}
	if err := json.Unmarshal(data, &tooling); err != nil {
		return fmt.Errorf("parse tooling.json: %w", err)
	}

	// Get integrations section
	integrationsRaw, ok := tooling["integrations"]
	if !ok {
		fmt.Fprintln(stdout, "No integrations found in tooling.json")
		return nil
	}

	integrations, ok := integrationsRaw.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid integrations format in tooling.json")
	}

	if len(integrations) == 0 {
		fmt.Fprintln(stdout, "No integrations found in tooling.json")
		return nil
	}

	// Ensure integrations directory exists
	integrationsDir := filepath.Join(projectRoot, ".swytchcode", "integrations")
	if err := util.EnsureDir(integrationsDir, 0o755); err != nil {
		return fmt.Errorf("create integrations directory: %w", err)
	}

	regClient := registry.NewClient(registry.DefaultConfig())

	// Process each integration
	var fetched []string
	var skipped []string
	var failed []string

	for projectLibrary, versionRaw := range integrations {
		// Parse project.library format
		parts := strings.SplitN(projectLibrary, ".", 2)
		if len(parts) != 2 {
			failed = append(failed, fmt.Sprintf("%s (invalid format)", projectLibrary))
			continue
		}
		projectName := parts[0]
		libraryName := parts[1]

		// Get version
		versionMap, ok := versionRaw.(map[string]interface{})
		if !ok {
			failed = append(failed, fmt.Sprintf("%s (invalid version format)", projectLibrary))
			continue
		}
		version, ok := versionMap["version"].(string)
		if !ok || version == "" {
			failed = append(failed, fmt.Sprintf("%s (missing version)", projectLibrary))
			continue
		}

		// Check if integration already exists
		integrationPath := filepath.Join(integrationsDir, projectName, libraryName, version)
		if _, err := os.Stat(integrationPath); err == nil {
			// Check if wrekenfile exists
			wrekenPath := filepath.Join(integrationPath, "wrekenfile.yaml")
			if _, err := os.Stat(wrekenPath); err == nil {
				skipped = append(skipped, projectLibrary)
				continue
			}
		}

		// Fetch the integration
		spinner := util.NewSpinner(fmt.Sprintf("Fetching %s...", projectLibrary))
		spinner.Start()

		if err := fetchIntegration(ctx, regClient, projectRoot, projectName, libraryName, version); err != nil {
			spinner.Stop()
			failed = append(failed, fmt.Sprintf("%s: %v", projectLibrary, err))
			continue
		}

		spinner.StopWithMessage(fmt.Sprintf("✓ Fetched %s@%s\n", projectLibrary, version))
		fetched = append(fetched, projectLibrary)
	}

	// Print summary
	if len(fetched) > 0 {
		fmt.Fprintf(stdout, "\nFetched %d integration(s):\n", len(fetched))
		for _, name := range fetched {
			fmt.Fprintf(stdout, "  - %s\n", name)
		}
	}
	if len(skipped) > 0 {
		fmt.Fprintf(stdout, "\nSkipped %d integration(s) (already installed):\n", len(skipped))
		for _, name := range skipped {
			fmt.Fprintf(stdout, "  - %s\n", name)
		}
	}
	if len(failed) > 0 {
		fmt.Fprintf(stderr, "\nFailed to fetch %d integration(s):\n", len(failed))
		for _, name := range failed {
			fmt.Fprintf(stderr, "  - %s\n", name)
		}
		return fmt.Errorf("bootstrap failed for %d integration(s)", len(failed))
	}

	return nil
}

// fetchIntegration fetches a single integration using the registry client.
func fetchIntegration(ctx context.Context, regClient *registry.Client, projectRoot, projectName, libraryName, version string) error {
	// Fetch all bundles for this project
	bundlesResp, err := regClient.GetIntegrationBundles(ctx, projectName)
	if err != nil {
		return fmt.Errorf("fetch integration bundles: %w", err)
	}

	if bundlesResp == nil || len(bundlesResp.Bundles) == 0 {
		return fmt.Errorf("no bundles found for project %q", projectName)
	}

	// Find the bundle matching library and version
	var targetBundle *registry.IntegrationBundle
	for i := range bundlesResp.Bundles {
		bundle := bundlesResp.Bundles[i]
		if bundle.Integration == libraryName && bundle.Version == version {
			targetBundle = &bundle
			break
		}
	}

	if targetBundle == nil {
		return fmt.Errorf("bundle %s@%s not found", libraryName, version)
	}

	// Ensure directories exist
	integrationsDir := filepath.Join(projectRoot, ".swytchcode", "integrations")
	versionedDir := filepath.Join(integrationsDir, projectName, libraryName, version)
	if err := util.EnsureDir(versionedDir, 0o755); err != nil {
		return fmt.Errorf("create versioned directory: %w", err)
	}

	// Write wrekenfile
	wrekenPath := filepath.Join(versionedDir, "wrekenfile.yaml")
	wrekenBytes := util.DecodeBase64OrRaw(targetBundle.Files.Wreken.Content)
	if len(wrekenBytes) == 0 {
		return fmt.Errorf("empty Wrekenfile content")
	}
	if err := os.WriteFile(wrekenPath, wrekenBytes, 0o644); err != nil {
		return fmt.Errorf("write Wrekenfile: %w", err)
	}

	// Fetch workflows and methods
	workflowsResp, err := regClient.ListWorkflows(ctx, projectName)
	if err != nil {
		return fmt.Errorf("fetch workflows: %w", err)
	}
	registry.FillEmptyWorkflowNames(workflowsResp)
	methodsResp, err := regClient.ListMethods(ctx, projectName)
	if err != nil {
		return fmt.Errorf("fetch methods: %w", err)
	}

	// Write methods.json
	if methodsResp != nil {
		methodsPath := filepath.Join(versionedDir, "methods.json")
		data, err := json.MarshalIndent(methodsResp, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal methods: %w", err)
		}
		if err := os.WriteFile(methodsPath, data, 0o644); err != nil {
			return fmt.Errorf("write methods file: %w", err)
		}
	}

	// Write workflows.json
	if workflowsResp != nil {
		workflowsPath := filepath.Join(versionedDir, "workflows.json")
		data, err := json.MarshalIndent(workflowsResp, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal workflows: %w", err)
		}
		if err := os.WriteFile(workflowsPath, data, 0o644); err != nil {
			return fmt.Errorf("write workflows file: %w", err)
		}
	}

	// Update manifest.json
	methodsCount := 0
	workflowsCount := 0
	if methodsResp != nil {
		methodsCount = len(methodsResp.Methods)
	}
	if workflowsResp != nil {
		workflowsCount = len(workflowsResp.Workflows)
	}

	projectLibrary := fmt.Sprintf("%s.%s", projectName, libraryName)
	sandboxEndpoint := targetBundle.SandboxEndpoint
	productionEndpoint := targetBundle.ProductionEndpoint
	if sandboxEndpoint == "" {
		sandboxEndpoint = "http://localhost"
	}
	if productionEndpoint == "" {
		productionEndpoint = "http://localhost"
	}

	auth := make(map[string]interface{})
	if err := manifest.UpdateEntry(projectRoot, projectLibrary, version, sandboxEndpoint, productionEndpoint, methodsCount, workflowsCount, auth); err != nil {
		return fmt.Errorf("update manifest: %w", err)
	}

	return nil
}
