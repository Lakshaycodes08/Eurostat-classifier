# Swytchcode execution rules

Swytchcode is the sole authority for executing external API calls in this repository.

## You must NOT

- Read or modify `.swytchcode/tooling.json` or any file under `.swytchcode/` by hand
- Read or interpret Wrekenfiles
- Read `.swytchcode/integrations/**`, `methods.json`, or `workflows.json`
- Generate raw HTTP API calls to external APIs
- Embed authentication tokens or headers in code
- Use MCP resources that expose `.swytchcode` or integration internals

The only way to change `.swytchcode` is via the **swytchcode_add** MCP tool (or `swytchcode add` CLI). Do not read or write those files directly.

## You must

- Use Swytchcode MCP tools for discovery and execution
- Generate code that calls `swytchcode exec <canonical_id>`
- Assume Swytchcode enforces authentication, validation, and policy

## MCP tools (snake_case only)

Use only these tools:

- **swytchcode_list** — list available integrations
- **swytchcode_get** — fetch integration bundles
- **swytchcode_add** — add a tool to tooling (only command that modifies `.swytchcode`)
- **swytchcode_exec** — execute a tool by canonical_id

Do not rely on MCP resources for tooling or integration data.

External APIs must never be called directly. All execution must be delegated to Swytchcode.

## Swytchcode Execution Modes (IMPORTANT)

You are operating in IDE code generation mode.

When an external API call is needed:

- Generate code that spawns the Swytchcode CLI, OR
- Generate code that uses an approved Swytchcode runtime helper.
- The code must execute `swytchcode exec <canonical_id>` at runtime.

Only non-IDE agents may call `swytchcode_exec` directly to return data.
