// doctor.go implements `swytchcode doctor`.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/swytchcode-cli/internal/commands"
	"gitlab.com/swytchcode/swytchcode-cli/internal/util"
)

var doctorJSON bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run local diagnostics for the Swytchcode project",
	Long: `Checks tooling.json, installed integration bundles, manifest.json, execution base URLs,
and auth-related environment. Exits 1 if any check reports an error (useful in CI).

Warnings (e.g. no auth for offline exec) do not fail the command.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot, err := util.ProjectRoot()
		if err != nil {
			if doctorJSON {
				_ = util.WriteJSON(os.Stdout, map[string]interface{}{
					"ok":     false,
					"checks": []map[string]string{{"id": "project", "status": "error", "message": err.Error()}},
				})
			} else {
				fmt.Fprintln(os.Stderr, err.Error())
			}
			os.Exit(1)
		}

		rep := commands.RunDoctor(projectRoot)
		if doctorJSON {
			if err := util.WriteJSON(os.Stdout, rep); err != nil {
				return err
			}
			if !rep.OK {
				os.Exit(1)
			}
			return nil
		}
		fmt.Print(commands.FormatDoctorHuman(rep))
		if !rep.OK {
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "print report as JSON")
}
