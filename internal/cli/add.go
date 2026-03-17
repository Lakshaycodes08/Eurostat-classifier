// add.go implements the add command (adds tools to tooling.json by canonical_id).
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
	"gitlab.com/swytchcode/cli/internal/commands"
	"gitlab.com/swytchcode/cli/internal/util"
)

var noAutoInstall bool
var addAll bool

// addCmd is the main add command that accepts canonical_id directly.
// Usage:
//   - swytchcode add <canonical_id> (searches all integrations)
//   - swytchcode add <integration_spec> <canonical_id> (explicit scoped)
//   - swytchcode add --all <project> (add all methods/workflows for a project)
var addCmd = &cobra.Command{
	Use:   "add [integration_spec] <canonical_id>",
	Short: "Add a tool (method or workflow) to tooling.json by canonical_id",
	Args: func(cmd *cobra.Command, args []string) error {
		all, _ := cmd.Flags().GetBool("all")
		if all {
			if len(args) != 1 {
				return errors.New("usage: swytchcode add --all <project>")
			}
			return nil
		}
		if len(args) < 1 || len(args) > 2 {
			return errors.New("usage: swytchcode add [integration_spec] <canonical_id>")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if addAll {
			return commands.RunAddAll(context.Background(), args[0], os.Stdout, os.Stderr)
		}
		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return fmt.Errorf("detect project root: %w", err)
		}

		var canonicalID, integrationSpec string
		if len(args) == 1 {
			canonicalID = args[0]
			integrationSpec = ""
		} else {
			integrationSpec = args[0]
			canonicalID = args[1]
		}

		// If no explicit spec, resolve via search; allow interactive disambiguation
		if integrationSpec == "" {
			matches, err := commands.FindToolInAllIntegrations(projectRoot, canonicalID)
			if err != nil {
				return err
			}
			if len(matches) == 0 {
				return fmt.Errorf("canonical ID %q not found in any fetched integrations.\nRun: swytchcode get <project>", canonicalID)
			}
			if len(matches) == 1 {
				// Single match: RunAdd with empty spec will use it
				return commands.RunAdd(context.Background(), canonicalID, "", noAutoInstall, os.Stdout, os.Stderr)
			}
			// Multiple matches
			interactive := util.IsInteractive()
			if !interactive {
				var options []string
				for _, m := range matches {
					options = append(options, fmt.Sprintf("%s@%s.%s (%s)", m.Project, m.Library, m.Version, m.ToolType))
				}
				return fmt.Errorf("ambiguous canonical ID. Found in %d integrations:\n  %s\nUse: swytchcode add <integration@version> %s", len(matches), strings.Join(options, "\n  "), canonicalID)
			}
			options := make([]string, len(matches))
			for i, m := range matches {
				options[i] = fmt.Sprintf("%s@%s.%s (%s)", m.Project, m.Library, m.Version, m.ToolType)
			}
			fmt.Printf("\nCanonical ID %q found in multiple integrations:\n\n", canonicalID)
			for i, opt := range options {
				fmt.Printf("  %d) %s\n", i+1, opt)
			}
			fmt.Println()
			_, selected := util.SelectWithRetry("Select integration:", options)
			if selected == "" {
				return errors.New("no integration selected")
			}
			// Strip optional " (method)" suffix
			if idx := strings.Index(selected, " ("); idx > 0 {
				selected = selected[:idx]
			}
			integrationSpec = selected
		}

		return commands.RunAdd(context.Background(), canonicalID, integrationSpec, noAutoInstall, os.Stdout, os.Stderr)
	},
}

// addIntegrationCmd adds an integration version to tooling.json (does not fetch).
// Usage: swytchcode add integration weaviate@lyrid.v1
var addIntegrationCmd = &cobra.Command{
	Use:   "integration <project@library.version>",
	Short: "Add an integration version to tooling.json (explicit, reviewable; does not fetch)",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("integration spec required (e.g. weaviate@lyrid.v1)")
		}
		project, library, version := commands.ParseIntegrationSpec(args[0])
		if project == "" || library == "" || version == "" {
			return errors.New("integration must be <project@library.version> (e.g. weaviate@lyrid.v1)")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		project, library, version := commands.ParseIntegrationSpec(args[0])
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
		projectLibrary := fmt.Sprintf("%s.%s", project, library)
		integrations[projectLibrary] = map[string]interface{}{"version": version}
		data, err := json.MarshalIndent(tooling, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal tooling.json: %w", err)
		}
		if err := os.WriteFile(toolingPath, data, 0o644); err != nil {
			return fmt.Errorf("write tooling.json: %w", err)
		}
		if util.IsInteractive() {
			fmt.Printf("Added %s@%s.%s to tooling.json. Run 'swytchcode bootstrap' to install.\n", project, library, version)
		}
		return nil
	},
}

func init() {
	addCmd.AddCommand(addIntegrationCmd)
	addCmd.Flags().BoolVar(&noAutoInstall, "no-auto-install", false, "Do not auto-download missing library dependencies for multi-library workflows")
	addCmd.Flags().BoolVar(&addAll, "all", false, "Add all methods and workflows for a project (usage: swytchcode add --all <project>)")
}
