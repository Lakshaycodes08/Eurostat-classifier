// whoami.go implements `swytchcode whoami`: prints the current login state.
package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/cli/internal/auth"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the currently logged-in user",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := auth.Load()
		if err != nil {
			fmt.Fprintln(os.Stdout, "Not logged in.")
			return nil
		}
		if s.IsExpired() {
			fmt.Fprintln(os.Stdout, "Not logged in. (session expired — run `swytchcode login`)")
			return nil
		}

		remaining := time.Until(time.Unix(s.ExpiresAt, 0))
		minutes := int(remaining.Minutes())

		fmt.Fprintf(os.Stdout, "email:         %s\n", s.Email)
		fmt.Fprintf(os.Stdout, "customer_uuid: %s\n", s.CustomerUUID)
		fmt.Fprintf(os.Stdout, "session:       valid (expires in %d minutes)\n", minutes)
		return nil
	},
}
