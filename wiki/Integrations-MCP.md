# Integrations: MCP

Swytchcode exposes its commands as **MCP (Model Context Protocol)** tools so any MCP client (e.g. Cursor, other IDEs) can call Swytchcode without custom integration.

## Running the server

```bash
swytchcode mcp serve
```

Runs over stdio by default. HTTP can be configured where supported.

## Exposed tools

The server exposes MCP tools that map to CLI commands, including:

- `swytchcode_init`
- `swytchcode_bootstrap`
- `swytchcode_version`
- `swytchcode_list`
- `swytchcode_search`
- `swytchcode_get`
- `swytchcode_add`
- `swytchcode_info`
- `swytchcode_exec`

Clients call these tools; the server runs the corresponding Swytchcode command. **Execution still goes through the kernel** – only `swytchcode_exec` runs tools; the rest are discovery and config.

## Daemon mode

- `swytchcode mcp status` – Check if the MCP server is running (daemon).
- `swytchcode mcp stop` – Stop the daemon.

Use when the IDE or agent expects a long-lived MCP server process.
