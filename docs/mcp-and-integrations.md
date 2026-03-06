# Swytchcode CLI – MCP & Integrations

Swytchcode exposes its functionality over the Model Context Protocol (MCP) so editors and agents can call it in a standard way. This document explains how MCP tools map to CLI commands and how editor integrations are wired.

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

Clients call these tools; the server:

- Validates inputs.
- Invokes the corresponding logic in `internal/commands` or `kernel.Execute`.
- Normalizes responses into MCP result objects.

**Execution still goes through the kernel** – only `swytchcode_exec` actually runs tools; the rest are discovery and configuration helpers.

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

