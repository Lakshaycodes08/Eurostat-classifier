# Swytchcode CLI

Swytchcode is the **execution kernel** for tools. Editors, agents, and languages call `swytchcode exec` to execute tools deterministically.

**`tooling.json` defines what is trusted.**  
**Wrekenfiles define what is possible.**

### Docs

- [Architecture](docs/architecture.md) – modules and data flow (CLI, kernel, registry, MCP, editors).
- [Execution model](docs/execution-model.md) – how `swytchcode exec` works end-to-end.
- [Config spec](docs/config-spec.md) – `tooling.json` and `manifest.json`.
- [CLI reference](docs/cli-reference.md) – commands, inputs/outputs, exit codes.
- [MCP & integrations](docs/mcp-and-integrations.md) – MCP server and editor integrations.
- [Install & upgrade](docs/install-upgrade.md) – install scripts (including Windows) and upgrade behavior.
- [Security](docs/security.md) – trust boundaries, HTTPS, `SWYTCHCODE_INSECURE`.
- [Windows guide](docs/windows-guide.md) – JSON, `cmd.exe`, and `--body`.

### Install

**macOS / Linux:**
```bash
curl -fsSL https://cli.swytchcode.com/install.sh | sh
```
Optional: `VERSION=v1.0.8` for a specific release, `INSTALL_DIR=/path` to choose install location (default: `/usr/local/bin` if writable, else `~/.local/bin`).

**Windows (PowerShell):**
```powershell
irm https://cli.swytchcode.com/install.ps1 | iex
```
Optional: `$env:VERSION="v1.0.8"` for a specific release, `$env:INSTALL_DIR="C:\path"` to choose install location (default: `%LOCALAPPDATA%\Programs\swytchcode\bin`, added to user PATH automatically).

