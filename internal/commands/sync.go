// sync.go provides shared sync command logic for CLI and MCP.
// It re-fetches workflow and method lists from the backend and updates local files,
// without modifying tooling.json (user still decides which tools to add).
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/swytchcode/swytchcode-cli/internal/constants"
	"gitlab.com/swytchcode/swytchcode-cli/internal/manifest"
	"gitlab.com/swytchcode/swytchcode-cli/internal/output"
	"gitlab.com/swytchcode/swytchcode-cli/internal/registry"
	"gitlab.com/swytchcode/swytchcode-cli/internal/util"
)

// RunSync re-fetches the workflow/method list from the backend for each installed project,
// compares with what's on disk, and downloads new or updated files.
// If projectName is empty, syncs all installed projects.
func RunSync(ctx context.Context, projectName string, stdout, stderr io.Writer) error {
	projectRoot, err := util.ProjectRoot()
	if err != nil {
		return fmt.Errorf("detect project root: %w", err)
	}

	integrationsDir := util.IntegrationsDir(projectRoot)
	if _, err := os.Stat(integrationsDir); err != nil {
		return fmt.Errorf("no integrations found — run: swytchcode get <project>")
	}

	var projectsToSync []string
	if projectName != "" {
		projectsToSync = []string{projectName}
	} else {
		entries, err := os.ReadDir(integrationsDir)
		if err != nil {
			return fmt.Errorf("read integrations directory: %w", err)
		}
		for _, e := range entries {
			if e.IsDir() {
				projectsToSync = append(projectsToSync, e.Name())
			}
		}
	}

	if len(projectsToSync) == 0 {
		fmt.Fprintln(stdout, "No projects installed.")
		return nil
	}

	regClient := registry.NewClient(registry.DefaultConfig())
	for _, proj := range projectsToSync {
		fmt.Fprintf(stdout, "Syncing project: %s\n", proj)
		if err := syncProject(ctx, regClient, projectRoot, proj, stdout, stderr); err != nil {
			output.Warn(stderr, fmt.Sprintf("  failed to sync %s: %v", proj, err))
		}
	}
	return nil
}

func syncProject(ctx context.Context, regClient *registry.Client, projectRoot, projectName string, stdout, stderr io.Writer) error {
	// Fetch remote workflow list
	remoteResp, err := regClient.ListWorkflows(ctx, projectName)
	if err != nil {
		return fmt.Errorf("fetch workflows from backend: %w", err)
	}
	registry.FillEmptyWorkflowNames(remoteResp)

	// Load local workflows from the first workflows.json found in this project's dir
	projDir := filepath.Join(util.IntegrationsDir(projectRoot), projectName)
	var localWorkflows []registry.Workflow
	_ = filepath.Walk(projDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || info.Name() != constants.WorkflowsJSONFile || localWorkflows != nil {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var resp registry.ListWorkflowsResponse
		if json.Unmarshal(data, &resp) == nil {
			localWorkflows = resp.Workflows
		}
		return nil
	})

	// Index local workflows by canonical_id
	localMap := make(map[string]registry.Workflow)
	for _, w := range localWorkflows {
		if w.CanonicalID != "" {
			localMap[w.CanonicalID] = w
		}
	}

	// Identify new and updated workflows
	var newWorkflows, updatedWorkflows []string
	for _, remote := range remoteResp.Workflows {
		if remote.CanonicalID == "" {
			continue
		}
		local, exists := localMap[remote.CanonicalID]
		if !exists {
			newWorkflows = append(newWorkflows, remote.CanonicalID)
		} else if workflowStepsChanged(local.Steps, remote.Steps) {
			updatedWorkflows = append(updatedWorkflows, remote.CanonicalID)
		}
	}

	if len(newWorkflows) == 0 && len(updatedWorkflows) == 0 {
		fmt.Fprintf(stdout, "  ✓ Already up to date\n")
		return nil
	}

	// Re-fetch all bundles for this project
	spinner := util.NewSpinner(fmt.Sprintf("  Downloading updates for %s...", projectName))
	spinner.Start()
	bundlesResp, err := regClient.GetIntegrationBundles(ctx, projectName)
	if err != nil {
		spinner.Stop()
		return fmt.Errorf("fetch integration bundles: %w", err)
	}
	spinner.StopWithMessage(fmt.Sprintf("  ✓ Downloaded %d bundle(s)\n", len(bundlesResp.Bundles)))

	workflowsData, marshalErr := json.MarshalIndent(remoteResp, "", "  ")

	for _, bundle := range bundlesResp.Bundles {
		if bundle.Integration == "" || bundle.Version == "" {
			continue
		}
		versionedDir := util.IntegrationVersionDir(projectRoot, projectName, bundle.Integration, bundle.Version)
		if err := util.EnsureDir(versionedDir, 0o755); err != nil {
			continue
		}

		wrekenBytes := util.DecodeBase64OrRaw(bundle.Files.Wreken.Content)
		if len(wrekenBytes) > 0 {
			if err := os.WriteFile(filepath.Join(versionedDir, constants.WrekenfileYAMLFile), wrekenBytes, 0o644); err != nil {
				output.Warn(stderr, fmt.Sprintf("  failed to write wrekenfile for %s: %v", bundle.Integration, err))
			}
		}

		if marshalErr == nil {
			if err := os.WriteFile(filepath.Join(versionedDir, constants.WorkflowsJSONFile), workflowsData, 0o644); err != nil {
				output.Warn(stderr, fmt.Sprintf("  failed to write workflows.json for %s: %v", bundle.Integration, err))
			}
		}

		sandboxEndpoint := bundle.SandboxEndpoint
		productionEndpoint := bundle.ProductionEndpoint
		if sandboxEndpoint == "" {
			sandboxEndpoint = constants.DefaultLocalEndpoint
		}
		if productionEndpoint == "" {
			productionEndpoint = constants.DefaultLocalEndpoint
		}
		projectLibrary := fmt.Sprintf("%s.%s", projectName, bundle.Integration)
		manifest.UpdateEntry(projectRoot, projectLibrary, bundle.Version, sandboxEndpoint, productionEndpoint, 0, len(remoteResp.Workflows), map[string]interface{}{})
	}

	// Report new workflows
	if len(newWorkflows) > 0 {
		fmt.Fprintf(stdout, "  ✓ %d new workflow(s) available: %s\n", len(newWorkflows), strings.Join(newWorkflows, ", "))
	}

	// Warn about updated workflows already in tooling.json
	var toolingTools map[string]interface{}
	if tooling, err := util.LoadToolingJSON(projectRoot); err == nil {
		toolingTools, _ = tooling["tools"].(map[string]interface{})
	}

	if len(updatedWorkflows) > 0 && toolingTools != nil {
		for _, id := range updatedWorkflows {
			if _, inTooling := toolingTools[id]; inTooling {
				output.Warn(stderr, fmt.Sprintf("  ⚠ workflow %s was updated (already in tooling.json — run: swytchcode add %s to refresh)", id, id))
			} else {
				fmt.Fprintf(stdout, "  ✓ workflow %s updated\n", id)
			}
		}
	}

	// Check for stale methods by comparing stored method_hash against current wrekenfile
	checkStaleMethods(projectRoot, projectName, toolingTools, stdout, stderr)

	return nil
}

