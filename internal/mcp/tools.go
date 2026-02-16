// tools.go implements MCP tool handlers that wrap CLI commands.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gitlab.com/swytchcode/shell/internal/constants"
	"gitlab.com/swytchcode/shell/internal/kernel"
	"gitlab.com/swytchcode/shell/internal/commands"
	"gitlab.com/swytchcode/shell/internal/registry"
	"gitlab.com/swytchcode/shell/internal/util"
)

// ToolOutput represents the output from MCP tools (matches CLI output).
type ToolOutput struct {
	Output string `json:"output"`
}

// ListArgs represents arguments for swytchcode_list.
type ListArgs struct {
	Filter *string `json:"filter,omitempty" jsonschema:"Filter type: 'methods', 'workflows', 'integrations', or empty for all"`
	Prefix *string `json:"prefix,omitempty" jsonschema:"Project prefix filter (e.g., 'stripe')"`
	JSON   *bool   `json:"json,omitempty" jsonschema:"Output as JSON object"`
}

// GetArgs represents arguments for swytchcode_get.
type GetArgs struct {
	ProjectName string `json:"project_name" jsonschema:"Project name to fetch"`
	Yes         *bool  `json:"yes,omitempty" jsonschema:"Auto-confirm overwrite"`
}

// AddArgs represents arguments for swytchcode_add.
type AddArgs struct {
	CanonicalID     string  `json:"canonical_id" jsonschema:"Canonical ID of the tool to add"`
	IntegrationSpec *string `json:"integration_spec,omitempty" jsonschema:"Optional integration spec (project@library.version)"`
}

// SearchArgs represents arguments for swytchcode_search.
type SearchArgs struct {
	Filter  *string `json:"filter,omitempty" jsonschema:"Filter type: 'integrations' or 'methods'"`
	Keyword *string `json:"keyword,omitempty" jsonschema:"Search keyword"`
	JSON    *bool   `json:"json,omitempty" jsonschema:"Output as JSON array"`
}

// InitArgs represents arguments for swytchcode_init.
type InitArgs struct {
	Editor string `json:"editor" jsonschema:"Editor choice: 'cursor', 'copilot', 'claude', or 'none'"`
	Mode   string `json:"mode" jsonschema:"Execution mode: 'production' or 'sandbox'"`
}

// BootstrapArgs represents arguments for swytchcode_bootstrap (no arguments needed).
type BootstrapArgs struct{}

// VersionArgs represents arguments for swytchcode_version (no arguments needed).
type VersionArgs struct{}

// InfoArgs represents arguments for swytchcode_info.
type InfoArgs struct {
	CanonicalID string `json:"canonical_id" jsonschema:"Canonical ID of the tool to get information about"`
	JSON        *bool  `json:"json,omitempty" jsonschema:"Output as JSON array instead of human-readable format"`
}

// ExecArgs represents arguments for swytchcode_exec.
type ExecArgs struct {
	Tool      string                 `json:"tool" jsonschema:"Canonical ID of the tool to execute"`
	Args      map[string]interface{} `json:"args,omitempty" jsonschema:"Tool arguments (body, params, Authorization, etc.)"`
	DryRun    *bool                  `json:"dry_run,omitempty" jsonschema:"Show what would be executed without making HTTP call"`
	Raw       *bool                  `json:"raw,omitempty" jsonschema:"Output raw HTTP response instead of normalized JSON"`
	AllowRaw  *bool                  `json:"allow_raw,omitempty" jsonschema:"Allow execution of raw methods"`
	JSON      *bool                  `json:"json,omitempty" jsonschema:"Output response as a single JSON object"`
}

