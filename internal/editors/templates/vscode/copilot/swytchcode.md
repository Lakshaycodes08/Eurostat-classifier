# AI / Agent rules (Swytchcode)

This repository uses Swytchcode as the sole authority for external API execution.

## AI agents must

- Use Swytchcode MCP tools for discovery and execution: **swytchcode_list**, **swytchcode_get**, **swytchcode_add**, **swytchcode_exec**
- Generate code that calls `swytchcode exec <canonical_id>`

## AI agents must NOT

- Read or modify `.swytchcode/*` by hand (the only way to change it is via **swytchcode_add** or `swytchcode add`)
- Generate raw HTTP API calls to external APIs
- Embed API keys or auth logic in code
- Use MCP resources that expose `.swytchcode` or integration internals

External APIs must never be called directly. All execution must go through Swytchcode.

## Swytchcode Usage Rules

### IDE Mode (Copilot / Editor Assistance)

In this repository, Copilot operates in IDE mode.

In IDE mode:

- Do NOT call `swytchcode_exec` to retrieve or display live data.
- Do NOT generate raw HTTP API calls.
- Always generate executable code that invokes the Swytchcode kernel at runtime.

Generated code must:

- Spawn the Swytchcode CLI, or
- Use an approved Swytchcode runtime helper library
- Call `swytchcode exec <canonical_id>` when the application runs

### Direct Execution Mode (Non-IDE agents only)

Only non-IDE agents may call `swytchcode_exec` directly to fetch data.
