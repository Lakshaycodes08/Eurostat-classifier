# Swytchcode CLI – MCP & Integrations

Swytchcode exposes its functionality over the Model Context Protocol (MCP) so editors and agents can call it in a standard way. This document explains how MCP tools map to CLI commands and how editor integrations are wired.

## Authentication

The MCP server uses the same auth as the CLI: `SWYTCHCODE_TOKEN` or `~/.swytchcode/auth.json`. Set `SWYTCHCODE_TOKEN` in the environment of the process that starts the MCP server (e.g. via your IDE's MCP server env configuration) so tools that call the API are authenticated. See the CLI reference ("Setting SWYTCHCODE_TOKEN") for shell, env file, and IDE setup.

## MCP server

### Starting the server

```bash
# stdio (default) — used by editors that spawn the process at startup
swytchcode mcp serve

# HTTP/SSE — preferred for editors; daemon can start independently of the editor
swytchcode mcp serve --transport http --port 5476

# HTTP/SSE daemon (background, survives terminal close)
swytchcode mcp serve --transport http -d
```

- **stdio transport** (default): editors spawn the process and communicate over stdin/stdout. Tools are only available while the process runs and require an editor restart to activate.
- **HTTP/SSE transport**: the server binds to `127.0.0.1:5476` and exposes `/sse` (event stream) and `/message` (JSON-RPC POST). Editors connect to the already-running daemon — **no restart required**.
- The default port is **5476** (`constants.MCPDefaultPort`). Override with `--port`.
- The server binds to `127.0.0.1` (localhost only) and is not reachable externally. No bearer token is required for local use.

### Exposed MCP tools

From `internal/mcp/tools.go`, the server exposes tools that roughly mirror CLI commands:

- `swytchcode_init`
- `swytchcode_bootstrap`
- `swytchcode_version`
- `swytchcode_list`
- `swytchcode_search`
- `swytchcode_get`
- `swytchcode_add`
- `swytchcode_info`
- `swytchcode_exec`
- `swytchcode_check` — Check for integration updates (TinyFish)
- `swytchcode_inspect` — Show full upgrade proposal detail
- `swytchcode_upgrade` — Approve a pending upgrade
- `swytchcode_discover` — Semantic capability discovery by natural language intent
- `swytchcode_plan` — Show workflow steps for a canonical workflow ID

Clients call these tools; the server:

- Validates inputs.
- Invokes the corresponding logic in `internal/commands` or `kernel.Execute`.
- Normalizes responses into MCP result objects.

**Execution still goes through the kernel** – only `swytchcode_exec` actually runs tools; the rest are discovery and configuration helpers.

### Exec vs MCP HTTP server and integration base URLs

- `swytchcode_exec` **always** calls the kernel, which in turn:
  - Reads `tooling.json` to resolve the tool and integration.
  - Reads `.swytchcode/integrations/manifest.json` to resolve the base URL for that integration based on `mode` (`sandbox_endpoint` vs `production_endpoint`).
  - Builds the final URL as `baseURL + endpoint` (endpoint comes from the Wreken `METHODS` definition).
- The **MCP HTTP transport** (when you run `swytchcode mcp serve --transport http --port 5476`) is only a transport for MCP clients. It is **not** the base URL for tools executed via `swytchcode_exec`.
  - You can have MCP listening on `http://localhost:5476` while a tool targets `https://api.example.com` or any other host from `manifest.json`.
- If an integration bundle has an empty or placeholder endpoint for the active `mode`, the CLI falls back to `http://localhost` as the base URL for that integration (see “Base URL Resolution” in the main README).
  - In that case, a 404 like `Route PUT:/... not found` is coming from the **target API** at `http://localhost`, not from the MCP server.
  - To fix it, update the integration’s entry in `.swytchcode/integrations/manifest.json` so `sandbox_endpoint` / `production_endpoint` point at the correct API host, or start the appropriate backend locally on the configured base URL.

### Daemon mode

- `swytchcode mcp status` – Check if the MCP server daemon is running.
- `swytchcode mcp stop` – Stop the daemon and clean up PID files.

These commands are implemented in `internal/cli/mcp_unix.go`, `internal/cli/mcp_windows.go`, and `internal/mcp/pid.go`.

## Editor integrations

Swytchcode integrates with editors (Cursor, Claude) by installing rule templates that instruct them to use `swytchcode exec` instead of running tools directly.

### Cursor

Steps:

1. Install Swytchcode.
2. In your project:

   ```bash
   swytchcode init --editor=cursor --mode=sandbox
   ```

3. Add tools:

   ```bash
   swytchcode add <canonical_id>
   ```

What happens:

- `init` writes `.cursor/rules/swytchcode.mdc` from the embedded template in `internal/editors/templates/cursor/swytchcode.mdc`.
- `init` writes (or updates) `~/.cursor/mcp.json` with an SSE URL entry:
  ```json
  { “mcpServers”: { “swytchcode”: { “url”: “http://localhost:5476/sse” } } }
  ```
- `init` starts the MCP HTTP daemon (`swytchcode mcp serve --daemon --transport http`) if it isn't already running.

**No editor restart required.** The daemon is running before init finishes; Cursor can connect to `http://localhost:5476/sse` immediately.

Behavior:

- Cursor (and its MCP client) can call all swytchcode MCP tools:
  - `swytchcode_list`, `swytchcode_info`, `swytchcode_get`, `swytchcode_add`, `swytchcode_exec`, etc.
- Only tools in `tooling.json` can be executed.
- On “tool not found” or “integration not installed”, `exec` fails and the agent should surface the error, not retry with something else.

### Claude

Steps:

1. Install Swytchcode.
2. In your project:

   ```bash
   swytchcode init --editor=claude --mode=sandbox
   ```

3. Add tools as usual: `swytchcode add <canonical_id>`.

What happens:

- `init` writes `CLAUDE.md` from the embedded template in `internal/editors/templates/claude/CLAUDE.md`.
- `init` writes (or updates) `~/.claude/settings.json` with an SSE MCP entry:
  ```json
  { “mcpServers”: { “swytchcode”: { “type”: “sse”, “url”: “http://localhost:5476/sse” } } }
  ```
- `init` starts the MCP HTTP daemon (`swytchcode mcp serve --daemon --transport http`) if it isn't already running.

**No editor restart required.** The daemon is running immediately; Claude Code can connect to the SSE endpoint without restarting.

Behavior:

- Claude is given:
  - The contract (allowed tools from `tooling.json`) via `CLAUDE.md`.
  - The full swytchcode MCP toolset via the SSE connection.
  - The execution entrypoint: `swytchcode_exec`.
- Claude should:
  - Use `swytchcode_list` / `swytchcode_info` for discovery.
  - Use `swytchcode_exec` for execution.
  - Respect exit codes and JSON error output.

### General guidance for MCP clients

Regardless of the client (Cursor, Claude, custom MCP client):

- Do **not**:
  - Execute tools directly via the underlying APIs.
  - Retry with different tools if `swytchcode_exec` fails.
  - Decide policy from Wrekenfiles; use `tooling.json`.

- Do:
  - Use discovery tools (`swytchcode_list`, `swytchcode_info`) to understand capabilities.
  - Use `swytchcode_exec` to run tools.
  - Surface kernel errors to users, including the JSON error payload and exit codes.