// RegisterTools registers all MCP tools with the server.
// Registers 9 tools total:
//   1. swytchcode_init
//   2. swytchcode_bootstrap
//   3. swytchcode_version
//   4. swytchcode_list
//   5. swytchcode_search
//   6. swytchcode_get
//   7. swytchcode_add
//   8. swytchcode_exec
//   9. swytchcode_info
func RegisterTools(server *mcp.Server, streamOutput bool) error {
	// swytchcode_init
	mcp.AddTool(server, &mcp.Tool{
		Name:        "swytchcode_init",
		Description: "Initialize Swytchcode in this project. Creates .swytchcode/, tooling.json, and editor-specific config.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args InitArgs) (*mcp.CallToolResult, ToolOutput, error) {
		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return nil, ToolOutput{}, fmt.Errorf("detect project root: %w", err)
		}

		oc := NewOutputCapture(streamOutput)
		if err := commands.RunInit(projectRoot, args.Editor, args.Mode, oc.Stdout()); err != nil {
			return nil, ToolOutput{}, err
		}

		result := oc.GetCombinedOutput()
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, ToolOutput{Output: result}, nil
	})

	// swytchcode_bootstrap
	mcp.AddTool(server, &mcp.Tool{
		Name:        "swytchcode_bootstrap",
		Description: "Fetch all integrations declared in tooling.json that are not already installed.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args BootstrapArgs) (*mcp.CallToolResult, ToolOutput, error) {
		toolCtx, cancel := context.WithTimeout(ctx, constants.MCPRequestTimeout)
		defer cancel()

		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return nil, ToolOutput{}, fmt.Errorf("detect project root: %w", err)
		}

		oc := NewOutputCapture(streamOutput)
		if err := commands.RunBootstrap(toolCtx, projectRoot, oc.Stdout(), oc.Stderr()); err != nil {
			return nil, ToolOutput{}, err
		}

		result := oc.GetCombinedOutput()
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, ToolOutput{Output: result}, nil
	})

	// swytchcode_version
	mcp.AddTool(server, &mcp.Tool{
		Name:        "swytchcode_version",
		Description: "Get Swytchcode version.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args VersionArgs) (*mcp.CallToolResult, ToolOutput, error) {
		oc := NewOutputCapture(streamOutput)
		fmt.Fprintf(oc.Stdout(), "swytchcode version %s\n", constants.Version)

		result := oc.GetCombinedOutput()
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, ToolOutput{Output: result}, nil
	})

	// swytchcode_list
	mcp.AddTool(server, &mcp.Tool{
		Name:        "swytchcode_list",
		Description: "List locally available tools and integrations (from tooling.json and fetched integrations). No registry calls.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ListArgs) (*mcp.CallToolResult, ToolOutput, error) {
		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return nil, ToolOutput{}, fmt.Errorf("detect project root: %w", err)
		}

		filter := ""
		if args.Filter != nil {
			filter = *args.Filter
		}
		prefix := ""
		if args.Prefix != nil {
			prefix = *args.Prefix
		}
		jsonOutput := false
		if args.JSON != nil {
			jsonOutput = *args.JSON
		}

		oc := NewOutputCapture(streamOutput)
		_, err = commands.RunList(projectRoot, filter, prefix, jsonOutput, oc.Stdout())
		if err != nil {
			return nil, ToolOutput{}, err
		}

		result := oc.GetCombinedOutput()
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, ToolOutput{Output: result}, nil
	})

	// swytchcode_search
	mcp.AddTool(server, &mcp.Tool{
		Name:        "swytchcode_search",
		Description: "Search remote registry for available integrations. Read-only, never mutates local state.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args SearchArgs) (*mcp.CallToolResult, ToolOutput, error) {
		toolCtx, cancel := context.WithTimeout(ctx, constants.MCPRequestTimeout)
		defer cancel()

		oc := NewOutputCapture(streamOutput)
		argsMap := map[string]interface{}{}
		if args.Filter != nil {
			argsMap["filter"] = *args.Filter
		}
		if args.Keyword != nil {
			argsMap["keyword"] = *args.Keyword
		}
		if args.JSON != nil {
			argsMap["json"] = *args.JSON
		}
		result, err := handleSearch(toolCtx, argsMap, oc)
		if err != nil {
			return nil, ToolOutput{}, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, ToolOutput{Output: result}, nil
	})

	// swytchcode_get
	mcp.AddTool(server, &mcp.Tool{
		Name:        "swytchcode_get",
		Description: "Fetch and install integration bundles for a project",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args GetArgs) (*mcp.CallToolResult, ToolOutput, error) {
		toolCtx, cancel := context.WithTimeout(ctx, constants.MCPRequestTimeout)
		defer cancel()

		oc := NewOutputCapture(streamOutput)
		argsMap := map[string]interface{}{
			"project_name": args.ProjectName,
		}
		if args.Yes != nil {
			argsMap["yes"] = *args.Yes
		}
		result, err := handleGet(toolCtx, argsMap, oc)
		if err != nil {
			return nil, ToolOutput{}, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, ToolOutput{Output: result}, nil
	})

	// swytchcode_add
	mcp.AddTool(server, &mcp.Tool{
		Name:        "swytchcode_add",
		Description: "Add a tool (method or workflow) to tooling.json by canonical ID",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args AddArgs) (*mcp.CallToolResult, ToolOutput, error) {
		toolCtx, cancel := context.WithTimeout(ctx, constants.MCPRequestTimeout)
		defer cancel()

		oc := NewOutputCapture(streamOutput)
		argsMap := map[string]interface{}{
			"canonical_id": args.CanonicalID,
		}
		if args.IntegrationSpec != nil {
			argsMap["integration_spec"] = *args.IntegrationSpec
		}
		result, err := handleAdd(toolCtx, argsMap, oc)
		if err != nil {
			return nil, ToolOutput{}, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, ToolOutput{Output: result}, nil
	})

	// swytchcode_exec
	mcp.AddTool(server, &mcp.Tool{
		Name:        "swytchcode_exec",
		Description: "Execute a tool via the Swytchcode kernel. Use dry_run: true to see the planned request (method, url, headers, body) without making the HTTP call. The tool result content is always the full stdout/stderr output (dry-run payload or execution result).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ExecArgs) (*mcp.CallToolResult, ToolOutput, error) {
		toolCtx, cancel := context.WithTimeout(ctx, constants.MCPRequestTimeout)
		defer cancel()

		oc := NewOutputCapture(streamOutput)
		argsMap := map[string]interface{}{
			"tool": args.Tool,
		}
		if args.Args != nil {
			argsMap["args"] = args.Args
		}
		if args.DryRun != nil {
			argsMap["dry_run"] = *args.DryRun
		}
		if args.Raw != nil {
			argsMap["raw"] = *args.Raw
		}
		if args.AllowRaw != nil {
			argsMap["allow_raw"] = *args.AllowRaw
		}
		if args.JSON != nil {
			argsMap["json"] = *args.JSON
		}
		result, err := handleExec(toolCtx, argsMap, oc)
		output := oc.GetCombinedOutput()
		if err != nil {
			// Return captured output (stderr with kernel JSON error) so the client can see it; mark as error.
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: output},
				},
				IsError: true,
			}, ToolOutput{Output: output}, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, ToolOutput{Output: result}, nil
	})

	// swytchcode_info
	mcp.AddTool(server, &mcp.Tool{
		Name:        "swytchcode_info",
		Description: "Get information about a tool by canonical ID. Searches all fetched integrations and returns tool details from wrekenfiles.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args InfoArgs) (*mcp.CallToolResult, ToolOutput, error) {
		toolCtx, cancel := context.WithTimeout(ctx, constants.MCPRequestTimeout)
		defer cancel()

		oc := NewOutputCapture(streamOutput)
		toolInfos, err := commands.RunInfo(toolCtx, args.CanonicalID, oc.Stdout(), oc.Stderr())
		if err != nil {
			return nil, ToolOutput{}, err
		}

		jsonOutput := false
		if args.JSON != nil {
			jsonOutput = *args.JSON
		}

		if err := commands.FormatInfoOutput(toolInfos, jsonOutput, oc.Stdout()); err != nil {
			return nil, ToolOutput{}, err
		}

		result := oc.GetCombinedOutput()
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, ToolOutput{Output: result}, nil
	})

	return nil
}