**More:** [GitLab Pages](https://cli.swytchcode.com/) · [swytchcode.com](https://swytchcode.com)

---

## Quick start (2 minutes)

This is the fastest end-to-end flow:

```bash
# 1) Initialize Swytchcode in this repo
swytchcode init --editor=cursor --mode=sandbox

# 2) Fetch an integration (one-time per integration/version)
swytchcode get ngage

# 3) Add a tool (declare intent / trust boundary)
swytchcode add customers.external_account.create

# 4) Execute the tool (replace placeholders with real values)
echo '{
  "tool": "customers.external_account.create",
  "args": {
    "customerID": "your-customer-id",
    "Idempotency-Key": "your-idempotency-key",
    "body": {
      "variant": {
        "value": {
          "accountNumber": "000000000000",
          "accountType": "checking",
          "bankName": "Example Bank",
          "routingNumber": "000000000"
        }
      }
    }
  }
}' | swytchcode exec --json
```

**Mental model:**

- **`init`**: create `.swytchcode/` and `tooling.json` in your project
- **`get`**: download integrations (what’s possible)
- **`add`**: enable specific trusted tools in `tooling.json` (what’s allowed)
- **`exec`**: execute deterministically via the kernel

## Commands at a glance

**Typical flow:** `init` → `get` → `add` → `exec`

| Command | Purpose |
|--------|---------|
| `swytchcode -v` or `swytchcode --version` | Show Swytchcode version |
| `swytchcode init` | Create `.swytchcode/`, `tooling.json`, and editor rule files (Cursor / Claude) |
| `swytchcode get <project>` | Fetch integration bundles (Wrekenfiles, methods, workflows) |
| `swytchcode bootstrap` | Fetch all integrations declared in `tooling.json` |
| `swytchcode list` | List locally available tools and integrations (from tooling.json and fetched integrations) |
| `swytchcode list methods [pattern]` | List all methods from .swytchcode/integrations (optional pattern filters by canonical_id or project) |
| `swytchcode list workflows [pattern]` | List all workflows from .swytchcode/integrations (optional pattern filters by canonical_id or project) |
| `swytchcode list integrations` | List locally fetched integrations |
| `swytchcode search [keyword]` | Search remote registry; no keyword = all integrations, with keyword = matching names |
| `swytchcode sync [project_name]` | Re-fetch workflow/method list from backend; updates local files without modifying tooling.json |
| `swytchcode doctor` | Local diagnostics: `tooling.json`, bundles, `manifest.json`, HTTPS base URLs, auth; `--json` for machines; exits 1 on errors |
| `swytchcode add [spec] <canonical_id>` | Add a tool to `tooling.json`; auto-downloads missing library deps for multi-library workflows |
| `swytchcode info <canonical_id>` | Show information about a tool by canonical ID |
| `swytchcode exec [canonical_id]` | Execute a tool (CLI or JSON stdin); supports `--json`, `--raw`, `--dry-run` |
| `swytchcode mcp serve` | Start MCP server (stdio or HTTP); exposes `swytchcode_init`, `swytchcode_bootstrap`, `swytchcode_version`, `swytchcode_list`, `swytchcode_search`, `swytchcode_get`, `swytchcode_add`, `swytchcode_info`, `swytchcode_exec`, `swytchcode_check`, `swytchcode_inspect`, `swytchcode_upgrade`, `swytchcode_diff`, `swytchcode_discover`, `swytchcode_plan`, `swytchcode_doctor` |
| `swytchcode mcp status` | Check if MCP server is running (daemon mode) |
| `swytchcode mcp stop` | Stop MCP server (daemon mode) |
| `swytchcode login` | Device-flow browser login; saves session to `~/.swytchcode/auth.json` |
| `swytchcode logout` | Delete saved session |
| `swytchcode whoami` | Show current auth state: user session (email, UUID, expiry) or service token (masked); accepts `SWYTCHCODE_TOKEN` |
| `swytchcode check [project_or_library]` | Check for integration update proposals; exits 1 on breaking changes; requires `SWYTCHCODE_TOKEN` or login |
| `swytchcode inspect <library>` | Show full proposal detail for a library (requires login) |
| `swytchcode upgrade <library> [--apply]` | Approve a pending integration update proposal (requires login); `--apply` also re-downloads and refreshes tools |
| `swytchcode diff <library>` | Show method-level changes in a pending upgrade proposal (requires login or `SWYTCHCODE_TOKEN`) |

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
swytchcode version 1.0.2
```

The version is defined in `internal/constants/constants.go` and is overridden in release builds by Goreleaser via `-ldflags -X` so `swytchcode --version` matches the Git tag. To change the default version for local builds, update `constants.Version` and rebuild.

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
- `--editor`: Editor choice (`cursor | claude | none`)
- `--mode`: Execution mode (`production | sandbox`)
- `--non-interactive`: Disable prompts (required for CI)

**What it does:**
- Creates `.swytchcode/` and `.swytchcode/integrations/` directories
- Creates `tooling.json` with empty `integrations` and `tools` maps
- Sets `mode` and `version` in `tooling.json`
- Installs editor rule templates in the repo (if editor ≠ `none`):
  - **cursor** → `.cursor/rules/swytchcode.mdc` and updates `~/.cursor/mcp.json` with SSE URL entry
  - **claude** → `CLAUDE.md` (repo root) and updates `~/.claude/settings.json` with SSE URL entry
- Starts the MCP HTTP daemon in the background (`swytchcode mcp serve --daemon --transport http`) so editor tools are available **immediately** — no restart required

> **No restart needed:** Because init registers an SSE URL (`http://localhost:5476/sse`) rather than a stdio process, the editor can connect to the already-running daemon at any time. Tools are live as soon as init finishes.

**Error messages:**
- `"init requires --editor when running non-interactively"` — Missing `--editor` flag in non-interactive mode
- `"init requires --mode when running non-interactively"` — Missing `--mode` flag in non-interactive mode
- `"invalid mode %q (expected production or sandbox)"` — Invalid mode value
- `"unknown editor %q (expected cursor|claude|none)"` — Invalid editor value

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
- Uses `sandbox_endpoint` and `production_endpoint` directly from bundle response.
  - If an endpoint is missing for the active `mode`, the CLI falls back to `http://localhost` for that integration **and expects you to be running the corresponding backend locally**. This is unrelated to the MCP HTTP server address.

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

# List all methods (scans .swytchcode/integrations recursively)
swytchcode list methods

# List all workflows
swytchcode list workflows

# List only integrations
swytchcode list integrations

# Filter by pattern (matches canonical_id or project name, case-insensitive)
swytchcode list methods stripe
swytchcode list methods app.invite
swytchcode list workflows swytchcode

# JSON output
swytchcode list --json
swytchcode list methods --json
```

**Flags:**
- `--json`: Output as JSON object with `methods`, `workflows`, and `integrations` arrays

**What it does:**
- Scans `.swytchcode/integrations/` recursively: reads `methods.json` and `workflows.json` in each integration to list all available methods and workflows
- Lists integrations as `project.library@version` from the directory structure
- Optional **pattern** (second argument): filters results by canonical_id or project name (case-insensitive substring match)

**Output format:**

Default (human-readable): methods and workflows show `canonical_id` and `project.library@version` for easier identification.
```
Methods:
  stripe.customer.create  stripe.stripe@v1
  stripe.customer.update  stripe.stripe@v1

Workflows:
  stripe.checkout.session.create  stripe.stripe@v1

Integrations:
  stripe.stripe@v1
  weaviate.lyrid@v1
```

JSON (`--json`): methods and workflows are arrays of `{ "canonical_id": "...", "integration": "project.library@version" }`.
```json
{
  "methods": [
    { "canonical_id": "stripe.customer.create", "integration": "stripe.stripe@v1" },
    { "canonical_id": "stripe.customer.update", "integration": "stripe.stripe@v1" }
  ],
  "workflows": [
    { "canonical_id": "stripe.checkout.session.create", "integration": "stripe.stripe@v1" }
  ],
  "integrations": ["stripe.stripe@v1", "weaviate.lyrid@v1"]
}
```

**Error messages:**
- `"detect project root: %w"` — Failed to detect project root
- `"integrations directory not found at ... run 'swytchcode get <project>' first"` — No integrations fetched yet (when listing methods or workflows)

---

### `swytchcode search`

Search remote registry for available integrations. **Read-only** — never mutates local state.

**Usage:**
```bash
# List all available integrations
swytchcode search

# Search by keyword (case-insensitive substring match)
swytchcode search stripe
swytchcode search weaviate

# JSON output
swytchcode search --json
swytchcode search stripe --json
```

**Flags:**
- `--json`: Output as JSON array instead of one project name per line

**What it does:**
- Calls `GET /v2/cli/integrations` endpoint
- With no keyword: returns all project names from the registry
- With keyword: returns project names that contain the keyword (case-insensitive)
- Read-only operation — does not download or modify local state

**Output format:**
- Default: One project name per line
- JSON: Array of project name strings

**Error messages:**
- `"search: %w"` — Failed to search registry
- `"encode JSON: %w"` — Failed to encode JSON output

---

### `swytchcode sync [project_name]`

Pull the latest workflow and method list from the backend for installed projects and update local files. Does **not** modify `tooling.json` — the user still decides which tools to activate with `swytchcode add`.

**Usage:**
```bash
# Sync all installed projects
swytchcode sync

# Sync a single project
swytchcode sync stripe
```

**What it does:**
- For each installed project (or just the named one): calls the backend for the current workflow list
- Compares against the local `workflows.json`; identifies new and updated workflows
- If any changes found: re-fetches the full integration bundle (wrekenfiles + manifest), overwrites local files
- Prints which new workflows are now available; warns if an updated workflow is already in `tooling.json` (run `swytchcode add <canonical_id>` to refresh it in tooling)
- Does **not** touch `tooling.json`

**Output example:**
```
Syncing project: stripe
  ✓ 2 new workflow(s) available: stripe.charge_and_notify, stripe.refund_flow
  ⚠ stripe.checkout updated (already in tooling.json — run: swytchcode add stripe.checkout to refresh)

Syncing project: weaviate
  ✓ Already up to date
```

**Error messages:**
- `"no integrations found — run: swytchcode get <project>"` — No projects installed yet
- `"fetch workflows from backend: %w"` — Network or auth error when calling the backend

---

### `swytchcode doctor`

Run a **read-only** checklist for the current project: `tooling.json` parses, declared integration bundles exist and Wrekenfiles parse, `manifest.json` parses, non-empty integration base URLs satisfy HTTPS/loopback rules, auth posture (`SWYTCHCODE_TOKEN` or user session), and `SWYTCHCODE_INSECURE` / CI interaction.

**Usage:**
```bash
swytchcode doctor
swytchcode doctor --json
```

**Exit codes:** `0` if no **error**-level checks failed; `1` otherwise. **Warnings** (e.g. no auth — fine for offline `exec`) do not fail the command.

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
- **Multi-library dep resolution**: if the workflow has steps whose methods live in libraries not yet downloaded, the CLI auto-fetches all required bundles in a single request (`GET /v2/cli/integrations/{project}/workflow/{canonical_id}/bundles`) and saves them before proceeding. Use `--no-auto-install` to skip this and exit with an error instead.
- Adds a **single** workflow entry to `tooling.json` with `type: "workflow"`, `name`, `integration`, and `steps` as an **array of step definitions** (not top-level method entries). Each step object includes `canonical_id`, `name`, `summary`, `desc`, `inputs`, `integration`, and `index` so the workflow is self-contained.
- For multi-library workflows, each step’s `integration` field reflects its actual library (e.g. `stripe.stripe@v2.1`), not the workflow’s own library.
- Step methods are **not** added as separate top-level tools; they exist only inside the workflow’s `steps` array.
- Automatically adds integration to `integrations` section if not already present

**Flags:**
- `--no-auto-install` — Skip auto-downloading missing library deps for multi-library workflows; exits with error instead. Use in CI for explicit dependency control.
- `--all` — Add all methods and workflows for a project in one command (usage: `swytchcode add --all <project>`). Already-present tools are skipped. Prints a summary: `Added N, skipped M already present, K failed`.

**Error messages:**
- `"canonical ID %q not found in any fetched integrations.\nRun: swytchcode get <project>"` — Tool not found in any integration
- `"ambiguous canonical ID. Found in %d integrations:\n  %s\nUse: swytchcode add <integration@version> %s"` — Tool found in multiple integrations (non-interactive mode)
- `"invalid integration spec format: %q (expected: project@library.version)"` — Invalid integration spec format
- `"Integration %s not installed. Run: swytchcode get %s"` — Integration not fetched yet
- `"method %q not found in wrekenfile"` — Method not found in wrekenfile
- `"workflow %q not found in workflows.json"` — Workflow not found in workflows.json
- `"workflow has steps requiring libraries not yet downloaded..."` — `--no-auto-install` set but deps missing
- `"Warning: method %q from workflow step not found in wrekenfile: %v"` — Method referenced in workflow steps not found in wrekenfile (that step is skipped; workflow is still added)

---

### `swytchcode info <canonical_id>`

Show information about a tool (method or workflow) by canonical ID. Recursively searches all fetched integrations and displays tool details: **methods** from each integration’s wrekenfile, **workflows** from each integration’s `workflows.json` (no wrekenfile WORKFLOWS section required).

**Usage:**
```bash
# Human-readable output
swytchcode info api.cluster.create

# JSON output
swytchcode info api.cluster.create --json
```

**What it does:**
- Recursively searches `.swytchcode/integrations/{project}/{library}/{version}/` directories
- Looks up the canonical_id in `methods.json` and `workflows.json` under each integration
- For **methods**: reads `wrekenfile.yaml` (METHODS section) for summary, description, and inputs
- For **workflows**: reads `workflows.json` for summary, description, and steps (no wrekenfile required)
- Returns all matches if the canonical_id appears in multiple integrations
- Displays: canonical_id, type, integration, summary, description, inputs (or steps for workflows), and full source entry

**Output format:**
- Default: Human-readable format with tool details
- `--json`: JSON array of tool information objects

**Error messages:**
- `"canonical ID %q not found in any fetched integrations"` — Tool not found in any integration
- `"Warning: failed to read tool data for %s.%s@%s: %v"` — Failed to read wrekenfile or workflows.json for a specific integration

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

**Example output (single method):**
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

**Checking for API errors:** The CLI always exits `0` when it received an HTTP response, even for 4xx/5xx. Check `status_code` in the output:
```js
const result = JSON.parse(stdout);
if (result.status_code >= 400) {
  // result.data contains the API error body
  throw new Error(`API error ${result.status_code}: ${JSON.stringify(result.data)}`);
}
```

**Workflow output:**
```json
{
  "success": false,
  "error": "workflow failed at step 2 (create-order): HTTP 400: Bad Request",
  "steps": [
    { "step": 1, "name": "create-session", "status_code": 200, "data": {...} },
    { "step": 2, "name": "create-order", "status_code": 400, "data": {...}, "failed": true }
  ]
}
```
Check `result.success` and find the failed step via `result.steps.find(s => s.failed)`.

**Exit codes:**
- `0` — Success **or API-level error** (4xx/5xx received from target API — check `status_code` in output)
- `1` — Invalid input (bad JSON, missing tool, validation failure)
- `2` — Tool not found in `tooling.json` or bundle
- `3` — Auth missing/invalid (reserved)
- `4` — Network/transport failure (DNS, timeout, connection refused — no response received)
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

# HTTP transport (SSE — fully implemented)
swytchcode mcp serve --transport http --port 5476

# HTTP transport in daemon mode
swytchcode mcp serve --transport http --port 5476 -d
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

---

### Cloud commands (login / logout / whoami / check / inspect / upgrade)

Auth requirements vary by command:

| Command | Auth required | Accepts `SWYTCHCODE_TOKEN` |
|---------|--------------|---------------------------|
| `login` | No (this is how you authenticate) | — |
| `logout` | No | — |
| `whoami` | Optional — prints "Not logged in" if no session or token | Yes |
| `check` | Optional — passes empty token if missing (server returns 401) | Yes |
| `inspect` | Yes — user session only | No |
| `upgrade` | Yes — user session only | No |

Cloud commands contact `SWYTCHCODE_API_URL` (default: `https://api-v2.swytchcode.com`).

**Telemetry:** Usage events are sent when authenticated — either via `swytchcode login` (user session) or `SWYTCHCODE_TOKEN` (service token). The backend identifies your account from the bearer token, so server-side exec calls are tracked the same way as interactive ones. With no auth, telemetry is skipped and a one-time hint may be shown. See `CLI_TELEMETRY.md` for the full contract. Telemetry is **independent of exec success** — HTTP 4xx/5xx responses from the target API (including 404s from `http://localhost`) are still real API responses; debug them via tool info, `--dry-run`, and your project’s `tooling.json` / `manifest.json`, not by changing telemetry settings.

**Where to set SWYTCHCODE_TOKEN:** The CLI reads the token only from the process environment (it does not load `.env` files). Quick reference:

- **Mac/Linux — current session:** `export SWYTCHCODE_TOKEN=your_token_here`
- **Mac/Linux — permanent (Zsh):** `echo 'export SWYTCHCODE_TOKEN=your_token_here' >> ~/.zshrc && source ~/.zshrc`
- **Mac/Linux — permanent (Bash):** `echo 'export SWYTCHCODE_TOKEN=your_token_here' >> ~/.bashrc && source ~/.bashrc`
- **Windows — permanent (PowerShell):** `[System.Environment]::SetEnvironmentVariable("SWYTCHCODE_TOKEN","your_token_here","User")`
- **Windows — permanent (cmd):** `setx SWYTCHCODE_TOKEN "your_token_here"` (open a new terminal after)
- **Node.js projects:** `node --env-file=.env src/index.js` (Node 20.6+) or add `require('dotenv').config()` in your entry point
- **MCP daemon:** Export the token before starting `swytchcode mcp serve` so the daemon inherits it.
- **CI/CD:** Define as a secret/CI variable.

See `docs/cli-reference.md` → "Setting SWYTCHCODE_TOKEN" for full detail.

**Calling the CLI from Node.js code (without the runtime package):**

The recommended way is the [swytchcode-runtime](https://www.npmjs.com/package/swytchcode-runtime) package, which handles all of this automatically. If you call the CLI directly, use `spawnSync`/`spawn` — **not** `child_process.exec`:

| Problem with `exec()` | Why it fails |
|----------------------|--------------|
| Default `maxBuffer` is 200KB | Large API responses are silently truncated |
| No easy stdin | CLI in JSON-stdin mode hangs waiting for input |
| stdout/stderr arrive as Buffer | `JSON.parse(buffer)` throws unless you call `.toString()` |
| Uses a subshell — PATH may differ | `swytchcode` not found even if installed |

Sync pattern:
```js
import { spawnSync } from 'node:child_process';

const result = spawnSync(
  'swytchcode',
  ['exec', '--json', 'books.book.get'],
  {
    input: JSON.stringify({ params: { id: '123' } }), // JSON args on stdin
    encoding: 'utf8',             // stdout/stderr as strings, not Buffers
    maxBuffer: 10 * 1024 * 1024, // 10MB — match runtime default
    env: process.env,             // SWYTCHCODE_TOKEN is inherited automatically
  }
);

if (result.error) throw result.error; // binary not found / permission denied
if (result.status !== 0) throw new Error(result.stderr || 'swytchcode failed');

const output = JSON.parse(result.stdout);
if (output.status_code >= 400) throw new Error(`API ${output.status_code}: ${JSON.stringify(output.data)}`);
```

Async pattern:
```js
import { spawn } from 'node:child_process';

function execTool(canonicalId, args = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn('swytchcode', ['exec', '--json', canonicalId], { env: process.env });
    let stdout = '', stderr = '';
    child.stdout.on('data', d => { stdout += d; });
    child.stderr.on('data', d => { stderr += d; });
    child.stdin.write(JSON.stringify(args));
    child.stdin.end();
    child.on('close', code => {
      if (code !== 0) return reject(new Error(stderr || `exit ${code}`));
      try { resolve(JSON.parse(stdout)); }
      catch { reject(new Error('invalid JSON: ' + stdout)); }
    });
  });
}
```

If `swytchcode` is not found, pass the absolute binary path (e.g. `/usr/local/bin/swytchcode`) or ensure the spawning process has the correct `PATH`.

#### `swytchcode login`

Opens a browser-based device-flow login. Saves the session to `~/.swytchcode/auth.json` on success.

```bash
swytchcode login
```

#### `swytchcode logout`

Deletes the saved session file (`~/.swytchcode/auth.json`).

```bash
swytchcode logout
```

#### `swytchcode whoami`

Prints the current session: email, customer UUID, and token expiry.

```bash
swytchcode whoami
```

If `SWYTCHCODE_TOKEN` is set, prints the masked token and auth source instead of the user session:

```
auth:    service token (SWYTCHCODE_TOKEN)
token:   sc12****5678
```

Prints "Not logged in." if neither a session nor a token is present.

#### `swytchcode check [project_or_library]`

Fetches integration update proposals for the authenticated customer and prints a summary. Both `SWYTCHCODE_TOKEN` (service token) and `swytchcode login` (user session) are fully supported — proposals are scoped to your account in both cases.

```bash
swytchcode check                    # all proposals for the authed user
swytchcode check weaviate           # filter by project name
swytchcode check weaviate.lyrid     # filter by project.library
```

**Flags:**
- `--json` — Output structured JSON (for programmatic use)

**Exit codes:**
- `0` — No proposals (or no breaking ones)
- `1` — Breaking-impact proposals found
- `2` — CLI/auth error

Requires `SWYTCHCODE_TOKEN` or `~/.swytchcode/auth.json`.

**Server-side usage (Node.js):**

```js
const { spawnSync } = require('node:child_process');

const r = spawnSync('swytchcode', ['check', '--json'], {
  encoding: 'utf8',
  env: process.env,  // inherits SWYTCHCODE_TOKEN
});
if (r.error) throw r.error;
const result = JSON.parse(r.stdout);
// result.up_to_date    — true if no proposals
// result.has_breaking  — true if any major-impact proposals
// result.proposals     — full proposal array
if (result.has_breaking) {
  const breaking = result.proposals.filter(p => p.impact === 'major');
  console.warn('Breaking integration updates available:', breaking);
}
```

#### `swytchcode inspect <library>`

Shows full proposal detail for the named library (two-step: looks up proposal UUID, then fetches detail).

```bash
swytchcode inspect stripe
swytchcode inspect stripe.stripe
```

Requires user login (`swytchcode login`).

#### `swytchcode upgrade <library>`

Approves a pending integration update proposal for the named library.

```bash
swytchcode upgrade stripe
swytchcode upgrade stripe.stripe
swytchcode upgrade stripe --apply   # also refreshes local bundle + tooling.json
```

**Flags:**
- `--apply` — After approval, run `get` and re-add all affected tools.

Requires user login (`swytchcode login`).

#### `swytchcode diff <library>`

Show method-level changes in a pending upgrade proposal before approving.

```bash
swytchcode diff stripe
swytchcode diff weaviate.lyrid
```

Prints `ADDED`, `REMOVED`, and `CHANGED` methods with field-level input diffs. Requires login or `SWYTCHCODE_TOKEN`.

See `commands.md` for full verification detail on these commands.

**Flags:**
- `-d`: Run in daemon mode (properly daemonized background process). The server runs in a new session, completely detached from the terminal. Returns control immediately. Creates PID file at `.swytchcode/mcp.pid` for `status` and `stop` commands. Requires `--transport http`.
- `--log-file <path>`: Path to log file (only used in daemon mode; if not provided, logs are suppressed)
- `--transport <type>`: Transport type (`stdio` or `http`), default: `stdio`
- `--port <number>`: Port for HTTP transport, default: `5476`

**What it does:**
- Starts an MCP server exposing fifteen tools:
  - `swytchcode_init` — Initialize Swytchcode in the project
  - `swytchcode_bootstrap` — Fetch all integrations declared in tooling.json
  - `swytchcode_version` — Get Swytchcode version
  - `swytchcode_list` — List locally available tools and integrations (no registry calls)
  - `swytchcode_search` — Search remote registry for available integrations
  - `swytchcode_get` — Fetch integration bundles
  - `swytchcode_add` — Add tools to tooling.json
  - `swytchcode_info` — Get information about a tool by canonical ID
  - `swytchcode_exec` — Execute tools
  - `swytchcode_check` — Check for integration update proposals
  - `swytchcode_inspect` — Show full proposal detail for a library
  - `swytchcode_upgrade` — Approve a pending integration update proposal
  - `swytchcode_diff` — Show method-level signature diff for a pending upgrade
  - `swytchcode_discover` — Semantic search for capabilities by natural-language intent
  - `swytchcode_plan` — Preview ordered steps of a workflow
  - `swytchcode_doctor` — Local project diagnostics (tooling, bundles, manifest, HTTPS URLs, auth)
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
- Supports HTTP/SSE transport: listens on `127.0.0.1:<port>/sse` (SSE event stream) and `/message` (JSON-RPC POST). The server binds to localhost only — not reachable externally. No auth token required for local use.

**MCP Tools:**

**swytchcode_init**
- Parameters: `editor` (string, required) — Editor choice: `"cursor"`, `"claude"`, or `"none"`; `mode` (string, required) — Execution mode: `"production"` or `"sandbox"`
- Returns: CLI output as-is. Initializes Swytchcode in the project (creates `.swytchcode/`, `tooling.json`, and editor-specific config).

**swytchcode_bootstrap**
- Parameters: None
- Returns: CLI output as-is. Fetches all integrations declared in `tooling.json` that are not already installed.

**swytchcode_version**
- Parameters: None
- Returns: Version string (e.g., `"swytchcode version 1.0.2\n"`).

**swytchcode_list**
- Parameters: `filter` (string, optional) — Filter type: `"methods"`, `"workflows"`, `"integrations"`, or empty for all; `prefix` (string, optional) — Pattern to filter by canonical_id or project name (case-insensitive); `json` (boolean, optional) — Output as JSON object
- Returns: Locally available tools and integrations. Methods and workflows include `canonical_id` and `integration` (project.library@version). JSON format: `{"methods": [{"canonical_id":"...","integration":"..."}], "workflows": [...], "integrations": ["..."]}`

**swytchcode_search**
- Parameters: `keyword` (string, optional) — Search keyword; if empty, returns all integrations; `json` (boolean, optional) — Output as JSON array
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
      "steps": [
        {
          "canonical_id": "api.cluster.create",
          "name": "Create cluster",
          "summary": "Create a new cluster instance",
          "desc": "Create a new cluster instance",
          "inputs": [...],
          "integration": "weaviate.lyrid@v1",
          "index": 0
        },
        {
          "canonical_id": "api.cluster.update",
          "name": "Update cluster",
          "summary": "Update a cluster instance",
          "desc": "Update a cluster instance",
          "inputs": [...],
          "integration": "weaviate.lyrid@v1",
          "index": 1
        }
      ]
    }
  }
}
```

- **mode**: Execution mode (`production` or `sandbox`) — determines which endpoint from `manifest.json` is used
- **integrations**: Pinned integration versions (keys are `project.library` format)
- **tools**: Unified map of all trusted tools, keyed by `canonical_id`
  - **Methods**: `type: "method"` with `summary`, `desc`, `inputs`, `integration`.
  - **Workflows**: `type: "workflow"` with `name`, `integration`, and `steps` array. Each element of `steps` is a step definition object (`canonical_id`, `name`, `summary`, `desc`, `inputs`, `integration`, `index`). Step methods are defined only inside the workflow’s `steps`; they are not separate top-level entries in `tools`.

---

## Editor rules (init)

When you run `swytchcode init --editor=<cursor|claude>`, the CLI installs rule templates so the editor uses Swytchcode for API execution and does not read `.swytchcode/` or Wrekenfiles directly.

| Editor | Files installed | Global config updated |
|--------|------------------|----------------------|
| **cursor** | `.cursor/rules/swytchcode.mdc` | `~/.cursor/mcp.json` (SSE URL entry) |
| **claude** | `CLAUDE.md` (repo root) | `~/.claude/settings.json` (SSE URL entry) |

Templates are embedded in the binary; source lives in `editors/` (see `editors/README.md`). Rules require using MCP tools `swytchcode_init`, `swytchcode_bootstrap`, `swytchcode_version`, `swytchcode_list`, `swytchcode_search`, `swytchcode_get`, `swytchcode_add`, `swytchcode_info`, `swytchcode_exec`, `swytchcode_check`, `swytchcode_inspect`, `swytchcode_upgrade`, `swytchcode_diff`, `swytchcode_discover`, `swytchcode_plan`, `swytchcode_doctor` and generating runtime code that calls `swytchcode exec <canonical_id>`.

---

## Base URL Resolution

When executing a tool, the base URL is resolved from `manifest.json` based on the `mode` in `tooling.json`:

- If `mode` is `"sandbox"` → use `sandbox_endpoint` from `manifest.json`
- Otherwise → use `production_endpoint` from `manifest.json`

The full URL is constructed as: `baseURL + endpoint` (where `endpoint` comes from the Wreken METHOD definition).

### HTTPS, localhost, CI, and Docker

Before sending a request, the CLI validates the integration base URL:

- **`https://`** is allowed for any host.
- **`http://`** is allowed **only** for loopback: `localhost`, `127.0.0.1`, or `::1` (any port). Other `http://` bases are rejected.

