// tools.go implements MCP tool handlers that wrap CLI commands.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gitlab.com/swytchcode/swytchcode-cli/internal/auth"
	"gitlab.com/swytchcode/swytchcode-cli/internal/commands"
	"gitlab.com/swytchcode/swytchcode-cli/internal/constants"
	"gitlab.com/swytchcode/swytchcode-cli/internal/kernel"
	"gitlab.com/swytchcode/swytchcode-cli/internal/registry"
	"gitlab.com/swytchcode/swytchcode-cli/internal/util"
)


// RegisterTools registers all MCP tools with the server.
// Registers 16 tools total:
//  1. swytchcode_init
//  2. swytchcode_bootstrap
//  3. swytchcode_version
//  4. swytchcode_list
//  5. swytchcode_search
//  6. swytchcode_get
//  7. swytchcode_add
//  8. swytchcode_exec
//  9. swytchcode_check
//  10. swytchcode_inspect
//  11. swytchcode_upgrade
//  12. swytchcode_diff
//  13. swytchcode_discover
//  14. swytchcode_plan
//  15. swytchcode_doctor
//  16. swytchcode_info
func RegisterTools(server *mcp.Server, streamOutput bool) error {
	// swytchcode_init
	mcp.AddTool(server, &mcp.Tool{
		Name:        "swytchcode_init",
		Description: "Initialize Swytchcode in this project. Creates .swytchcode/, tooling.json, and editor-specific config.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args InitArgs) (*mcp.CallToolResult, ToolOutput, error) {
		projectRoot, err := util.InitProjectRoot()
		if err != nil {
			return nil, ToolOutput{}, fmt.Errorf("detect project root: %w", err)
		}

		oc := NewOutputCapture(streamOutput)
		if err := commands.RunInit(projectRoot, args.Editor, args.Mode, oc.Stdout(), oc.Stderr()); err != nil {
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

		filter := args.Filter
		prefix := args.Prefix
		jsonOutput := args.JSON

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
		argsMap := map[string]interface{}{
			"keyword": args.Keyword,
			"json":    args.JSON,
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
			"yes":          args.Yes,
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
		Description: "Add a tool (method or workflow) to tooling.json by canonical ID. Set all=true and provide a project name as canonical_id to add all tools from a project at once.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args AddArgs) (*mcp.CallToolResult, ToolOutput, error) {
		toolCtx, cancel := context.WithTimeout(ctx, constants.MCPRequestTimeout)
		defer cancel()

		oc := NewOutputCapture(streamOutput)

		if args.All {
			if args.CanonicalID == "" {
				return nil, ToolOutput{}, fmt.Errorf("canonical_id (project name) is required when all=true")
			}
			if err := commands.RunAddAll(toolCtx, args.CanonicalID, oc.Stdout(), oc.Stderr()); err != nil {
				return nil, ToolOutput{}, err
			}
			result := oc.GetCombinedOutput()
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: result}},
			}, ToolOutput{Output: result}, nil
		}

		argsMap := map[string]interface{}{
			"canonical_id": args.CanonicalID,
		}
		if args.IntegrationSpec != "" {
			argsMap["integration_spec"] = args.IntegrationSpec
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
		Description: "Execute a tool via the Swytchcode kernel. Use dry_run: true to see the planned request (method, url, headers, body) without making the HTTP call. Use verbose: true to log full request/response details to stderr (sensitive headers redacted). The tool result content is always the full stdout/stderr output (dry-run payload or execution result).",
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
		argsMap["dry_run"] = args.DryRun
		argsMap["raw"] = args.Raw
		argsMap["allow_raw"] = args.AllowRaw
		argsMap["json"] = args.JSON
		argsMap["verbose"] = args.Verbose
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

	// swytchcode_check
	mcp.AddTool(server, &mcp.Tool{
		Name:        "swytchcode_check",
		Description: "Check for integration update proposals from the TinyFish agent. Exits with hasBreaking=true if any major (breaking) changes exist. Optionally filter by project or project.library name.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args CheckArgs) (*mcp.CallToolResult, ToolOutput, error) {
		apiURL := os.Getenv("SWYTCHCODE_API_URL")
		if apiURL == "" {
			apiURL = constants.RegistryURL
		}

		token, _, _ := auth.ResolveToken()

		oc := NewOutputCapture(streamOutput)
		hasBreaking, err := commands.RunCheck(commands.CheckConfig{
			APIURL:  apiURL,
			Token:   token,
			Library: args.Library,
		}, oc.Stdout())
		if err != nil {
			return nil, ToolOutput{}, err
		}

		result := oc.GetCombinedOutput()
		toolResult := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}
		if hasBreaking {
			toolResult.IsError = true
		}
		return toolResult, ToolOutput{Output: result}, nil
	})

	// swytchcode_inspect
	mcp.AddTool(server, &mcp.Tool{
		Name:        "swytchcode_inspect",
		Description: "Show full proposal detail for a specific library. Requires user login or SWYTCHCODE_TOKEN.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args InspectArgs) (*mcp.CallToolResult, ToolOutput, error) {
		if args.Library == "" {
			return nil, ToolOutput{}, fmt.Errorf("library is required")
		}

		apiURL := os.Getenv("SWYTCHCODE_API_URL")
		if apiURL == "" {
			apiURL = constants.RegistryURL
		}

		token, _, _ := auth.ResolveToken()
		if token == "" {
			return nil, ToolOutput{}, fmt.Errorf("not authenticated — run `swytchcode login` or set SWYTCHCODE_TOKEN")
		}

		oc := NewOutputCapture(streamOutput)

		proposals, err := commands.FetchProposals(commands.CheckConfig{
			APIURL:  apiURL,
			Token:   token,
			Library: args.Library,
		})
		if err != nil {
			return nil, ToolOutput{}, err
		}

		var proposalUUID string
		for _, p := range proposals {
			if strings.EqualFold(p.LibraryName, args.Library) {
				proposalUUID = p.ProposalUUID
				break
			}
		}
		if proposalUUID == "" {
			fmt.Fprintf(oc.Stdout(), "No proposals found for %s\n", args.Library)
			result := oc.GetCombinedOutput()
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: result}},
			}, ToolOutput{Output: result}, nil
		}

		detail, err := commands.FetchProposalDetail(apiURL, token, proposalUUID)
		if err != nil {
			return nil, ToolOutput{}, err
		}

		from := detail.CurrentVersion
		if from == "" {
			from = "unknown"
		}
		to := detail.ProposedVersion
		if to == "" {
			to = "unknown"
		}
		header := fmt.Sprintf("%s   %s -> %s   [%s]   confidence: %.2f",
			args.Library, from, to, detail.Impact, detail.Confidence)
		divider := strings.Repeat("─", len(header))
		fmt.Fprintf(oc.Stdout(), "%s\n%s\n", header, divider)
		if detail.Summary != "" {
			fmt.Fprintf(oc.Stdout(), "Summary:  %s\n", detail.Summary)
		}
		if detail.ChangeSet != nil {
			fmt.Fprintf(oc.Stdout(), "Status:   %s\n", detail.Status)
		}

		result := oc.GetCombinedOutput()
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: result}},
		}, ToolOutput{Output: result}, nil
	})

	// swytchcode_upgrade
	mcp.AddTool(server, &mcp.Tool{
		Name:        "swytchcode_upgrade",
		Description: "Approve a pending integration upgrade proposal. Requires user login (not a service token). Set confirm=true to approve.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args UpgradeArgs) (*mcp.CallToolResult, ToolOutput, error) {
		if args.Library == "" {
			return nil, ToolOutput{}, fmt.Errorf("library is required")
		}

		apiURL := os.Getenv("SWYTCHCODE_API_URL")
		if apiURL == "" {
			apiURL = constants.RegistryURL
		}

		token, _, _ := auth.ResolveToken()
		if token == "" {
			return nil, ToolOutput{}, fmt.Errorf("not authenticated — run `swytchcode login`")
		}

		oc := NewOutputCapture(streamOutput)
		confirm := args.Confirm
		err := commands.RunUpgrade(commands.UpgradeConfig{
			APIURL:  apiURL,
			Token:   token,
			Library: args.Library,
			Apply:   args.Apply,
			Stderr:  oc.Stderr(),
		}, func(_ string) bool { return confirm }, oc.Stdout())
		if err != nil {
			return nil, ToolOutput{}, err
		}

		result := oc.GetCombinedOutput()
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: result}},
		}, ToolOutput{Output: result}, nil
	})

	// swytchcode_diff
	mcp.AddTool(server, &mcp.Tool{
		Name:        "swytchcode_diff",
		Description: "Show method-level signature changes in a pending upgrade proposal for a library. Use before swytchcode_upgrade to assess impact. Requires auth.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args DiffArgs) (*mcp.CallToolResult, ToolOutput, error) {
		if args.Library == "" {
			return nil, ToolOutput{}, fmt.Errorf("library is required")
		}

		token, _, err := auth.ResolveToken()
		if err != nil || token == "" {
			return nil, ToolOutput{}, fmt.Errorf("not authenticated — run `swytchcode login` or set SWYTCHCODE_TOKEN")
		}

		toolCtx, cancel := context.WithTimeout(ctx, constants.MCPRequestTimeout)
		defer cancel()

		oc := NewOutputCapture(streamOutput)
		if err := commands.RunDiff(toolCtx, args.Library, token, oc.Stdout(), oc.Stderr()); err != nil {
			return nil, ToolOutput{}, err
		}

		result := oc.GetCombinedOutput()
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: result}},
		}, ToolOutput{Output: result}, nil
	})

	// swytchcode_discover
	mcp.AddTool(server, &mcp.Tool{
		Name:        "swytchcode_discover",
		Description: "Discover API capabilities by natural language intent. Returns matching methods and a recommended workflow. Use before swytchcode_exec when you don't know the exact canonical ID.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args DiscoverArgs) (*mcp.CallToolResult, ToolOutput, error) {
		toolCtx, cancel := context.WithTimeout(ctx, constants.MCPRequestTimeout)
		defer cancel()

		topK := args.Top
		if topK == 0 {
			topK = 5
		}

		oc := NewOutputCapture(streamOutput)
		if err := commands.RunDiscover(toolCtx, args.Intent, args.Project, args.Library, topK, args.JSON, oc.Stdout(), oc.Stderr()); err != nil {
			return nil, ToolOutput{}, err
		}

		result := oc.GetCombinedOutput()
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: result}},
		}, ToolOutput{Output: result}, nil
	})

	// swytchcode_plan
	mcp.AddTool(server, &mcp.Tool{
		Name:        "swytchcode_plan",
		Description: "Show the steps for a workflow by its canonical ID. Use after swytchcode_discover to inspect a recommended workflow before running it.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args PlanArgs) (*mcp.CallToolResult, ToolOutput, error) {
		toolCtx, cancel := context.WithTimeout(ctx, constants.MCPRequestTimeout)
		defer cancel()

		if args.CanonicalID == "" {
			return nil, ToolOutput{}, fmt.Errorf("canonical_id is required")
		}

		oc := NewOutputCapture(streamOutput)
		if err := commands.RunPlan(toolCtx, args.CanonicalID, args.Project, args.JSON, oc.Stdout(), oc.Stderr()); err != nil {
			return nil, ToolOutput{}, err
		}

		result := oc.GetCombinedOutput()
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: result}},
		}, ToolOutput{Output: result}, nil
	})

	// swytchcode_doctor
	mcp.AddTool(server, &mcp.Tool{
		Name:        "swytchcode_doctor",
		Description: "Run local diagnostics: tooling.json, integration bundles, manifest.json, HTTPS base URLs, and auth posture. Use json=true for machine-readable output. Fails if any error-level check fails (CI-friendly).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args DoctorArgs) (*mcp.CallToolResult, ToolOutput, error) {
		projectRoot, err := util.ProjectRoot()
		if err != nil {
			return nil, ToolOutput{}, fmt.Errorf("detect project root: %w", err)
		}
		rep := commands.RunDoctor(projectRoot)
		var out string
		if args.JSON {
			b, e := json.MarshalIndent(rep, "", "  ")
			if e != nil {
				return nil, ToolOutput{}, e
			}
			out = string(b)
		} else {
			out = commands.FormatDoctorHuman(rep)
		}
		result := &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: out}}}
		to := ToolOutput{Output: out}
		if !rep.OK {
			return result, to, fmt.Errorf("doctor: one or more checks failed")
		}
		return result, to, nil
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

		jsonOutput := args.JSON

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
	var keyword string
	if k, ok := args["keyword"].(string); ok {
		keyword = k
	}

	regClient := registry.NewClient(registry.DefaultConfig())
	integrationsResp, err := regClient.ListIntegrations(ctx)
	if err != nil {
		return "", fmt.Errorf("search: %w", err)
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

	verbose := false
	if verboseRaw, ok := args["verbose"].(bool); ok {
		verbose = verboseRaw
	}

	// Convert exec request to JSON for kernel.Execute
	reqJSON, err := json.Marshal(execReq)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	// Resolve auth token for registry calls (e.g. workflow execution)
	mcpToken, _, _ := auth.ResolveToken()

	// Create a reader from JSON
	reqReader := util.NewJSONReader(reqJSON)
	exitCode := kernel.Execute(reqReader, oc.Stdout(), oc.Stderr(), kernel.ExecOptions{
		AllowRaw: allowRaw, DryRun: dryRun, RawOutput: rawOutput, JSONOutput: jsonOutput, Token: mcpToken, Verbose: verbose,
	})

	if exitCode != kernel.ExitCodeOK {
		log.Printf("[swytchcode_exec] failed tool=%s exit_code=%d (stderr captured)", tool, exitCode)
		return "", fmt.Errorf("execution failed with exit code %d", exitCode)
	}

	return oc.GetCombinedOutput(), nil
}

// runAddCommand runs the add command logic.
func runAddCommand(ctx context.Context, canonicalID, integrationSpec string, stdout, stderr io.Writer) error {
	return commands.RunAdd(ctx, canonicalID, integrationSpec, false, stdout, stderr)
}