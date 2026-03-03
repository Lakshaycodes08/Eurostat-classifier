// login.go implements the `swytchcode login` command: device-flow OAuth.
package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/commands"
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
		apiURL := os.Getenv("SWYTCHCODE_API_URL")
		if apiURL == "" {
			apiURL = "http://localhost:80"
		}

		cfg := commands.LoginConfig{APIURL: apiURL}
		if loginOpen {
			cfg.OnURL = openBrowser
		}
		return commands.RunLogin(cfg, os.Stdout)
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
		fmt.Fprintf(os.Stderr, "could not open browser: %v\n", err)
	}
}

func init() {
	loginCmd.Flags().BoolVar(&loginOpen, "open", false, "Open the login URL in your browser automatically")
}
