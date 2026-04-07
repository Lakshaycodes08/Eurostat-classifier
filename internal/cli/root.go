// root.go defines the root cobra command and wires all subcommands (init, get, exec, etc.).
package cli

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/swytchcode-cli/internal/constants"
)

var rootCmd = &cobra.Command{
	Use:          "swytchcode",
	Short:        "Swytchcode execution kernel",
	Long:         "Swytchcode is the single execution authority for tools. Editors, agents, and languages are guests.",
	Version:      constants.Version,
	SilenceUsage: true,
}

// Execute is the main entrypoint invoked by cmd/swytchcode/main.go.
func Execute() {
	// Shorthand: `swytchcode stripe.create_payment [flags]` → `swytchcode exec stripe.create_payment [flags]`
	// Canonical IDs always contain a dot; known subcommands never do.
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") && strings.Contains(os.Args[1], ".") {
		os.Args = append([]string{os.Args[0], "exec"}, os.Args[1:]...)
	}
	if err := rootCmd.Execute(); err != nil {
		// Root-level errors are considered invalid invocation.
		os.Exit(1)
	}
}

func init() {
	// Command structs and flags live in sibling files; subcommands are attached here.
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(demoCmd)
	rootCmd.AddCommand(bootstrapCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(whoamiCmd)
	rootCmd.AddCommand(inspectCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(discoverCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(syncCmd)
}

