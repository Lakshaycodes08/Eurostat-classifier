// search.go implements swytchcode search: searches remote registry for available integrations.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/swytchcode-cli/internal/registry"
)

var (
	searchJSON bool
)

// searchCmd implements `swytchcode search` - searches remote registry.
var searchCmd = &cobra.Command{
	Use:   "search [keyword]",
	Short: "Search remote registry for available integrations",
	Long:  "Searches the remote registry for available integrations. With no keyword, lists all; with a keyword, returns matching project names. Read-only, never mutates local state.",
	RunE: func(cmd *cobra.Command, args []string) error {
		keyword := ""
		if len(args) > 0 {
			keyword = args[0]
		}

		regClient := registry.NewClient(registry.DefaultConfig())
		ctx := context.Background()

		integrationsResp, err := regClient.ListIntegrations(ctx)
		if err != nil {
			return fmt.Errorf("search: %w", err)
		}

		// Collect unique project names (all if no keyword, else matching keyword)
		projectMap := make(map[string]bool)
		for _, project := range integrationsResp.Projects {
			if keyword == "" || strings.Contains(strings.ToLower(project.ProjectName), strings.ToLower(keyword)) {
				projectMap[project.ProjectName] = true
			}
		}
		projectNames := make([]string, 0, len(projectMap))
		for projectName := range projectMap {
			projectNames = append(projectNames, projectName)
		}

		if searchJSON {
			if err := json.NewEncoder(os.Stdout).Encode(projectNames); err != nil {
				return fmt.Errorf("encode JSON: %w", err)
			}
		} else {
			for _, projectName := range projectNames {
				fmt.Println(projectName)
			}
		}

		return nil
	},
}

func init() {
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "output as JSON array")
}
