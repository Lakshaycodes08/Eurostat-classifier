// validate_cmd.go implements swytchcode validate: checks a proposal for correctness without mutating tooling.json.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/util"
)

// validateCmd implements `swytchcode validate <proposal>`: full validation, no side effects.
// IDE-first flow: IDEs generate proposals; validate is how the kernel proves correctness.
var validateCmd = &cobra.Command{
	Use:   "validate <proposal_file>",
	Short: "Validate a proposal (structural correctness, invariants, compatibility)",
	Long:  "Fully validates proposal correctness. Ensures compatibility with installed integrations. Produces structured errors. No side effects.",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("proposal file path required")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		proposalPath := args[0]

		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return fmt.Errorf("detect project root: %w", err)
		}

		if !filepath.IsAbs(proposalPath) {
			proposalPath = filepath.Join(projectRoot, proposalPath)
		}

		data, err := os.ReadFile(proposalPath)
		if err != nil {
			return fmt.Errorf("read proposal file: %w", err)
		}

		var proposal map[string]interface{}
		if err := json.Unmarshal(data, &proposal); err != nil {
			return fmt.Errorf("parse proposal file: %w", err)
		}

		errs := validateProposalContent(proposal, projectRoot)
		if len(errs) > 0 {
			_ = util.WriteJSON(os.Stderr, map[string]interface{}{
				"valid":  false,
				"errors": errs,
			})
			return fmt.Errorf("validation failed: %d error(s)", len(errs))
		}

		_ = util.WriteJSON(os.Stdout, map[string]interface{}{
			"valid":   true,
			"message": "proposal is valid",
		})
		return nil
	},
}
