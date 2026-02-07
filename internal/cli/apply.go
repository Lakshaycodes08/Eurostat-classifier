// apply.go implements swytchcode apply: merges a validated proposal into tooling.json and archives it.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/util"
)

var applyCmd = &cobra.Command{
	Use:   "apply <proposal_file>",
	Short: "Apply a proposal to tooling.json",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("proposal file path required")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		proposalPath := args[0]

		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return fmt.Errorf("detect project root: %w", err)
		}
		if !filepath.IsAbs(proposalPath) {
			proposalPath = filepath.Join(projectRoot, proposalPath)
		}

		// Read proposal file
		proposalData, err := os.ReadFile(proposalPath)
		if err != nil {
			return fmt.Errorf("read proposal file: %w", err)
		}

		var proposal map[string]interface{}
		if err := json.Unmarshal(proposalData, &proposal); err != nil {
			return fmt.Errorf("parse proposal file: %w", err)
		}

		// Reuse same validation as swytchcode validate; fail if invalid
		if errs := validateProposalContent(proposal, projectRoot); len(errs) > 0 {
			return fmt.Errorf("invalid proposal: %s", errs[0])
		}

		toolingFragment, _ := proposal["tooling_fragment"].(map[string]interface{})
		tools, _ := toolingFragment["tools"].(map[string]interface{})

		toolingPath := filepath.Join(projectRoot, ".swytchcode", "tooling.json")

		// Load existing tooling.json
		var tooling map[string]interface{}
		if data, err := os.ReadFile(toolingPath); err == nil {
			if err := json.Unmarshal(data, &tooling); err != nil {
				return fmt.Errorf("parse tooling.json: %w", err)
			}
		} else {
			return fmt.Errorf("tooling.json not found; run 'swytchcode init' first")
		}

		// Ensure tools and integrations maps exist
		if _, ok := tooling["tools"]; !ok {
			tooling["tools"] = make(map[string]interface{})
		}
		if _, ok := tooling["integrations"]; !ok {
			tooling["integrations"] = make(map[string]interface{})
		}

		// Merge proposal integrations (with explicit versions) into tooling.json
		if proposalIntegrations, ok := proposal["integrations"].(map[string]interface{}); ok {
			existingIntegrations := tooling["integrations"].(map[string]interface{})
			for name, ent := range proposalIntegrations {
				existingIntegrations[name] = ent
			}
		}

		existingTools, ok := tooling["tools"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid tooling.json: tools must be an object")
		}

		// Merge proposal tools into tooling.json
		for toolID, toolDef := range tools {
			existingTools[toolID] = toolDef
		}

		// Write updated tooling.json
		data, err := json.MarshalIndent(tooling, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal tooling.json: %w", err)
		}

		if err := os.WriteFile(toolingPath, data, 0o644); err != nil {
			return fmt.Errorf("write tooling.json: %w", err)
		}

		// Archive or delete proposal file
		proposalsDir := filepath.Join(projectRoot, ".swytchcode", "proposals")
		archivedDir := filepath.Join(proposalsDir, "applied")
		if err := util.EnsureDir(archivedDir, 0o755); err == nil {
			proposalName := filepath.Base(proposalPath)
			archivedPath := filepath.Join(archivedDir, proposalName)
			os.Rename(proposalPath, archivedPath)
		} else {
			// If we can't archive, just delete
			os.Remove(proposalPath)
		}

		if util.IsInteractive() {
			fmt.Printf("Applied proposal to tooling.json\n")
		}

		return nil
	},
}
