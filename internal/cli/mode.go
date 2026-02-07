// mode.go implements swytchcode mode: reads or sets execution mode (production | sandbox) in tooling.json.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/util"
)

// modeCmd implements `swytchcode mode [production|sandbox]`.
//
// Purpose: Set or display the execution mode for the project.
// Modes:
//   - production: Use production credentials and enforce strict policies
//   - sandbox: Use sandbox/test credentials and allow experimental features
var modeCmd = &cobra.Command{
	Use:   "mode [production|sandbox]",
	Short: "Set or display the execution mode",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errors.New("too many arguments")
		}
		if len(args) == 1 {
			mode := strings.ToLower(args[0])
			if mode != "production" && mode != "sandbox" {
				return fmt.Errorf("invalid mode %q (expected production or sandbox)", args[0])
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return fmt.Errorf("detect project root: %w", err)
		}

		swytchDir := filepath.Join(projectRoot, ".swytchcode")
		toolingPath := filepath.Join(swytchDir, "tooling.json")

		// If no argument provided, display current mode
		if len(args) == 0 {
			var tooling map[string]interface{}
			if data, err := os.ReadFile(toolingPath); err == nil {
				if err := json.Unmarshal(data, &tooling); err == nil {
					if mode, ok := tooling["mode"].(string); ok {
						fmt.Println(mode)
						return nil
					}
				}
			}
			// Default mode if not set
			fmt.Println("production")
			return nil
		}

		// Set mode
		mode := strings.ToLower(args[0])

		// Ensure .swytchcode directory exists
		if err := util.EnsureDir(swytchDir, 0o755); err != nil {
			return fmt.Errorf("create .swytchcode directory: %w", err)
		}

		// Load existing tooling.json or create new
		tooling := make(map[string]interface{})
		if data, err := os.ReadFile(toolingPath); err == nil {
			if err := json.Unmarshal(data, &tooling); err != nil {
				// If tooling.json is invalid, start fresh
				tooling = make(map[string]interface{})
			}
		}

		// Ensure tools map exists
		if _, ok := tooling["tools"]; !ok {
			tooling["tools"] = make(map[string]interface{})
		}

		// Update mode
		tooling["mode"] = mode

		// Write tooling.json
		data, err := json.MarshalIndent(tooling, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal tooling.json: %w", err)
		}

		if err := os.WriteFile(toolingPath, data, 0o644); err != nil {
			return fmt.Errorf("write tooling.json: %w", err)
		}

		if util.IsInteractive() {
			fmt.Printf("Mode set to %s\n", mode)
		}

		return nil
	},
}
