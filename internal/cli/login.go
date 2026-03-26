// login.go implements the `swytchcode login` command: device-flow OAuth.
package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/swytchcode-cli/internal/auth"
	"gitlab.com/swytchcode/swytchcode-cli/internal/commands"
	"gitlab.com/swytchcode/swytchcode-cli/internal/output"
	"gitlab.com/swytchcode/swytchcode-cli/internal/telemetry"
)

var loginOpen bool

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to Swytchcode via browser",
	Long: `Starts the device-flow login. A URL is printed — open it in your browser
to authenticate with your Swytchcode account. The CLI polls until the browser
flow completes and saves your session to ~/.swytchcode/auth.json.

Use --open to have the CLI open the browser automatically.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiURL := auth.ResolveAPIURL()
		cfg := commands.LoginConfig{APIURL: apiURL}
		if loginOpen {
			cfg.OnURL = openBrowser
		}
		if err := commands.RunLogin(cfg, os.Stdout); err != nil {
			return err
		}
		// Telemetry only after successful login (we have a token now).
		if session, loadErr := auth.Load(); loadErr == nil {
			telemetry.SendEvent(apiURL, session.AccessToken, true, "login", "", nil, nil)
		}
		return nil
	},
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		output.Warn(os.Stderr, fmt.Sprintf("could not open browser: %v", err))
	}
}

func init() {
	loginCmd.Flags().BoolVar(&loginOpen, "open", false, "Open the login URL in your browser automatically")
}
