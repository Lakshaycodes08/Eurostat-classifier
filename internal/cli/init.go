// init.go implements swytchcode init: creates .swytchcode/, tooling.json, and editor-specific config (one-time project setup).
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/editors"
	"gitlab.com/swytchcode/shell/internal/util"
)

// defaultRegistryURL returns the registry base URL from env or default (used when writing tooling.json).
func defaultRegistryURL() string {
	if u := os.Getenv("SWYTCHCODE_REGISTRY_URL"); u != "" {
		return u
	}
	return "https://localhost"
}

var (
	initEditor         string
	initMode           string
	initNonInteractive bool
)

// initCmd implements `swytchcode init`.
//
// Interaction rules:
//   - When running on a TTY and --non-interactive is NOT set, init is
//     allowed to prompt for the editor.
//   - In non-interactive mode (no TTY or --non-interactive), --editor
//     is REQUIRED; otherwise the command fails with exit code 1.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Swytchcode in this project",
	RunE: func(cmd *cobra.Command, args []string) error {
		isTTY := util.IsInteractive()
		interactive := isTTY && !initNonInteractive

		if !interactive {
			if initEditor == "" {
				return errors.New("init requires --editor when running non-interactively")
			}
			if initMode == "" {
				return errors.New("init requires --mode when running non-interactively")
			}
		}

		editorChoice := initEditor
		if interactive && editorChoice == "" {
			// Interactive mode: prompt for editor selection
			fmt.Println()
			_, editorChoice = util.SelectWithRetry("Which editor do you use?", []string{"cursor", "vscode", "claude", "none"})
		}

		modeChoice := strings.ToLower(initMode)
		if interactive && modeChoice == "" {
			// Interactive mode: prompt for mode selection
			fmt.Println()
			_, modeChoice = util.SelectWithRetry("Which execution mode do you want to use?", []string{"production", "sandbox"})
		}
		if modeChoice != "production" && modeChoice != "sandbox" {
			return fmt.Errorf("invalid mode %q (expected production or sandbox)", initMode)
		}

		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return fmt.Errorf("detect project root: %w", err)
		}

		swytchDir := filepath.Join(projectRoot, ".swytchcode")
		if err := util.EnsureDir(swytchDir, 0o755); err != nil {
			return fmt.Errorf("create .swytchcode directory: %w", err)
		}
		if err := util.EnsureDir(filepath.Join(swytchDir, "wrekenfiles"), 0o755); err != nil {
			return fmt.Errorf("create wrekenfiles directory: %w", err)
		}
		if err := util.EnsureDir(filepath.Join(swytchDir, "proposals"), 0o755); err != nil {
			return fmt.Errorf("create proposals directory: %w", err)
		}

		// Create or update tooling.json with mode
		toolingPath := filepath.Join(swytchDir, "tooling.json")
		var tooling map[string]interface{}
		
		if data, err := os.ReadFile(toolingPath); err == nil {
			// Load existing tooling.json
			if err := json.Unmarshal(data, &tooling); err != nil {
				// If invalid, start fresh
				tooling = make(map[string]interface{})
			}
		} else {
			// Create new tooling.json
			tooling = make(map[string]interface{})
		}

		// Ensure tools and integrations maps exist (integrations = pinned versions for determinism)
		if _, ok := tooling["tools"]; !ok {
			tooling["tools"] = make(map[string]interface{})
		}
		if _, ok := tooling["integrations"]; !ok {
			tooling["integrations"] = make(map[string]interface{})
		}

		// Set mode, version, and registry URL so they are visible and project-specific.
		// version and registry_url are owned by the kernel: only set when absent (never overwrite on re-init).
		tooling["mode"] = modeChoice
		if _, hasVersion := tooling["version"]; !hasVersion {
			tooling["version"] = "1.0"
		}
		if _, hasRegistryURL := tooling["registry_url"]; !hasRegistryURL {
			tooling["registry_url"] = defaultRegistryURL()
		}

		// Write tooling.json
		data, err := json.MarshalIndent(tooling, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal tooling.json: %w", err)
		}
		if err := os.WriteFile(toolingPath, data, 0o644); err != nil {
			return fmt.Errorf("write tooling.json: %w", err)
		}

		switch editorChoice {
		case "cursor":
			if err := editors.WriteCursorRules(projectRoot); err != nil {
				return fmt.Errorf("write Cursor rules: %w", err)
			}
		case "vscode":
			if err := editors.WriteVSCodeConfig(projectRoot); err != nil {
				return fmt.Errorf("write VS Code config: %w", err)
			}
		case "claude":
			if err := editors.WriteClaudeConfig(projectRoot); err != nil {
				return fmt.Errorf("write Claude config: %w", err)
			}
		case "none", "":
			// No editor configuration.
		default:
			return fmt.Errorf("unknown editor %q (expected cursor|vscode|claude|none)", editorChoice)
		}

		// For humans on a TTY, a short confirmation is acceptable here.
		if isTTY {
			fmt.Println("Swytchcode initialized for project at", projectRoot)
		}

		return nil
	},
}

func init() {
	initCmd.Flags().StringVar(&initEditor, "editor", "", "cursor | vscode | claude | none")
	initCmd.Flags().StringVar(&initMode, "mode", "", "production | sandbox")
	initCmd.Flags().BoolVar(&initNonInteractive, "non-interactive", false, "disable prompts; suitable for CI")
}

