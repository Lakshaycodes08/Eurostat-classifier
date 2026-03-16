// discover.go implements RunDiscover: semantic capability search via POST /discover.
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"gitlab.com/swytchcode/cli/internal/output"
	"gitlab.com/swytchcode/cli/internal/registry"
)

// RunDiscover searches for API capabilities matching intent.
// projectName is optional (empty = cross-project search).
// topK is the max number of results to return.
func RunDiscover(ctx context.Context, intent, projectName string, topK int, jsonOutput bool, stdout, stderr io.Writer) error {
	regClient := registry.NewClient(registry.DefaultConfig())
	defer regClient.CloseIdleConnections()

	result, err := regClient.DiscoverCapabilities(ctx, intent, projectName, topK)
	if err != nil {
		output.Error(stderr, fmt.Sprintf("discovery failed: %v", err))
		output.Hint(stderr, "ensure the API is reachable and try again")
		return err
	}

	if jsonOutput {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	if len(result.Capabilities) == 0 {
		fmt.Fprintf(stdout, "No capabilities found matching %q\n", intent)
		output.Hint(stderr, "try a broader query, or run 'swytchcode get <project>' to add an integration")
		return nil
	}

	fmt.Fprintf(stdout, "Capabilities matching %q:\n\n", intent)
	for i, cap := range result.Capabilities {
		label := cap.CanonicalID
		if label == "" {
			label = fmt.Sprintf("%s.%s", cap.Library, cap.Type)
		}
		summary := strings.TrimSpace(cap.Summary)
		if len(summary) > 60 {
			summary = summary[:57] + "..."
		}
		fmt.Fprintf(stdout, "  %d. %-40s %s  [%s]\n", i+1, label, summary, cap.Type)
		fmt.Fprintf(stdout, "     exec: swytchcode exec %s\n\n", label)
	}

	if result.RecommendedWorkflow != nil {
		wf := result.RecommendedWorkflow
		label := wf.CanonicalID
		if label == "" {
			label = wf.Library
		}
		summary := strings.TrimSpace(wf.Summary)
		if len(summary) > 60 {
			summary = summary[:57] + "..."
		}
		fmt.Fprintf(stdout, "Recommended workflow:\n")
		fmt.Fprintf(stdout, "  %-40s %s\n", label, summary)
		fmt.Fprintf(stdout, "  exec: swytchcode exec %s\n", label)
		fmt.Fprintf(stdout, "  plan: swytchcode plan %s\n", label)
	}

	return nil
}