// checkStaleMethods re-hashes each method's wrekenfile entry against the stored method_hash.
// If the hash differs, it warns the developer to re-add the tool.
// tools is the already-parsed tooling.json "tools" map; nil or empty means nothing to check.
func checkStaleMethods(projectRoot, projectName string, tools map[string]interface{}, stdout, stderr io.Writer) {
	if tools == nil {
		return
	}

	staleCount := 0

	for canonicalID, raw := range tools {
		entry, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if entry["type"] != "method" {
			continue
		}
		storedHash, _ := entry["method_hash"].(string)
		if storedHash == "" {
			continue // no hash recorded; skip (added before this feature)
		}
		integration, _ := entry["integration"].(string)
		if integration == "" {
			continue
		}

		// Parse "project.library@version"
		atIdx := strings.LastIndex(integration, "@")
		if atIdx < 0 {
			continue
		}
		projLib := integration[:atIdx]
		version := integration[atIdx+1:]
		parts := strings.SplitN(projLib, ".", 2)
		if len(parts) != 2 {
			continue
		}
		proj, lib := parts[0], parts[1]

		// Only check tools belonging to the currently synced project
		if proj != projectName {
			continue
		}

		wrekenPath := filepath.Join(util.IntegrationVersionDir(projectRoot, proj, lib, version), constants.WrekenfileYAMLFile)
		methodEntry, err := findMethodInWrekenfile(wrekenPath, canonicalID)
		if err != nil {
			continue
		}

		currentHash := computeMethodHash(methodEntry)
		if currentHash != "" && currentHash != storedHash {
			output.Warn(stderr, fmt.Sprintf("  ⚠ method %s has changed — run: swytchcode add %s to refresh tooling.json", canonicalID, canonicalID))
			staleCount++
		}
	}

	if staleCount == 0 && tools != nil {
		fmt.Fprintf(stdout, "  ✓ All method definitions up to date\n")
	}
}

// workflowStepsChanged reports whether two step lists differ by canonical_id order or count.
func workflowStepsChanged(local, remote []registry.WorkflowStep) bool {
	if len(local) != len(remote) {
		return true
	}
	for i := range local {
		if local[i].CanonicalID != remote[i].CanonicalID {
			return true
		}
	}
	return false
}
