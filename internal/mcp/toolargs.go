// toolargs.go holds MCP tool argument types shared by RegisterTools.
package mcp

// ToolOutput represents the output from MCP tools (matches CLI output).
type ToolOutput struct {
	Output string `json:"output"`
}

// ListArgs represents arguments for swytchcode_list.
type ListArgs struct {
	Filter string `json:"filter,omitempty" jsonschema:"Filter type: 'methods', 'workflows', 'integrations', 'tooling' (enabled in tooling.json only), or empty for all"`
	Prefix string `json:"prefix,omitempty" jsonschema:"Project prefix filter (e.g., 'stripe')"`
	JSON   bool   `json:"json,omitempty" jsonschema:"Output as JSON object"`
}

// GetArgs represents arguments for swytchcode_get.
type GetArgs struct {
	ProjectName string `json:"project_name" jsonschema:"Project name to fetch"`
	Yes         bool   `json:"yes,omitempty" jsonschema:"Auto-confirm overwrite"`
}

// AddArgs represents arguments for swytchcode_add.
type AddArgs struct {
	CanonicalID     string `json:"canonical_id,omitempty" jsonschema:"Canonical ID of the tool to add (required unless all=true)"`
	IntegrationSpec string `json:"integration_spec,omitempty" jsonschema:"Optional integration spec (project@library.version)"`
	All             bool   `json:"all,omitempty" jsonschema:"When true, add all tools from the project specified by canonical_id (treated as project name)"`
}

// SearchArgs represents arguments for swytchcode_search.
type SearchArgs struct {
	Keyword string `json:"keyword,omitempty" jsonschema:"Optional search keyword; if empty, returns all integrations"`
	JSON    bool   `json:"json,omitempty" jsonschema:"Output as JSON array"`
}

// InitArgs represents arguments for swytchcode_init.
type InitArgs struct {
	Editor string `json:"editor" jsonschema:"Editor choice: 'cursor', 'claude', 'copilot', or 'none'"`
	Mode   string `json:"mode" jsonschema:"Execution mode: 'production' or 'sandbox'"`
}

// BootstrapArgs represents arguments for swytchcode_bootstrap (no arguments needed).
type BootstrapArgs struct{}

// VersionArgs represents arguments for swytchcode_version (no arguments needed).
type VersionArgs struct{}

// InfoArgs represents arguments for swytchcode_info.
type InfoArgs struct {
	CanonicalID string `json:"canonical_id" jsonschema:"Canonical ID of the tool to get information about"`
	JSON        bool   `json:"json,omitempty" jsonschema:"Output as JSON array instead of human-readable format"`
}

// CheckArgs represents arguments for swytchcode_check.
type CheckArgs struct {
	Library string `json:"library,omitempty" jsonschema:"Optional project or project.library filter"`
}

// InspectArgs represents arguments for swytchcode_inspect.
type InspectArgs struct {
	Library string `json:"library" jsonschema:"Project or project.library name to inspect"`
}

// UpgradeArgs represents arguments for swytchcode_upgrade.
type UpgradeArgs struct {
	Library string `json:"library" jsonschema:"Project or project.library name to upgrade"`
	Confirm bool   `json:"confirm" jsonschema:"Set to true to confirm the upgrade"`
	Apply   bool   `json:"apply,omitempty" jsonschema:"When true, automatically re-fetch integration bundle and re-add affected methods after approval"`
}

// DiffArgs represents arguments for swytchcode_diff.
type DiffArgs struct {
	Library string `json:"library" jsonschema:"Project or project.library name to show diff for"`
}

// DiscoverArgs represents arguments for swytchcode_discover.
type DiscoverArgs struct {
	Intent  string `json:"intent" jsonschema:"Natural language description of the capability to find"`
	Project string `json:"project,omitempty" jsonschema:"Optional project name to scope the search"`
	Library string `json:"library,omitempty" jsonschema:"Optional library name to scope the search within a project"`
	Top     int    `json:"top,omitempty" jsonschema:"Number of results to return (default 5)"`
	JSON    bool   `json:"json,omitempty" jsonschema:"Output as JSON"`
}

// PlanArgs represents arguments for swytchcode_plan.
type PlanArgs struct {
	CanonicalID string `json:"canonical_id" jsonschema:"Canonical workflow ID to show steps for"`
	Project     string `json:"project,omitempty" jsonschema:"Project name (defaults to prefix of canonical_id)"`
	JSON        bool   `json:"json,omitempty" jsonschema:"Output as JSON"`
}

// ExecArgs represents arguments for swytchcode_exec.
type ExecArgs struct {
	Tool     string                 `json:"tool" jsonschema:"Canonical ID of the tool to execute"`
	Args     map[string]interface{} `json:"args,omitempty" jsonschema:"Tool arguments (body, params, Authorization, etc.)"`
	DryRun   bool                   `json:"dry_run,omitempty" jsonschema:"Show what would be executed without making HTTP call"`
	Raw      bool                   `json:"raw,omitempty" jsonschema:"Output raw HTTP response instead of normalized JSON"`
	AllowRaw bool                   `json:"allow_raw,omitempty" jsonschema:"Allow execution of raw methods"`
	JSON     bool                   `json:"json,omitempty" jsonschema:"Output response as a single JSON object"`
	Verbose  bool                   `json:"verbose,omitempty" jsonschema:"Log request and response details to stderr (sensitive headers redacted)"`
}

// DoctorArgs represents arguments for swytchcode_doctor.
type DoctorArgs struct {
	JSON bool `json:"json,omitempty" jsonschema:"Emit JSON instead of human text"`
}
