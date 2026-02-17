// init.go implements swytchcode init: creates .swytchcode/, tooling.json, and editor-specific config (one-time project setup).
package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/commands"
	"gitlab.com/swytchcode/shell/internal/util"
)

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
			_, editorChoice = util.SelectWithRetry("Which editor do you use?", []string{"cursor", "claude", "none"})
		}

		modeChoice := strings.ToLower(initMode)
		if interactive && modeChoice == "" {
			// Interactive mode: prompt for mode selection
			fmt.Println()
			_, modeChoice = util.SelectWithRetry("Which execution mode do you want to use?", []string{"production", "sandbox"})
		}

		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return fmt.Errorf("detect project root: %w", err)
		}

		// Use shared RunInit function (non-interactive mode)
		return commands.RunInit(projectRoot, editorChoice, modeChoice, os.Stdout)
	},
}

func init() {
	initCmd.Flags().StringVar(&initEditor, "editor", "", "cursor | claude | none")
	initCmd.Flags().StringVar(&initMode, "mode", "", "production | sandbox")
	initCmd.Flags().BoolVar(&initNonInteractive, "non-interactive", false, "disable prompts; suitable for CI")
}

