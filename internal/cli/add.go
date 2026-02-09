// add.go implements the add workflow and add integration subcommands (verified workflows and pinned integrations).
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/registry"
	"gitlab.com/swytchcode/shell/internal/util"
)

// addCmd is the parent command for add subcommands.
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add verified workflows, integrations, or tools",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// addIntegrationCmd adds an integration version pin to tooling.json (does not fetch).
// Usage: swytchcode add integration stripe@2025-01-10
var addIntegrationCmd = &cobra.Command{
	Use:   "integration <name>@<version>",
	Short: "Pin an integration version in tooling.json (explicit, reviewable; does not fetch)",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("integration spec required (e.g. stripe@2025-01-10)")
		}
		name, version := parseIntegrationSpec(args[0])
		if name == "" || version == "" {
			return errors.New("integration must be <name>@<version> (exact version, no 'latest')")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		name, version := parseIntegrationSpec(args[0])
		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return fmt.Errorf("detect project root: %w", err)
		}
		toolingPath := filepath.Join(projectRoot, ".swytchcode", "tooling.json")
		var tooling map[string]interface{}
		if data, err := os.ReadFile(toolingPath); err == nil {
			if err := json.Unmarshal(data, &tooling); err != nil {
				return fmt.Errorf("parse tooling.json: %w", err)
			}
		} else {
			return fmt.Errorf("tooling.json not found; run 'swytchcode init' first: %w", err)
		}
		integrations, _ := tooling["integrations"].(map[string]interface{})
		if integrations == nil {
			integrations = make(map[string]interface{})
			tooling["integrations"] = integrations
		}
		integrations[name] = map[string]interface{}{"version": version}
		data, err := json.MarshalIndent(tooling, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal tooling.json: %w", err)
		}
		if err := os.WriteFile(toolingPath, data, 0o644); err != nil {
			return fmt.Errorf("write tooling.json: %w", err)
		}
		if util.IsInteractive() {
			fmt.Printf("Pinned %s@%s in tooling.json. Run 'swytchcode bootstrap' to install.\n", name, version)
		}
		return nil
	},
}

// parseIntegrationSpec parses "name@version" and returns (name, version). No "latest".
func parseIntegrationSpec(spec string) (name, version string) {
	i := strings.LastIndex(spec, "@")
	if i < 0 {
		return spec, ""
	}
	return strings.TrimSpace(spec[:i]), strings.TrimSpace(spec[i+1:])
}

var addWorkflowCmd = &cobra.Command{
	Use:   "workflow <workflow_id>",
	Short: "Add a verified workflow to tooling.json",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("workflow ID required")
		}
		return nil
	},
		RunE: func(cmd *cobra.Command, args []string) error {
		workflowID := args[0]

		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return fmt.Errorf("detect project root: %w", err)
		}
		regClient := registry.NewClient(registry.ConfigFromProjectRoot(projectRoot))
		ctx := context.Background()

		// Load project_uuid from tooling.json
		toolingPath := filepath.Join(projectRoot, ".swytchcode", "tooling.json")
		var tooling map[string]interface{}
		if data, err := os.ReadFile(toolingPath); err == nil {
			if err := json.Unmarshal(data, &tooling); err != nil {
				return fmt.Errorf("parse tooling.json: %w", err)
			}
		} else {
			// Create new if doesn't exist
			tooling = make(map[string]interface{})
		}

		var projectUUID string
		if uuid, ok := tooling["project_uuid"].(string); ok && uuid != "" {
			projectUUID = uuid
		}
		if projectUUID == "" {
			return fmt.Errorf("project_uuid not found in tooling.json; run 'swytchcode init' first")
		}

		workflow, err := regClient.GetWorkflow(ctx, workflowID)
		if err != nil {
			return fmt.Errorf("fetch workflow: %w", err)
		}

		// Ensure tools map exists
		if _, ok := tooling["tools"]; !ok {
			tooling["tools"] = make(map[string]interface{})
		}

		tools, ok := tooling["tools"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid tooling.json: tools must be an object")
		}

		// Note: The new API returns workflow.steps instead of tooling_fragment.tools
		// For now, we'll need to convert steps to tools format or handle it differently
		// This is a breaking change - workflows now return steps array, not tooling_fragment
		// TODO: Update this to handle the new workflow.steps structure
		// For now, we'll create a placeholder tool from the workflow
		if len(workflow.Steps) > 0 {
			// Create a tool ID from workflow_id
			toolID := workflow.WorkflowID
			tools[toolID] = map[string]interface{}{
				"title": workflow.Title,
				"version": workflow.Version,
				"steps": workflow.Steps,
			}
		}

		// Write updated tooling.json
		data, err := json.MarshalIndent(tooling, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal tooling.json: %w", err)
		}

		if err := os.WriteFile(toolingPath, data, 0o644); err != nil {
			return fmt.Errorf("write tooling.json: %w", err)
		}

		if util.IsInteractive() {
			fmt.Printf("Added verified workflow %s (version %s) to tooling.json\n", workflowID, workflow.Version)
		}

		return nil
	},
}

func init() {
	addCmd.AddCommand(addWorkflowCmd)
	addCmd.AddCommand(addIntegrationCmd)
}
