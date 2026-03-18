// diff.go implements RunDiff: shows method-level signature changes between the installed
// version and the pending TinyFish proposal for a library.
package commands

import (
	"context"
	"fmt"
	"io"
	"strings"

	"gitlab.com/swytchcode/cli/internal/registry"
)

// RunDiff shows a human-readable diff of what would change if the pending proposal for
// library is applied. Requires a token (same requirements as swytchcode inspect).
func RunDiff(ctx context.Context, library, token string, stdout, stderr io.Writer) error {
	regClient := registry.NewClient(registry.DefaultConfigWithToken(token))
	defer regClient.CloseIdleConnections()

	diff, err := regClient.GetProposalDiff(ctx, library)
	if err != nil {
		return fmt.Errorf("fetch proposal diff: %w", err)
	}
	if diff == nil {
		fmt.Fprintf(stdout, "No pending proposal found for %q\n", library)
		return nil
	}

	fmt.Fprintf(stdout, "%s  %s → %s\n\n", diff.Library, diff.CurrentVersion, diff.ProposedVersion)

	if len(diff.Added) == 0 && len(diff.Removed) == 0 && len(diff.Changed) == 0 {
		fmt.Fprintf(stdout, "No method-level changes detected.\n")
		return nil
	}

	for _, id := range diff.Added {
		fmt.Fprintf(stdout, "ADDED    %s\n", id)
	}

	for _, id := range diff.Removed {
		fmt.Fprintf(stdout, "REMOVED  %s  [breaking]\n", id)
	}

	for _, change := range diff.Changed {
		fmt.Fprintf(stdout, "CHANGED  %s\n", change.CanonicalID)
		for _, f := range change.AddedInputs {
			req := ""
			if f.Required {
				req = " [breaking]"
			}
			fmt.Fprintf(stdout, "  + inputs.%-30s (%s, %s)%s\n", f.Name, f.Type, requiredLabel(f.Required), req)
		}
		for _, f := range change.RemovedInputs {
			fmt.Fprintf(stdout, "  - inputs.%-30s (%s) [breaking]\n", f.Name, f.Type)
		}
		for _, f := range change.ChangedInputs {
			fmt.Fprintf(stdout, "  ~ inputs.%-30s type: %s → %s\n", f.Name, f.OldType, f.NewType)
		}
		fmt.Fprintln(stdout)
	}

	// Summary
	var parts []string
	if n := len(diff.Added); n > 0 {
		parts = append(parts, fmt.Sprintf("%d added", n))
	}
	if n := len(diff.Removed); n > 0 {
		parts = append(parts, fmt.Sprintf("%d removed", n))
	}
	if n := len(diff.Changed); n > 0 {
		parts = append(parts, fmt.Sprintf("%d changed", n))
	}
	fmt.Fprintf(stdout, "Summary: %s\n", strings.Join(parts, ", "))
	fmt.Fprintf(stdout, "\nTo apply: swytchcode upgrade %s\n", library)
	return nil
}

func requiredLabel(required bool) string {
	if required {
		return "required"
	}
	return "optional"
}
