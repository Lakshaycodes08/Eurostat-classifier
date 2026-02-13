# Swytchcode Kernel

Swytchcode is the **execution kernel** for tools. Editors, agents, and languages call `swytchcode exec` to execute tools deterministically.

**`tooling.json` defines what is trusted.**  
**Wrekenfiles define what is possible.**

## Commands

### `swytchcode init`

Initialize Swytchcode in this project. Creates `.swytchcode/` directory structure and `tooling.json`.

**Usage:**
```bash
# Interactive mode (prompts for editor and mode)
swytchcode init

# Non-interactive mode (CI)
swytchcode init --editor=cursor --mode=production --non-interactive
```

**Flags:**
- `--editor`: Editor choice (`cursor | vscode | claude | none`)
- `--mode`: Execution mode (`production | sandbox`)
- `--non-interactive`: Disable prompts (required for CI)

**What it does:**
- Creates `.swytchcode/` and `.swytchcode/integrations/` directories
- Creates `tooling.json` with empty `integrations` and `tools` maps
- Sets `mode` and `registry_url` in `tooling.json`
- Writes editor-specific configuration files (if editor ≠ `none`)

**Error messages:**
- `"init requires --editor when running non-interactively"` — Missing `--editor` flag in non-interactive mode
- `"init requires --mode when running non-interactively"` — Missing `--mode` flag in non-interactive mode
- `"invalid mode %q (expected production or sandbox)"` — Invalid mode value
- `"unknown editor %q (expected cursor|vscode|claude|none)"` — Invalid editor value

---

### `swytchcode get <project_name>`

Fetch and install integration bundles (Wrekenfiles, methods, workflows) for a project. Does **not** modify `tooling.json` — use `swytchcode add` to enable tools.

**Usage:**
```bash
# Interactive mode (prompts for project if not provided)
swytchcode get

# With project name
swytchcode get weaviate

# Non-interactive mode
swytchcode get weaviate --yes --non-interactive
```

**Flags:**
- `--yes`: Auto-confirm overwrite in non-interactive mode
- `--non-interactive`: Disable prompts

**What it does:**
- Shows spinner animation during fetching operations
- Fetches all integration bundles for the project from registry
- Saves to `.swytchcode/integrations/{project}/{library}/{version}/`:
  - `wrekenfile.yaml` — Wrekenfile spec with METHODS section
  - `methods.json` — Methods list for this integration version
  - `workflows.json` — Workflows list for this integration version
- Updates `.swytchcode/integrations/manifest.json` with `project.library` entries (version, `sandbox_endpoint`, `production_endpoint`, methods count, workflows count)
- Uses `sandbox_endpoint` and `production_endpoint` directly from bundle response (uses `http://localhost` if empty)

**Error messages:**
- `"library name required when running non-interactively"` — Missing project name in non-interactive mode
- `"no integrations available"` — Registry returned no integrations
- `"no bundles found for project %q"` — No bundles found for the specified project
- `"Version %q for %s/%s already exists; set yes parameter to true to overwrite"` — Integration version already exists (use `--yes` flag or set `yes` parameter to `true` in MCP)
- `"fetch available integrations: %w"` — Failed to fetch integrations from registry
- `"fetch integration bundles: %w"` — Failed to fetch bundles from registry
- `"Failed to fetch workflows: %v"` — Failed to fetch workflows
- `"Failed to fetch methods: %v"` — Failed to fetch methods

---

### `swytchcode bootstrap`

Fetch all integrations declared in `tooling.json` that are not already installed. Non-interactive command suitable for CI.

**Usage:**
```bash
swytchcode bootstrap
```

**What it does:**
- Reads `integrations` section from `tooling.json`
- Parses `project.library` keys and extracts project names
- Checks if integration already exists at `.swytchcode/integrations/{project}/{library}/{version}/`
- Fetches missing integrations using registry API (non-interactive)
- Shows spinner animation for each integration being fetched
- Fails fast if any fetch fails
- Prints summary: fetched, skipped, and failed integrations

**Error messages:**
- `"tooling.json not found; run 'swytchcode init' first: %w"` — Project not initialized
- `"parse tooling.json: %w"` — Invalid tooling.json format
- `"invalid integrations format in tooling.json"` — Invalid integrations section format
- `"%s (invalid format)"` — Invalid project.library format
- `"%s (invalid version format)"` — Invalid version format in integrations
- `"%s (missing version)"` — Missing version for integration
- `"%s: %v"` — Failed to fetch integration (shows error details)

---

### `swytchcode list`

List all available integrations from the registry.

**Usage:**
```bash
# Plain text output (one ID per line)
swytchcode list

# JSON output
swytchcode list --json
```

**Flags:**
- `--json`: Output as JSON array instead of one ID per line

**What it does:**
- Calls `GET /v2/shell/integrations` endpoint
- Outputs integration IDs (one per line by default, or JSON array with `--json`)

**Error messages:**
- `"fetch integrations: %w"` — Failed to fetch integrations from registry
- `"encode JSON: %w"` — Failed to encode JSON output