// handleSearch handles swytchcode_search tool (remote registry queries).
func handleSearch(ctx context.Context, args map[string]interface{}, oc *OutputCapture) (string, error) {
	projectRoot, err := util.ProjectRoot()
	if err != nil {
		return "", fmt.Errorf("detect project root: %w", err)
	}

	var keyword string
	if k, ok := args["keyword"].(string); ok {
		keyword = k
	}

	regClient := registry.NewClient(registry.ConfigFromProjectRoot(projectRoot))
	integrationsResp, err := regClient.ListIntegrations(ctx)
	if err != nil {
		return "", fmt.Errorf("search integrations: %w", err)
	}

	// Collect unique project names (with keyword filter)
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

	jsonOutput := false
	if jsonRaw, ok := args["json"].(bool); ok {
		jsonOutput = jsonRaw
	}

	if jsonOutput {
		if err := json.NewEncoder(oc.Stdout()).Encode(projectNames); err != nil {
			return "", fmt.Errorf("encode JSON: %w", err)
		}
	} else {
		for _, projectName := range projectNames {
			fmt.Fprintln(oc.Stdout(), projectName)
		}
	}

	return oc.GetCombinedOutput(), nil
}

// handleGet handles swytchcode_get tool.
func handleGet(ctx context.Context, args map[string]interface{}, oc *OutputCapture) (string, error) {
	projectName, ok := args["project_name"].(string)
	if !ok || projectName == "" {
		return "", fmt.Errorf("project_name is required")
	}

	yes := false
	if yesRaw, ok := args["yes"].(bool); ok {
		yes = yesRaw
	}

	err := commands.RunGet(ctx, projectName, yes, oc.Stdout(), oc.Stderr())
	if err != nil {
		return "", err
	}

	return oc.GetCombinedOutput(), nil
}

