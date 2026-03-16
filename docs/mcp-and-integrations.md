# Swytchcode CLI – MCP & Integrations

Swytchcode exposes its functionality over the Model Context Protocol (MCP) so editors and agents can call it in a standard way. This document explains how MCP tools map to CLI commands and how editor integrations are wired.

## Authentication

The MCP server uses the same auth as the CLI: `SWYTCHCODE_TOKEN` or `~/.swytchcode/auth.json`. Set `SWYTCHCODE_TOKEN` in the environment of the process that starts the MCP server (e.g. via your IDE's MCP server env configuration) so tools that call the API are authenticated. See the CLI reference ("Setting SWYTCHCODE_TOKEN") for shell, env file, and IDE setup.

## MCP server

### Starting the server

```bash
swytchcode mcp serve
```

- Runs over stdio by default.
- HTTP transport can be enabled via `internal/mcp/transport.go` (using `MCPDefaultPort` and `MCPBearerToken` from `internal/constants`).

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
- The **MCP HTTP transport** (when you run `swytchcode mcp serve --transport http --port 3000`) is only a transport for MCP clients. It is **not** the base URL for tools executed via `swytchcode_exec`.
  - You can have MCP listening on `http://localhost:3000` while a tool targets `https://api.example.com` or any other host from `manifest.json`.
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
   swytchcode init --editor=cursor
   ```

3. Add tools:

   ```bash
   swytchcode add <canonical_id>
   ```

What happens:

- `init` writes `.cursor/rules/swytchcode.mdc` from the embedded template in `internal/editors/templates/cursor/swytchcode.mdc`.
- This rule:
  - Tells Cursor how to discover tools (read-only view over `tooling.json` + integrations).
  - Tells Cursor to call `swytchcode exec` for execution.

Behavior:

- Cursor (and its MCP client, if used) can call:
  - `swytchcode list`
  - `swytchcode info`
  - `swytchcode exec`
- Only tools in `tooling.json` can be executed.
- On “tool not found” or “integration not installed”, `exec` fails and the agent should surface the error, not retry with something else.

### Claude

Steps:

1. Install Swytchcode.
2. In your project:

   ```bash
   swytchcode init --editor=claude
   ```

3. Add tools as usual: `swytchcode add <canonical_id>`.

What happens:

- `init` writes `CLAUDE.md` from the embedded template in `internal/editors/templates/claude/CLAUDE.md`.
- `CLAUDE.md` describes:
  - How Claude should interpret `tooling.json` and integrations.
  - How to call `swytchcode exec` as the single execution entrypoint.

Behavior:

- Claude is given:
  - The contract (allowed tools from `tooling.json`).
  - The execution entrypoint: `swytchcode exec`.
- Claude should:
  - Use `swytchcode list` / `swytchcode info` for discovery.
  - Use `swytchcode exec` for execution.
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

