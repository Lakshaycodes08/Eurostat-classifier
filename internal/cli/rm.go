// rm.go implements swytchcode rm: removes an integration’s Wrekenfile and optional proposals from the project.
package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/util"
)

var (
	rmAutoYes        bool
	rmNonInteractive bool
)

// rmCmd implements `swytchcode rm <library>`, which deletes a local
// Wrekenfile spec for a given library from .swytchcode/wrekenfiles/.
//
// Interaction rules:
//   - On a TTY without --non-interactive, rm MAY prompt for confirmation
//     before deleting. Until prompts are implemented, we require --yes
//     to proceed with deletion.
//   - In non-interactive mode (--non-interactive or no TTY), rm must not
//     prompt and requires --yes for destructive changes.
var rmCmd = &cobra.Command{
	Use:   "rm <library>",
	Short: "Remove a Wrekenfile spec for a library",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("library name required")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		library := args[0]
		interactive := util.IsInteractive() && !rmNonInteractive

		if !rmAutoYes {
			// Until explicit prompts are implemented, require --yes for safety.
			return errors.New("destructive delete requires --yes (interactive confirmation not implemented yet)")
		}

		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return fmt.Errorf("detect project root: %w", err)
		}

		wrekenPath := filepath.Join(projectRoot, ".swytchcode", "wrekenfiles", library+".yaml")
		if _, err := os.Stat(wrekenPath); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("no Wrekenfile spec found for library %q", library)
		} else if err != nil {
			return fmt.Errorf("stat Wrekenfile: %w", err)
		}

		if err := os.Remove(wrekenPath); err != nil {
			return fmt.Errorf("remove Wrekenfile: %w", err)
		}

		// Remove any proposals for this library so they don't reference a removed integration.
		if n, errProposals := removeProposalsForLibrary(projectRoot, library); errProposals != nil {
			return fmt.Errorf("removed Wrekenfile; removed %d proposal(s) but then failed: %w", n, errProposals)
		} else if n > 0 && interactive {
			fmt.Printf("Removed %d proposal(s) for %s\n", n, library)
		}

		if interactive {
			fmt.Printf("Removed Wrekenfile spec for %s\n", library)
		}

		return nil
	},
}

func init() {
	rmCmd.Flags().BoolVar(&rmAutoYes, "yes", false, "auto-confirm deletion of the Wrekenfile spec")
	rmCmd.Flags().BoolVar(&rmNonInteractive, "non-interactive", false, "disable prompts; suitable for CI")
}

