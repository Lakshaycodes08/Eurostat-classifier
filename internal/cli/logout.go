// logout.go implements `swytchcode logout`: deletes the local session file.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/auth"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and remove the local session",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if a session exists first so we can give a useful message.
		if _, err := auth.Load(); err != nil {
			fmt.Fprintln(os.Stdout, "Not logged in.")
			return nil
		}
		if err := auth.Delete(); err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, "Logged out.")
		return nil
	},
}
