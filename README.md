# Swytchcode Kernel

Swytchcode is the **execution kernel** for tools. Editors, agents, and languages call `swytchcode exec` to execute tools deterministically.

**`tooling.json` defines what is trusted.**  
**Wrekenfiles define what is possible.**

## Commands at a glance

| Command | Purpose |
|--------|---------|
| `swytchcode -v` or `swytchcode --version` | Show Swytchcode version |
| `swytchcode init` | Create `.swytchcode/`, `tooling.json`, and editor rule files (Cursor / Claude / VS Code) |
| `swytchcode get <project>` | Fetch integration bundles (Wrekenfiles, methods, workflows) |
| `swytchcode bootstrap` | Fetch all integrations declared in `tooling.json` |
| `swytchcode list` | List locally available tools and integrations (from tooling.json and fetched integrations) |
| `swytchcode list methods [prefix]` | List locally available methods (optionally filtered by project prefix) |
| `swytchcode list workflows [prefix]` | List locally available workflows (optionally filtered by project prefix) |
| `swytchcode list integrations` | List locally fetched integrations |
| `swytchcode search [integrations] [keyword]` | Search remote registry for available integrations |
| `swytchcode add [spec] <canonical_id>` | Add a tool to `tooling.json` |
| `swytchcode info <canonical_id>` | Show information about a tool by canonical ID |
| `swytchcode exec [canonical_id]` | Execute a tool (CLI or JSON stdin); supports `--json`, `--raw`, `--dry-run` |
| `swytchcode mcp serve` | Start MCP server (stdio or HTTP); exposes `swytchcode_init`, `swytchcode_bootstrap`, `swytchcode_version`, `swytchcode_list`, `swytchcode_search`, `swytchcode_get`, `swytchcode_add`, `swytchcode_info`, `swytchcode_exec` |
| `swytchcode mcp status` | Check if MCP server is running (daemon mode) |
| `swytchcode mcp stop` | Stop MCP server (daemon mode) |

---

## Commands (detail)

### `swytchcode -v` or `swytchcode --version`

Display the Swytchcode version.

**Usage:**
```bash
swytchcode -v
# or
swytchcode --version
```

**Output:**
```
swytchcode version 0.0.1
```

The version is a build-time constant defined in `internal/constants/constants.go`. To change the version, update `constants.Version` and rebuild.

---

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
- `--editor`: Editor choice (`cursor | copilot | claude | none`)
- `--mode`: Execution mode (`production | sandbox`)
- `--non-interactive`: Disable prompts (required for CI)

**What it does:**
- Creates `.swytchcode/` and `.swytchcode/integrations/` directories
- Creates `tooling.json` with empty `integrations` and `tools` maps
- Sets `mode` and `version` in `tooling.json`
- Installs editor rule templates in the repo (if editor ≠ `none`):
  - **cursor** → `.cursor/rules/swytchcode.mdc`
  - **copilot** → `.github/instructions/swytchcode.md`
  - **claude** → `CLAUDE.md` (repo root)

**Error messages:**
- `"init requires --editor when running non-interactively"` — Missing `--editor` flag in non-interactive mode
- `"init requires --mode when running non-interactively"` — Missing `--mode` flag in non-interactive mode
- `"invalid mode %q (expected production or sandbox)"` — Invalid mode value
- `"unknown editor %q (expected cursor|copilot|claude|none)"` — Invalid editor value

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
- `"Version %q for %s/%s already exists; use --yes to overwrite"` — Integration version already exists (CLI: use `--yes`; MCP: set `yes` parameter to `true`)
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

List locally available tools and integrations. **No registry calls** — reads from `tooling.json` and `.swytchcode/integrations/` directory.

**Usage:**
```bash
# List everything (methods, workflows, integrations)
swytchcode list

# List only methods
swytchcode list methods

# List only workflows
swytchcode list workflows

# List only integrations
swytchcode list integrations

# Filter methods by project prefix
swytchcode list methods stripe

# Filter workflows by project prefix
swytchcode list workflows stripe

# JSON output
swytchcode list --json
swytchcode list methods --json
```

**Flags:**
- `--json`: Output as JSON object with `methods`, `workflows`, and `integrations` arrays

