// root.go defines the root cobra command and wires all subcommands (init, get, exec, etc.).
package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "swytchcode",
	Short: "Swytchcode execution kernel",
	Long:  "Swytchcode is the single execution authority for tools. Editors, agents, and languages are guests.",
}

// Execute is the main entrypoint invoked by cmd/swytchcode/main.go.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Root-level errors are considered invalid invocation.
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(bootstrapCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(mcpCmd)
}

