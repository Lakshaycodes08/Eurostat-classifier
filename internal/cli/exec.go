package cli

import (
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/kernel"
)

var execAllowRaw bool

// execCmd implements `swytchcode exec`.
//
// It must:
//   - Never prompt.
//   - Never branch on TTY presence.
//   - Only accept JSON on stdin and emit JSON on stdout/stderr.
//   - Require --allow-raw for raw method execution.
var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "Execute a tool via the Swytchcode kernel",
	Run: func(cmd *cobra.Command, args []string) {
		exitCode := kernel.Execute(os.Stdin, os.Stdout, os.Stderr, execAllowRaw)
		os.Exit(exitCode)
	},
}

func init() {
	execCmd.Flags().BoolVar(&execAllowRaw, "allow-raw", false, "allow execution of raw methods (required for tools starting with 'raw.')")
}