**What it does:**
- Reads `tooling.json` to list methods and workflows that are executable locally
- Scans `.swytchcode/integrations/` directory to list fetched integrations
- Supports filtering by type (methods/workflows/integrations)
- Supports prefix filtering to show tools from a specific project (e.g., `list methods stripe`)

**Output format:**

Default (human-readable):
```
Methods:
  stripe.customer.create
  stripe.customer.update

Workflows:
  stripe.checkout.session.create

Integrations:
  stripe.stripe@v1
  weaviate.lyrid@v1
```

JSON (`--json`):
```json
{
  "methods": ["stripe.customer.create", "stripe.customer.update"],
  "workflows": ["stripe.checkout.session.create"],
  "integrations": ["stripe.stripe@v1", "weaviate.lyrid@v1"]
}
```

**Error messages:**
- `"detect project root: %w"` — Failed to detect project root
- `"tooling.json not found; run 'swytchcode init' first"` — Project not initialized

---

### `swytchcode search`

Search remote registry for available integrations. **Read-only** — never mutates local state.

**Usage:**
```bash
# Search all available integrations
swytchcode search
swytchcode search integrations

# Search by keyword
swytchcode search stripe
swytchcode search integrations stripe

# JSON output
swytchcode search --json
```

**Flags:**
- `--json`: Output as JSON array instead of one project name per line

**What it does:**
- Calls `GET /v2/shell/integrations` endpoint
- Returns project names available in the registry
- Supports keyword filtering (case-insensitive substring match)
- Read-only operation — does not download or modify local state

**Output format:**
- Default: One project name per line
- JSON: Array of project name strings

**Error messages:**
- `"search integrations: %w"` — Failed to search registry
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

For **methods**:
- Searches `methods.json` files in `.swytchcode/integrations/`
- Reads wrekenfile to get full method details
- Adds method entry to `tooling.json` with `type: "method"`, `integration`, `summary`, `desc`, and `inputs`
- Automatically adds integration to `integrations` section if not already present

For **workflows**:
- Searches `workflows.json` files in `.swytchcode/integrations/`
- Reads workflow definition (with `name`, `canonical_id`, and `steps` array)
- For each step in the workflow:
  - Looks up the method's canonical_id in the wrekenfile `METHODS` section
  - Adds the method to `tooling.json` with full details (`summary`, `desc`, `inputs`, etc.)
  - Adds an `index` field to preserve the order of methods in the workflow
  - Skips methods that already exist in `tooling.json` (duplicates)
- Adds workflow entry to `tooling.json` with `type: "workflow"`, `name`, `integration`, and `steps` array (array of method canonical_ids in order)
- Automatically adds integration to `integrations` section if not already present

**Error messages:**
- `"canonical ID %q not found in any fetched integrations.\nRun: swytchcode get <project>"` — Tool not found in any integration
- `"ambiguous canonical ID. Found in %d integrations:\n  %s\nUse: swytchcode add <integration@version> %s"` — Tool found in multiple integrations (non-interactive mode)
- `"invalid integration spec format: %q (expected: project@library.version)"` — Invalid integration spec format
- `"Integration %s not installed. Run: swytchcode get %s"` — Integration not fetched yet
- `"method %q not found in wrekenfile"` — Method not found in wrekenfile
- `"workflow %q not found in workflows.json"` — Workflow not found in workflows.json
- `"Warning: method %q from workflow step not found in wrekenfile: %v"` — Method referenced in workflow steps not found in wrekenfile

---

### `swytchcode info <canonical_id>`

Show information about a tool (method or workflow) by canonical ID. Recursively searches all fetched integrations and displays tool details from wrekenfiles.

**Usage:**
```bash
# Human-readable output
swytchcode info api.cluster.create

# JSON output
swytchcode info api.cluster.create --json
```

**What it does:**
- Recursively searches `.swytchcode/integrations/{project}/{library}/{version}/` directories
- Checks `methods.json` and `workflows.json` files for the canonical_id
- Reads the corresponding `wrekenfile.yaml` to extract tool details
- Returns all matches if the canonical_id appears in multiple integrations
- Displays: canonical_id, type, integration, summary, description, inputs, and full wrekenfile entry