---

### `swytchcode add [integration_spec] <canonical_id>`

Add a tool (method or workflow) to `tooling.json` by canonical ID. Searches `methods.json` and `workflows.json` files across all fetched integrations.

**Usage:**
```bash
# Mode 1: Search all integrations (may prompt if ambiguous)
swytchcode add api.cluster.create

# Mode 2: Explicit integration spec (CI-safe)
swytchcode add weaviate@lyrid.v1 api.cluster.create

# Add integration version only (does not fetch)
swytchcode add integration weaviate@lyrid.v1
```

**What it does:**
- Searches `methods.json` and `workflows.json` files in `.swytchcode/integrations/`
- Determines tool type (method or workflow) based on which file contains the canonical_id
- Reads wrekenfile to get full tool details
- Adds tool entry to `tooling.json` with `type`, `integration`, `summary`, `desc`, and `inputs`
- Automatically adds integration to `integrations` section if not already present

**Error messages:**
- `"canonical ID %q not found in any fetched integrations.\nRun: swytchcode get <project>"` — Tool not found in any integration
- `"ambiguous canonical ID. Found in %d integrations:\n  %s\nUse: swytchcode add <integration@version> %s"` — Tool found in multiple integrations (non-interactive mode)
- `"invalid integration spec format: %q (expected: project@library.version)"` — Invalid integration spec format
- `"Integration %s not installed. Run: swytchcode get %s"` — Integration not fetched yet
- `"method %q not found in wrekenfile"` — Method not found in wrekenfile
- `"workflow %q not found in wrekenfile"` — Workflow not found in wrekenfile

---

### `swytchcode exec [canonical_id]`

Execute a tool via the Swytchcode kernel. The **only** execution path for tools. Pure, deterministic, non-interactive, and offline-capable.

**Usage:**
```bash
# CLI args mode
swytchcode exec api.cluster.create --body cluster.json --input Authorization="Bearer token123"

# With query params
swytchcode exec api.cluster.get --param id=cluster-123 --input Authorization="Bearer token123"

# JSON stdin mode
echo '{"tool":"api.cluster.create","args":{"body":{"name":"my-cluster"},"Authorization":"Bearer token123"}}' | swytchcode exec

# Dry-run (show what would be executed)
swytchcode exec api.cluster.create --body cluster.json --dry-run

# Raw output mode
swytchcode exec api.cluster.get --param id=123 --raw
```

**Flags:**
- `--allow-raw`: Required for executing raw methods (disabled by default)
- `--dry-run`: Show what would be executed without making HTTP call
- `--body <file>`: Path to JSON file containing request body
- `--input <key=value>`: Input key=value pairs (can be specified multiple times)
- `--param <key=value>`: Query parameter key=value pairs (can be specified multiple times)
- `--raw`: Output raw HTTP response instead of normalized JSON

**Execution pipeline:**
1. Parse request (from CLI args or JSON stdin)
2. Load `tooling.json` and resolve tool entry
3. Load integration bundle from `.swytchcode/integrations/{project}/{library}/{version}/wrekenfile.yaml`
4. Resolve method from Wreken `METHODS:` section by `canonical_id`
5. Get base URL from `manifest.json` based on `mode` (`sandbox_endpoint` if mode is "sandbox", `production_endpoint` otherwise)
6. Validate input schema against `tooling.json` inputs
7. Build HTTP request (method from Wreken, URL = baseURL + endpoint, headers, body from args)
8. Execute HTTP request (or dry-run)
9. Output JSON response (normalized or raw)

**Output format:**
- **Default**: Normalized JSON with `request`, `status_code`, and `data` fields
- **--raw**: Raw HTTP response with `request`, `status_code`, `status`, `headers`, and `body` (string)
- **--dry-run**: JSON showing `method`, `url`, `headers`, and `body` that would be sent

**Example output:**
```json
{
  "request": {
    "method": "POST",
    "url": "http://localhost/api/serverless/cloud/credential"
  },
  "status_code": 200,
  "data": { ... }
}
```

**Exit codes:**
- `0` — Success
- `1` — Invalid input
- `2` — Tool not found
- `3` — Auth missing/invalid (not currently used)
- `4` — SDK execution failure
- `5` — Internal error

**Error messages (JSON on stderr):**
- `{"error": "invalid json input"}` — Invalid JSON in stdin
- `{"error": "tool is required"}` — Missing tool in request
- `{"error": "tooling.json not found; run 'swytchcode init' first"}` — Project not initialized
- `{"error": "tool %q not found in tooling.json. Run: swytchcode add %s"}` — Tool not in tooling.json
- `{"error": "integration %s not installed. Run: swytchcode get %s"}` — Integration bundle not installed
- `{"error": "method %q not found in wrekenfile"}` — Method not found in wrekenfile
- `{"error": "manifest.json not found. Run: swytchcode get <project>"}` — Manifest not found
- `{"error": "no %s endpoint found for integration %q in manifest.json"}` — Endpoint missing for mode
- `{"error": "input validation failed: %s"}` — Input validation failed
- `{"error": "failed to build request: %s"}` — Failed to build HTTP request
- `{"error": "execution failed: %s"}` — HTTP execution failed
- `{"error": "raw method execution requires --allow-raw flag"}` — Raw method requires flag

