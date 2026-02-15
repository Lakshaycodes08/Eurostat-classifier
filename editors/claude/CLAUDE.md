# Swytchcode Execution Rules

Swytchcode is the sole authority for executing external API calls.

You are operating in **IDE code generation mode**.

---

## Default workflow (follow this unless instructed otherwise)

When asked to integrate or use an external API:

1. Use Swytchcode discovery tools to find available integrations and tools.
2. Discover canonical IDs and required parameters via discovery—do not guess by trial and error.
3. If a required tool is missing from tooling.json, ask the user before adding it.
4. Add tools only after explicit user approval.
5. Generate runtime application code that delegates execution to Swytchcode at runtime.
6. Do NOT fetch or return live API data unless explicitly requested.

If a tool requires parameters, discover required inputs via Swytchcode discovery rather than inferring them from failed executions.

---

## Forbidden

You MUST NOT:

- Read or inspect `.swytchcode/` files directly (including Wrekenfiles, `methods.json`, `workflows.json`)
- Generate raw HTTP requests to external APIs
- Embed credentials or auth logic in code
- Execute tools to fetch live data in IDE mode
- Guess canonical IDs by trial and error
- Add tools to tooling.json without explicit user approval
- Generate code that shells out to the `swytchcode exec` command (e.g. in a terminal or subprocess string). Use a runtime library or the programmatic CLI contract instead (see below).

---

## Required behavior

Generated code MUST:

- Call Swytchcode at runtime (via a runtime library or the programmatic CLI contract).
- Return executable code, not execution results.

Invoke the kernel in one of these ways only:

- **Preferred:** Use an official Swytchcode runtime library for the target language (Node/TS, Go, Python) when available.
- **If no runtime exists:** Invoke the CLI programmatically: spawn the process, send a single JSON object to stdin (kernel args: `body`, `params`, `Authorization`, `headers`, etc.), read stdout and stderr, parse JSON. Do not construct or run the `swytchcode exec ...` command as a shell string.

See the project’s **runtime-libraries README** (Other languages / CLI contract) for the exact stdin/stdout/stderr and exit-code contract.

---

## MCP usage

- **Allowed:** Discovery only (list integrations, list tools, describe schemas).
- **Forbidden:** Direct execution to fetch or display live data; any mutation of `.swytchcode` without explicit user approval.

---

## Execution modes

- **IDE agents** generate code only. They never return live API data.
- **Non-IDE agents** may execute tools and return results.

All execution authority belongs to Swytchcode.