**Output format:**
- Default: Human-readable format with tool details
- `--json`: JSON array of tool information objects

**Error messages:**
- `"canonical ID %q not found in any fetched integrations"` — Tool not found in any integration
- `"Warning: failed to read wrekenfile for %s.%s@%s: %v"` — Failed to read wrekenfile for a specific integration

---

### `swytchcode exec [canonical_id]`

Execute a tool via the Swytchcode kernel. The **only** execution path for tools. Pure, deterministic, non-interactive, and offline-capable.

**Usage:**
```bash
# CLI args mode
swytchcode exec api.cluster.create --body cluster.json --input Authorization="Bearer token123"

# With query params
swytchcode exec api.cluster.get --param id=cluster-123 --input Authorization="Bearer token123"

# With custom headers
swytchcode exec api.cluster.get --header X-Request-Id=abc-123 --param id=cluster-123

# JSON stdin mode
echo '{"tool":"api.cluster.create","args":{"body":{"name":"my-cluster"},"Authorization":"Bearer token123"}}' | swytchcode exec

# Dry-run (show what would be executed)
swytchcode exec api.cluster.create --body cluster.json --dry-run

# Raw output mode
swytchcode exec api.cluster.get --param id=123 --raw

# JSON output (single JSON object to stdout, for scripting)
swytchcode exec api.cluster.get --param id=123 --json
```

**Flags:**
- `--allow-raw`: Required for executing raw methods (disabled by default)
- `--dry-run`: Show what would be executed without making HTTP call
- `--body <file>`: Path to JSON file containing request body
- `--input <key=value>`: Input key=value pairs (can be specified multiple times)
- `--param <key=value>`: Query parameter key=value pairs (can be specified multiple times)
- `--header <key=value>`: Request header key=value pairs (can be specified multiple times)
- `--raw`: Output raw HTTP response instead of normalized JSON
- `--json`: Output response as a single JSON object to stdout (for piping and scripting)

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
- **Default** / **--json**: Single JSON object to stdout — normalized shape with `request`, `status_code`, and `data` (or raw shape with `--raw`). Use `--json` explicitly when piping.
- **--raw**: Raw HTTP response JSON: `request`, `status_code`, `status`, `headers`, `body` (string)
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

Start the Model Context Protocol (MCP) server. Exposes swytchcode commands (`list`, `get`, `add`, `info`, `exec`) as MCP tools for agent communication.

**Usage:**
```bash
# Interactive mode (stdio transport, shows output)
swytchcode mcp serve

# Daemon mode (background, no output, returns control immediately)
swytchcode mcp serve -d

# Daemon mode with log file
swytchcode mcp serve -d --log-file /path/to/mcp.log

# HTTP transport
swytchcode mcp serve --transport http --port 3000

# HTTP transport in daemon mode
swytchcode mcp serve --transport http --port 3000 -d
```

### `swytchcode mcp status`

Check if the MCP server is running (daemon mode only).

**Usage:**
```bash
swytchcode mcp status
```

**What it does:**
- Checks for PID file at `.swytchcode/mcp.pid`
- Verifies if the process is still running
- Removes stale PID files if process is not running
- Prints server status (running with PID, or not running)

**Output:**
- `"MCP server is running (PID: <pid>)"` — Server is running
- `"MCP server is not running"` — No PID file found
- `"MCP server is not running (stale PID file removed)"` — PID file existed but process was not running

### `swytchcode mcp stop`

Stop the running MCP server (daemon mode only).

**Usage:**
```bash
swytchcode mcp stop
```

**What it does:**
- Reads PID file at `.swytchcode/mcp.pid`
- Sends SIGTERM signal to stop the server gracefully
- Removes PID file after stopping
- Only works for servers started in daemon mode

**Error messages:**
- `"MCP server is not running: %w"` — No PID file found or server not running
- `"MCP server is not running (stale PID file removed)"` — PID file existed but process was not running
- `"stop MCP server: %w"` — Failed to send stop signal

**Flags:**
- `-d`: Run in daemon mode (properly daemonized background process). The server runs in a new session, completely detached from the terminal. Returns control immediately. Creates PID file at `.swytchcode/mcp.pid` for `status` and `stop` commands. Requires `--transport http`.
- `--log-file <path>`: Path to log file (only used in daemon mode; if not provided, logs are suppressed)
- `--transport <type>`: Transport type (`stdio` or `http`), default: `stdio`
- `--port <number>`: Port for HTTP transport, default: `3000`

