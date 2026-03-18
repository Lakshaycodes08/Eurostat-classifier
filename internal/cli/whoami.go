// whoami.go implements `swytchcode whoami`: prints the current authentication state.
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
	Short: "Show the current authentication state",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Priority 1: service token via env var
		if t := os.Getenv("SWYTCHCODE_TOKEN"); t != "" {
			fmt.Fprintf(os.Stdout, "auth:    service token (SWYTCHCODE_TOKEN)\n")
			fmt.Fprintf(os.Stdout, "token:   %s\n", maskToken(t))
			return nil
		}

		// Priority 2: user session from swytchcode login
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
		fmt.Fprintf(os.Stdout, "auth:          user session\n")
		fmt.Fprintf(os.Stdout, "email:         %s\n", s.Email)
		fmt.Fprintf(os.Stdout, "customer_uuid: %s\n", s.CustomerUUID)
		fmt.Fprintf(os.Stdout, "session:       valid (expires in %d minutes)\n", minutes)
		return nil
	},
}

// maskToken shows the first 4 and last 4 characters with **** in between.
// Short tokens (≤8 chars) are fully masked.
func maskToken(t string) string {
	if len(t) <= 8 {
		return "****"
	}
	return t[:4] + "****" + t[len(t)-4:]
}
