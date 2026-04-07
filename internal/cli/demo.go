// demo.go implements `swytchcode demo list` — shows all available demo tools.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/swytchcode-cli/internal/auth"
	"gitlab.com/swytchcode/swytchcode-cli/internal/constants"
)

var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Explore and run demo integrations",
}

var demoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available demo tools",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiURL := auth.ResolveAPIURL()

		client := constants.NewHTTPClient(constants.HTTPClientTimeout)
		resp, err := client.Get(apiURL + constants.DemoToolsPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not reach demo service: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var result struct {
			Tools []string `json:"tools"`
		}
		if err := json.Unmarshal(body, &result); err != nil || len(result.Tools) == 0 {
			fmt.Fprintln(os.Stderr, "No demo tools available.")
			os.Exit(1)
		}

		printDemoList(result.Tools)
		return nil
	},
}

// printDemoList groups tools by their prefix (the part before the first dot)
// and prints them in a clean, scannable format.
func printDemoList(tools []string) {
	// Group by prefix: "stripe.create_payment" → group "stripe"
	groups := make(map[string][]string)
	for _, t := range tools {
		prefix := t
		if i := strings.Index(t, "."); i > 0 {
			prefix = t[:i]
		}
		groups[prefix] = append(groups[prefix], t)
	}

	// Sort group names for stable output
	prefixes := make([]string, 0, len(groups))
	for p := range groups {
		prefixes = append(prefixes, p)
	}
	sort.Strings(prefixes)

	fmt.Println("\nAvailable demos:\n")
	for _, prefix := range prefixes {
		// Capitalise first letter without the deprecated strings.Title
		label := strings.ToUpper(prefix[:1]) + prefix[1:]
		fmt.Printf("  %s\n", label)
		for _, tool := range groups[prefix] {
			fmt.Printf("  → %s\n", tool)
		}
		fmt.Println()
	}
	fmt.Println("Run any demo:")
	fmt.Println("  npx swytchcode <tool>")
	fmt.Println()
}

func init() {
	demoCmd.AddCommand(demoListCmd)
}