**What it does:**
- Starts an MCP server exposing nine tools:
  - `swytchcode_init` — Initialize Swytchcode in the project
  - `swytchcode_bootstrap` — Fetch all integrations declared in tooling.json
  - `swytchcode_version` — Get Swytchcode version
  - `swytchcode_list` — List locally available tools and integrations (no registry calls)
  - `swytchcode_search` — Search remote registry for available integrations
  - `swytchcode_get` — Fetch integration bundles
  - `swytchcode_add` — Add tools to tooling.json
  - `swytchcode_info` — Get information about a tool by canonical ID
  - `swytchcode_exec` — Execute tools
- All tool output is captured and returned through the MCP protocol (not streamed to terminal)
- In daemon mode (`-d`):
  - Forks a new process that runs independently in the background
  - **Unix/Linux/macOS**: Creates a new session (detached from terminal) so it survives parent process termination
  - **Windows**: Runs as independent process that survives parent termination
  - Returns control to terminal immediately
  - Process continues running even if terminal is closed or parent process exits
  - Logs to file if `--log-file` provided, otherwise output is discarded
  - Requires `--transport http` (stdio transport cannot be used in daemon mode)
  - **Cross-platform**: Works on Windows, Linux, and macOS
- Supports stdio transport (default) for direct process communication (non-daemon mode only)
- Supports HTTP transport with bearer token authentication

**MCP Tools:**

**swytchcode_init**
- Parameters: `editor` (string, required) — Editor choice: `"cursor"`, `"copilot"`, `"claude"`, or `"none"`; `mode` (string, required) — Execution mode: `"production"` or `"sandbox"`
- Returns: CLI output as-is. Initializes Swytchcode in the project (creates `.swytchcode/`, `tooling.json`, and editor-specific config).

**swytchcode_bootstrap**
- Parameters: None
- Returns: CLI output as-is. Fetches all integrations declared in `tooling.json` that are not already installed.

**swytchcode_version**
- Parameters: None
- Returns: Version string (e.g., `"swytchcode version 0.0.1\n"`).

**swytchcode_list**
- Parameters: `filter` (string, optional) — Filter type: `"methods"`, `"workflows"`, `"integrations"`, or empty for all; `prefix` (string, optional) — Project prefix filter (e.g., `"stripe"`); `json` (boolean, optional) — Output as JSON object
- Returns: Locally available tools and integrations (from tooling.json and fetched integrations). JSON format: `{"methods": [...], "workflows": [...], "integrations": [...]}`

**swytchcode_search**
- Parameters: `filter` (string, optional) — Filter type: `"integrations"` or `"methods"`; `keyword` (string, optional) — Search keyword; `json` (boolean, optional) — Output as JSON array
- Returns: Remote registry search results (project names). Read-only, never mutates local state.

**swytchcode_get**
- Parameters: `project_name` (string, required), `yes` (boolean, optional) — Auto-confirm overwrite
- Returns: CLI output as-is

**swytchcode_add**
- Parameters: `canonical_id` (string, required), `integration_spec` (string, optional) — Integration spec (project@library.version)
- Returns: CLI output as-is

**swytchcode_info**
- Parameters: `canonical_id` (string, required), `json` (boolean, optional) — Output as JSON array
- Returns: Tool information (human-readable or JSON array)

**swytchcode_exec**
- Parameters:
  - `tool` (string, required) — Canonical ID of tool to execute
  - `args` (object, optional) — Tool arguments: `body`, `params` (query/path), `Authorization`, `headers` (map of header name to value), and any other top-level keys as query params
  - `dry_run` (boolean, optional) — If true, show the planned request (method, url, headers, body) without making the HTTP call; use this to inspect the input that would be sent
  - `raw` (boolean, optional) — Output raw HTTP response
  - `allow_raw` (boolean, optional) — Allow execution of raw methods
  - `json` (boolean, optional) — Output response as a single JSON object
