package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/util"
)

var (
	upgradeAutoYes        bool
	upgradeNonInteractive bool
)

// upgradeCmd implements `swytchcode upgrade <library>`, which refreshes
// an existing Wrekenfile spec from the registry.
//
// Semantics:
//   - Logically equivalent to "get, but require that the spec already
//     exists and treat the operation as an explicit upgrade".
//
// Interaction rules:
//   - On a TTY without --non-interactive, upgrade MAY prompt for
//     confirmation before overwriting. Until prompts are implemented,
//     we require --yes to proceed.
//   - In non-interactive mode (--non-interactive or no TTY), upgrade
//     must not prompt and requires --yes to overwrite.
var upgradeCmd = &cobra.Command{
	Use:   "upgrade <library>",
	Short: "Upgrade a Wrekenfile spec for a library",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("library name required")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		library := args[0]
		_ = util.IsInteractive() // reserved for future prompt behavior

		if !upgradeAutoYes {
			// Safety: until explicit prompts are implemented, require --yes.
			return errors.New("upgrade that overwrites specs requires --yes (interactive confirmation not implemented yet)")
		}

		// TODO:
		// 1. Verify that .swytchcode/wrekenfiles/<library>.json already exists.
		// 2. Resolve library -> registry endpoint.
		// 3. Fetch latest Wrekenfile JSON/YAML.
		// 4. Validate via internal/wreken.
		// 5. Overwrite existing spec atomically.

		fmt.Printf("Wrekenfile spec for %s would be upgraded from the registry (implementation pending)\n", library)
		return nil
	},
}

func init() {
	upgradeCmd.Flags().BoolVar(&upgradeAutoYes, "yes", false, "auto-confirm overwrite during upgrade")
	upgradeCmd.Flags().BoolVar(&upgradeNonInteractive, "non-interactive", false, "disable prompts; suitable for CI")
}

