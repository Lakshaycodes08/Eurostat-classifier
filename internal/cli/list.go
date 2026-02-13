// list.go implements swytchcode list: lists available integrations from the registry.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/swytchcode/shell/internal/registry"
	"gitlab.com/swytchcode/shell/internal/util"
)

var (
	listJSON bool
)

// listCmd implements `swytchcode list`.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available integrations",
	Long:  "Lists all available integrations from the registry.",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return fmt.Errorf("detect project root: %w", err)
		}

		regClient := registry.NewClient(registry.ConfigFromProjectRoot(projectRoot))
		ctx := context.Background()

		integrationsResp, err := regClient.ListIntegrations(ctx)
		if err != nil {
			return fmt.Errorf("fetch integrations: %w", err)
		}

		if listJSON {
			// Output as JSON array of IDs
			ids := make([]string, len(integrationsResp.Integrations))
			for i, integration := range integrationsResp.Integrations {
				ids[i] = integration.ID
			}
			if err := json.NewEncoder(os.Stdout).Encode(ids); err != nil {
				return fmt.Errorf("encode JSON: %w", err)
			}
		} else {
			// Output one ID per line
			for _, integration := range integrationsResp.Integrations {
				fmt.Println(integration.ID)
			}
		}

		return nil
	},
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "output as JSON array")
}