---

### `swytchcode mcp serve`

Start the Model Context Protocol (MCP) server. Exposes swytchcode commands (`list`, `get`, `add`, `exec`) as MCP tools for agent communication.

**Usage:**
```bash
# Interactive mode (stdio transport, shows output)
swytchcode mcp serve

# Daemon mode (background, no output)
swytchcode mcp serve --daemon

# Daemon mode with log file
swytchcode mcp serve --daemon --log-file /path/to/mcp.log

# HTTP transport
swytchcode mcp serve --transport http --port 3000

# HTTP transport in daemon mode
swytchcode mcp serve --transport http --port 3000 --daemon
```

**Flags:**
- `--daemon` / `-d`: Run in daemon mode (background, no terminal output)
- `--log-file <path>`: Path to log file (only used in daemon mode; if not provided, logs are suppressed)
- `--transport <type>`: Transport type (`stdio` or `http`), default: `stdio`
- `--port <number>`: Port for HTTP transport, default: `3000`

**What it does:**
- Starts an MCP server exposing four tools:
  - `swytchcode_list` — List available integrations
  - `swytchcode_get` — Fetch integration bundles
  - `swytchcode_add` — Add tools to tooling.json
  - `swytchcode_exec` — Execute tools
- All tool output is captured and returned through the MCP protocol (not streamed to terminal)
- In daemon mode: suppresses logs and returns control immediately (logs to file if `--log-file` provided)
- Supports stdio transport (default) for direct process communication
- Supports HTTP transport with bearer token authentication

**MCP Tools:**

**swytchcode_list**
- Parameters: `json` (boolean, optional) — Output as JSON array
- Returns: CLI output as-is (one ID per line or JSON array)

**swytchcode_get**
- Parameters: `project_name` (string, required), `yes` (boolean, optional) — Auto-confirm overwrite
- Returns: CLI output as-is

**swytchcode_add**
- Parameters: `canonical_id` (string, required), `integration_spec` (string, optional) — Integration spec (project@library.version)
- Returns: CLI output as-is

**swytchcode_exec**
- Parameters:
  - `tool` (string, required) — Canonical ID of tool to execute
  - `args` (object, optional) — Tool arguments (body, params, Authorization, etc.)
  - `dry_run` (boolean, optional) — Show what would be executed
  - `raw` (boolean, optional) — Output raw HTTP response
  - `allow_raw` (boolean, optional) — Allow execution of raw methods
- Returns: CLI output as-is (matches `swytchcode exec` output format)

**Error messages:**
- `"create MCP server: %w"` — Failed to create server
- `"open log file: %w"` — Failed to open log file
- `"register tools: %w"` — Failed to register MCP tools
- HTTP transport errors: `"missing Authorization header"`, `"invalid authorization token"`, `"method not allowed"`

---

## Project Structure

After `swytchcode init`, the following structure is created:

```
.swytchcode/
├── tooling.json              # Project configuration (integrations, tools, mode, registry_url)
└── integrations/
    ├── manifest.json         # Registry manifest with project.library entries (version, endpoints)
    └── {project}/{library}/{version}/
        ├── wrekenfile.yaml   # Wrekenfile spec with METHODS section
        ├── methods.json      # Methods list for this integration version
        └── workflows.json    # Workflows list for this integration version
```

### `tooling.json` structure

```json
{
  "version": "1.0",
  "mode": "production",
  "registry_url": "https://localhost",
  "integrations": {
    "weaviate.lyrid": { "version": "v1" }
  },
  "tools": {
    "api.cluster.create": {
      "summary": "Create a new cluster instance",
      "integration": "weaviate.lyrid@v1",
      "type": "method",
      "desc": "Create a new cluster instance",
      "inputs": [
        {
          "Authorization": {
            "LOCATION": "header",
            "TYPE": "STRING"
          }
        }
      ]
    }
  }
}
```

- **mode**: Execution mode (`production` or `sandbox`) — determines which endpoint from `manifest.json` is used
- **integrations**: Pinned integration versions (keys are `project.library` format)
- **tools**: Unified map of all trusted tools, keyed by `canonical_id`

---

## Base URL Resolution

When executing a tool, the base URL is resolved from `manifest.json` based on the `mode` in `tooling.json`:

- If `mode` is `"sandbox"` → use `sandbox_endpoint` from `manifest.json`
- Otherwise → use `production_endpoint` from `manifest.json`

The full URL is constructed as: `baseURL + endpoint` (where `endpoint` comes from the Wreken METHOD definition).

---

## Error Handling

All errors are written to stderr as JSON:
```json
{"error": "error message here"}
```

Exit codes are stable and documented above. The kernel never prompts during execution and never calls the registry during `exec`.
