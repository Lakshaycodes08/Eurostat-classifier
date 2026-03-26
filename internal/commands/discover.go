// discover.go implements RunDiscover: semantic capability search via POST /discover.
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

// RunDiscover searches for API capabilities matching intent.
// projectName and libraryName are optional (empty = cross-project/library search).
// topK is the max number of results to return.
func RunDiscover(ctx context.Context, intent, projectName, libraryName string, topK int, jsonOutput bool, stdout, stderr io.Writer) error {
	regClient := registry.NewClient(registry.DefaultConfig())
	defer regClient.CloseIdleConnections()

	result, err := regClient.DiscoverCapabilities(ctx, intent, projectName, libraryName, topK)
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
		if len(summary) > 80 {
			summary = summary[:77] + "..."
		}
		fmt.Fprintf(stdout, "  %d. %s\n", i+1, label)
		fmt.Fprintf(stdout, "     %s\n", buildLibLabel(cap.Library, cap.Version, cap.LibraryUUIDs))
		fmt.Fprintf(stdout, "     [%s] %s\n\n", cap.Type, summary)
	}

	if result.RecommendedWorkflow != nil {
		wf := result.RecommendedWorkflow
		label := wf.CanonicalID
		if label == "" {
			label = wf.Library
		}
		summary := strings.TrimSpace(wf.Summary)
		if len(summary) > 80 {
			summary = summary[:77] + "..."
		}
		fmt.Fprintf(stdout, "Recommended workflow:\n")
		fmt.Fprintf(stdout, "  %s\n", label)
		fmt.Fprintf(stdout, "  %s\n", buildLibLabel(wf.Library, wf.Version, wf.LibraryUUIDs))
		fmt.Fprintf(stdout, "  [%s] %s\n", wf.Type, summary)
		fmt.Fprintf(stdout, "  plan: swytchcode plan %s\n", label)
	}

	return nil
}

// buildLibLabel returns the library display string for a capability result.
// Multi-library workflows show "primary + N more"; single-library shows "name@version".
func buildLibLabel(library, version string, libraryUUIDs []string) string {
	extra := len(libraryUUIDs) - 1
	if extra > 0 {
		if library == "" {
			return fmt.Sprintf("%d libraries", len(libraryUUIDs))
		}
		return fmt.Sprintf("%s + %d more", library, extra)
	}
	if version != "" {
		return library + "@" + version
	}
	return library
}
