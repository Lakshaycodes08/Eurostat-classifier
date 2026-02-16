// search.go implements swytchcode search: searches remote registry for available integrations.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/registry"
	"gitlab.com/swytchcode/shell/internal/util"
)

var (
	searchJSON bool
)

// searchCmd implements `swytchcode search` - searches remote registry.
var searchCmd = &cobra.Command{
	Use:   "search [integrations|methods] [keyword]",
	Short: "Search remote registry for available integrations",
	Long:  "Searches the remote registry for available integrations. Read-only, never mutates local state.",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return fmt.Errorf("detect project root: %w", err)
		}

		var filter string // "integrations" or "methods" (methods is optional for now)
		var keyword string

		if len(args) > 0 {
			filter = args[0]
			if len(args) > 1 {
				keyword = args[1]
			} else if filter != "integrations" && filter != "methods" {
				// If first arg is not a filter, treat it as keyword
				keyword = filter
				filter = "integrations"
			}
		} else {
			filter = "integrations"
		}

		regClient := registry.NewClient(registry.ConfigFromProjectRoot(projectRoot))
		ctx := context.Background()

		if filter == "integrations" || filter == "" {
			integrationsResp, err := regClient.ListIntegrations(ctx)
			if err != nil {
				return fmt.Errorf("search integrations: %w", err)
			}

			// Collect unique project names
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
		} else if filter == "methods" {
			// Methods search not implemented yet - return empty for now
			if searchJSON {
				fmt.Println("[]")
			}
		}

		return nil
	},
}

func init() {
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "output as JSON array")
}
