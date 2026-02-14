# Swytchcode Kernel

Swytchcode is the **execution kernel** for tools. Editors, agents, and languages call `swytchcode exec` to execute tools deterministically.

**`tooling.json` defines what is trusted.**  
**Wrekenfiles define what is possible.**

## Commands at a glance

| Command | Purpose |
|--------|---------|
| `swytchcode init` | Create `.swytchcode/`, `tooling.json`, and editor rule files (Cursor / Claude / VS Code) |
| `swytchcode get <project>` | Fetch integration bundles (Wrekenfiles, methods, workflows) |
| `swytchcode bootstrap` | Fetch all integrations declared in `tooling.json` |
| `swytchcode list` | List available integrations from the registry |
| `swytchcode add [spec] <canonical_id>` | Add a tool to `tooling.json` |
| `swytchcode exec [canonical_id]` | Execute a tool (CLI or JSON stdin); supports `--json`, `--raw`, `--dry-run` |
| `swytchcode mcp serve` | Start MCP server (stdio or HTTP); exposes `swytchcode_list`, `swytchcode_get`, `swytchcode_add`, `swytchcode_exec` |
| `swytchcode mcp status` | Check if MCP server is running (daemon mode) |
| `swytchcode mcp stop` | Stop MCP server (daemon mode) |

---

## Commands (detail)

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
- Installs editor rule templates in the repo (if editor ≠ `none`):
  - **cursor** → `.cursor/rules/swytchcode.mdc`
  - **claude** → `CLAUDE.md` (repo root)
  - **vscode** → `.github/instructions/swytchcode.md` (Copilot) and `CLAUDE.md` (repo root)

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

# JSON output (single JSON object to stdout, for scripting)
swytchcode exec api.cluster.get --param id=123 --json
```

**Flags:**
- `--allow-raw`: Required for executing raw methods (disabled by default)
- `--dry-run`: Show what would be executed without making HTTP call
- `--body <file>`: Path to JSON file containing request body
- `--input <key=value>`: Input key=value pairs (can be specified multiple times)
- `--param <key=value>`: Query parameter key=value pairs (can be specified multiple times)
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

Start the Model Context Protocol (MCP) server. Exposes swytchcode commands (`list`, `get`, `add`, `exec`) as MCP tools for agent communication.

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
- Starts an MCP server exposing four tools:
  - `swytchcode_list` — List available integrations
  - `swytchcode_get` — Fetch integration bundles
  - `swytchcode_add` — Add tools to tooling.json
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
  - `json` (boolean, optional) — Output response as a single JSON object
- Returns: CLI output as-is (matches `swytchcode exec` output format)

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

## Editor rules (init)

When you run `swytchcode init --editor=<cursor|claude|vscode>`, the CLI installs rule templates so the editor uses Swytchcode for API execution and does not read `.swytchcode/` or Wrekenfiles directly.

| Editor | Files installed |
|--------|------------------|
| **cursor** | `.cursor/rules/swytchcode.mdc` |
| **claude** | `CLAUDE.md` (repo root) |
| **vscode** | `.github/instructions/swytchcode.md` (Copilot), `CLAUDE.md` (repo root) |

Templates are embedded in the binary; source lives in `editors/` (see `editors/README.md`). Rules require using MCP tools `swytchcode_list`, `swytchcode_get`, `swytchcode_add`, `swytchcode_exec` and generating runtime code that calls `swytchcode exec <canonical_id>`.

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