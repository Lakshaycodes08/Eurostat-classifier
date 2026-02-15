// tools.go implements MCP tool handlers that wrap CLI commands.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"

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
	JSON *bool `json:"json,omitempty" jsonschema:"Output as JSON array instead of one ID per line"`
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
func RegisterTools(server *mcp.Server, streamOutput bool) error {
	// swytchcode_list
	mcp.AddTool(server, &mcp.Tool{
		Name:        "swytchcode_list",
		Description: "List all available integrations from the registry",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ListArgs) (*mcp.CallToolResult, ToolOutput, error) {
		// Create context with timeout
		toolCtx, cancel := context.WithTimeout(ctx, constants.MCPRequestTimeout)
		defer cancel()

		oc := NewOutputCapture(streamOutput)
		argsMap := map[string]interface{}{}
		if args.JSON != nil {
			argsMap["json"] = *args.JSON
		}
		result, err := handleList(toolCtx, argsMap, oc)
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

	return nil
}

// handleList handles swytchcode_list tool.
func handleList(ctx context.Context, args map[string]interface{}, oc *OutputCapture) (string, error) {
	projectRoot, err := util.ProjectRoot()
	if err != nil {
		return "", fmt.Errorf("detect project root: %w", err)
	}

	regClient := registry.NewClient(registry.ConfigFromProjectRoot(projectRoot))
	integrationsResp, err := regClient.ListIntegrations(ctx)
	if err != nil {
		return "", fmt.Errorf("fetch integrations: %w", err)
	}

	jsonOutput := false
	if jsonRaw, ok := args["json"].(bool); ok {
		jsonOutput = jsonRaw
	}

	if jsonOutput {
		ids := make([]string, len(integrationsResp.Integrations))
		for i, integration := range integrationsResp.Integrations {
			ids[i] = integration.ID
		}
		if err := json.NewEncoder(oc.Stdout()).Encode(ids); err != nil {
			return "", fmt.Errorf("encode JSON: %w", err)
		}
	} else {
		for _, integration := range integrationsResp.Integrations {
			fmt.Fprintln(oc.Stdout(), integration.ID)
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