// handleAdd handles swytchcode_add tool.
func handleAdd(ctx context.Context, args map[string]interface{}, oc *OutputCapture) (string, error) {
	canonicalID, ok := args["canonical_id"].(string)
	if !ok || canonicalID == "" {
		return "", fmt.Errorf("canonical_id is required")
	}

	integrationSpec, _ := args["integration_spec"].(string)

	err := runAddCommand(ctx, canonicalID, integrationSpec, oc.Stdout(), oc.Stderr())
	if err != nil {
		return "", err
	}

	return oc.GetCombinedOutput(), nil
}

// handleExec handles swytchcode_exec tool.
func handleExec(ctx context.Context, args map[string]interface{}, oc *OutputCapture) (string, error) {
	_ = ctx // Context is passed for consistency but kernel.Execute doesn't use it
	tool, ok := args["tool"].(string)
	if !ok || tool == "" {
		return "", fmt.Errorf("tool is required")
	}

	// Build exec request matching CLI format
	execReq := kernel.ExecRequest{
		Tool: tool,
		Args: make(map[string]interface{}),
	}

	// Parse args object
	if argsRaw, ok := args["args"].(map[string]interface{}); ok {
		execReq.Args = argsRaw
	}

	// Parse flags
	dryRun := false
	if dryRunRaw, ok := args["dry_run"].(bool); ok {
		dryRun = dryRunRaw
	}

	rawOutput := false
	if rawRaw, ok := args["raw"].(bool); ok {
		rawOutput = rawRaw
	}

	allowRaw := false
	if allowRawRaw, ok := args["allow_raw"].(bool); ok {
		allowRaw = allowRawRaw
	}

	jsonOutput := false
	if jsonRaw, ok := args["json"].(bool); ok {
		jsonOutput = jsonRaw
	}

	// Convert exec request to JSON for kernel.Execute
	reqJSON, err := json.Marshal(execReq)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	// Create a reader from JSON
	reqReader := &jsonReader{data: reqJSON}
	exitCode := kernel.Execute(reqReader, oc.Stdout(), oc.Stderr(), allowRaw, dryRun, rawOutput, jsonOutput, "")

	if exitCode != kernel.ExitCodeOK {
		log.Printf("[swytchcode_exec] failed tool=%s exit_code=%d (stderr captured)", tool, exitCode)
		return "", fmt.Errorf("execution failed with exit code %d", exitCode)
	}

	return oc.GetCombinedOutput(), nil
}

// jsonReader implements io.Reader for JSON data
type jsonReader struct {
	data []byte
	pos  int
}

func (r *jsonReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	if r.pos >= len(r.data) {
		err = io.EOF
	}
	return n, err
}

// runAddCommand runs the add command logic.
func runAddCommand(ctx context.Context, canonicalID, integrationSpec string, stdout, stderr io.Writer) error {
	return commands.RunAdd(ctx, canonicalID, integrationSpec, stdout, stderr)
}