So remote or container service names (for example `http://api:8080` or `http://host.docker.internal`) must use **`https://`** unless you terminate TLS or proxy to something listening on loopback in the same namespace as `swytchcode`.

**GitHub Actions, GitLab CI, and similar:** Same rules as locally — use HTTPS for real APIs; use `http://127.0.0.1` / `http://localhost` only when your mock or API listens on loopback inside that job.

**`SWYTCHCODE_INSECURE=1`:** Skips TLS certificate verification (self-signed dev servers). Registry calls **refuse** this when `CI`, `GITHUB_ACTIONS`, or `GITLAB_CI` is set. It does **not** allow non-loopback `http://` execution URLs. See [docs/config-spec.md](docs/config-spec.md) (manifest section) for full detail.

---

## Error Handling

All errors are written to stderr as JSON:
```json
{"error": "error message here"}
```

Exit codes are stable and documented above. The kernel never prompts during execution and never calls the registry during `exec`.

### Debugging 404s from localhost

When a tool call returns a 404 with a URL starting with `http://localhost` (for example, a message like `Route PUT:/... not found` in the `data.error` field), the 404 is coming from the **target API** at that base URL, not from Swytchcode or the MCP server.

To debug:

1. Inspect the tool definition:
   - `swytchcode info <canonical_id> --json`
   - Confirm the HTTP method and endpoint path.
2. Inspect the resolved request without executing it:
   - `swytchcode exec <canonical_id> --dry-run --json`
   - Check the `method`, `url`, `headers`, and `body` fields.
3. Inspect your project configuration:
   - `.swytchcode/tooling.json` → confirm `mode` and `tools[canonical_id].integration`.
   - `.swytchcode/integrations/manifest.json` → confirm the integration’s `sandbox_endpoint` / `production_endpoint`.
4. Fix either:
   - Update the integration’s endpoints in `manifest.json` so they point at the correct API host (sandbox or production), **or**
   - Start/adjust the local backend so it actually implements the route at the configured `http://localhost` base URL.

MCP’s HTTP server (when enabled via `swytchcode mcp serve --transport http`) is only a transport for MCP clients and is **never** used as the base URL for tool execution.

---

## Testing with MCP Inspector

You can run the MCP server and inspect it with [MCP Inspector](https://modelcontextprotocol.io/docs/tools/inspector) (e.g. `npx @modelcontextprotocol/inspector`).

Using the shell

```sh
npx @modelcontextprotocol/inspector ./swytchcode mcp serve
```

Using the http

```sh
npx @modelcontextprotocol/inspector ./swytchcode mcp serve --transport http --port 5476
```