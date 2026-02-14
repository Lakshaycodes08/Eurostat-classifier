# Swytchcode Execution Rules

Swytchcode is the sole authority for executing external API calls in this repository.

You are operating in **IDE code generation mode**.

---

## Forbidden

You MUST NOT:

- Read, inspect, or modify any files under `.swytchcode/`
- Read or interpret Wrekenfiles, `methods.json`, or `workflows.json`
- Generate raw HTTP requests to external APIs
- Embed authentication tokens, headers, or credentials in code
- Use MCP to fetch or display live API data
- Call `swytchcode exec` (or `swytchcode_exec`) to obtain results directly

---

## Required behavior

When an external API interaction is needed, you MUST:

- Use Swytchcode MCP tools **only for discovery**
- Generate **runtime application code** that delegates execution to Swytchcode
- Treat Swytchcode as a black-box execution kernel

Generated code MUST:

- Invoke the Swytchcode kernel at runtime by:
  - spawning the `swytchcode` CLI, OR
  - using an approved Swytchcode runtime helper (TypeScript / Go / Python)
- Execute `swytchcode exec <canonical_id>` when the application runs
- Return executable code, not API responses

---

## MCP usage

Allowed MCP usage in IDE mode:

- Discovery only (e.g. listing available tools, describing schemas)

Forbidden MCP usage in IDE mode:

- Direct execution to fetch, preview, or display live data
- Any MCP action that mutates `.swytchcode` unless explicitly requested by the user

---

## Execution modes (important)

- **IDE agents** generate code only. They never return live API data.
- **Non-IDE agents** (analysis, inspection, automation) may call `swytchcode_exec` directly and return results.

External APIs must never be called directly.  
All execution authority belongs to Swytchcode.