- Returns: The full stdout/stderr output in the tool result content (dry-run payload, execution result, or kernel JSON error on failure). On failure the result is still returned with `isError: true` so the client can see the error output.

**Error messages:**
- `"create MCP server: %w"` — Failed to create server
- `"open log file: %w"` — Failed to open log file
- `"register tools: %w"` — Failed to register MCP tools
- `"write PID file: %w"` — Failed to write PID file in daemon mode
- HTTP transport errors: `"missing Authorization header"`, `"invalid authorization token"`, `"method not allowed"`

**Note:** In daemon mode (`-d`):
- A PID file is created at `.swytchcode/mcp.pid` to track the running server process
- **Unix/Linux/macOS**: The server process runs in a new session, completely detached from the terminal
- **Windows**: The server process runs independently, detached from the parent terminal
- The process will continue running even if:
  - The parent process exits
  - The terminal session closes
  - You log out of the shell (Unix/Linux/macOS)
- **Cross-platform support**: Daemon mode works on Windows, Linux, and macOS
- Use `swytchcode mcp status` to check if the server is running
- Use `swytchcode mcp stop` to gracefully stop the server (SIGTERM on Unix, process termination on Windows)
- The server can only be stopped via `swytchcode mcp stop` or by killing the process directly

---

## Project Structure

After `swytchcode init`, the following structure is created:

```
.swytchcode/
├── tooling.json              # Project configuration (integrations, tools, mode, version)
└── integrations/
    ├── manifest.json         # Registry manifest with project.library entries (version, endpoints)
    └── {project}/{library}/{version}/
        ├── wrekenfile.yaml   # Wrekenfile spec with METHODS section
        ├── methods.json      # Methods list for this integration version
        └── workflows.json    # Workflows list (name, canonical_id, steps array with name and canonical_id)
```

### `tooling.json` structure

```json
{
  "version": "1.0",
  "mode": "production",
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
    },
    "workflow.example": {
      "name": "Example Workflow",
      "integration": "weaviate.lyrid@v1",
      "type": "workflow",
      "steps": ["api.cluster.create", "api.cluster.update"]
    },
    "api.cluster.update": {
      "summary": "Update a cluster instance",
      "integration": "weaviate.lyrid@v1",
      "type": "method",
      "desc": "Update a cluster instance",
      "inputs": [...],
      "index": 1
    }
  }
}
```

- **mode**: Execution mode (`production` or `sandbox`) — determines which endpoint from `manifest.json` is used
- **integrations**: Pinned integration versions (keys are `project.library` format)
- **tools**: Unified map of all trusted tools, keyed by `canonical_id`
  - **Methods**: `type: "method"` with `summary`, `desc`, `inputs`, `integration`. Methods added from workflows include an `index` field to preserve execution order.
  - **Workflows**: `type: "workflow"` with `name`, `integration`, and `steps` array (ordered list of method canonical_ids). When a workflow is added, all its step methods are automatically added to `tools` with full details.

---

## Editor rules (init)

When you run `swytchcode init --editor=<cursor|copilot|claude>`, the CLI installs rule templates so the editor uses Swytchcode for API execution and does not read `.swytchcode/` or Wrekenfiles directly.

| Editor | Files installed |
|--------|------------------|
| **cursor** | `.cursor/rules/swytchcode.mdc` |
| **copilot** | `.github/instructions/swytchcode.md` |
| **claude** | `CLAUDE.md` (repo root) |

Templates are embedded in the binary; source lives in `editors/` (see `editors/README.md`). Rules require using MCP tools `swytchcode_init`, `swytchcode_bootstrap`, `swytchcode_version`, `swytchcode_list`, `swytchcode_search`, `swytchcode_get`, `swytchcode_add`, `swytchcode_info`, `swytchcode_exec` and generating runtime code that calls `swytchcode exec <canonical_id>`.

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

---

## Testing with MCP Inspector

You can run the MCP server and inspect it with [MCP Inspector](https://modelcontextprotocol.io/docs/tools/inspector) (e.g. `npx @modelcontextprotocol/inspector`).

Using the shell

```sh
npx @modelcontextprotocol/inspector ./swytchcode mcp serve
```

Using the http

```sh
npx @modelcontextprotocol/inspector ./swytchcode mcp serve --transport http --port 3000
```