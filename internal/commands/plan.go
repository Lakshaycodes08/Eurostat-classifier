// plan.go implements RunPlan: fetch and display workflow steps.
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"gitlab.com/swytchcode/swytchcode-cli/internal/output"
	"gitlab.com/swytchcode/swytchcode-cli/internal/registry"
)

// RunPlan fetches and prints the steps of a workflow by canonical ID.
// If projectName is empty, it is derived from the first segment of canonicalID
// (e.g. "stripe" from "stripe.charge_customer").
func RunPlan(ctx context.Context, canonicalID, projectName string, jsonOutput bool, stdout, stderr io.Writer) error {
	if projectName == "" {
		if i := strings.Index(canonicalID, "."); i > 0 {
			projectName = canonicalID[:i]
		} else {
			output.Error(stderr, "could not determine project name from canonical ID")
			output.Hint(stderr, "pass --project <project> explicitly")
			return fmt.Errorf("project name required")
		}
	}

	regClient := registry.NewClient(registry.DefaultConfig())
	defer regClient.CloseIdleConnections()

	wf, err := regClient.GetWorkflow(ctx, projectName, canonicalID)
	if err != nil {
		output.Error(stderr, err.Error())
		output.Hint(stderr, fmt.Sprintf("run 'swytchcode discover <intent> --project %s' to find workflows", projectName))
		return err
	}

	if jsonOutput {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(wf)
	}

	title := wf.Title
	if title == "" {
		title = wf.CanonicalID
	}
	fmt.Fprintf(stdout, "Workflow: %s\n", title)
	if len(wf.Steps) == 0 {
		fmt.Fprintln(stdout, "  (no steps defined)")
	} else {
		fmt.Fprintln(stdout, "Steps:")
		for i, step := range wf.Steps {
			name := step.Name
			if name == "" {
				name = step.CanonicalID
			}
			fmt.Fprintf(stdout, "  %d. %s\n", i+1, name)
		}
	}
	fmt.Fprintf(stdout, "\nRun with: swytchcode exec %s\n", wf.CanonicalID)

	return nil
}
