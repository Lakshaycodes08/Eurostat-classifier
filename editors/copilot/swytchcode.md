# Swytchcode Rules (GitHub Copilot)

Swytchcode is the **only execution authority** for all external APIs,
integrations, methods, and workflows in this repository.

You are operating in **IDE code-generation mode**.

---

## Mandatory Workflow (ALWAYS FOLLOW)

When generating code that involves an external API or integration:

1. Assume the API is executed ONLY via Swytchcode.
2. Assume the tool or workflow already exists OR ask the user to add it.
3. Generate runtime code that delegates execution to Swytchcode.
4. Do NOT execute the tool or fetch live data.

If any information is missing, STOP and ask the user.

---

## Hard Constraints (STRICT)

You MUST NOT:

- Generate raw HTTP requests (fetch, axios, curl, requests, etc.)
- Use SDKs for external services (Stripe SDK, AWS SDK, etc.)
- Embed endpoints, URLs, headers, or credentials
- Guess or invent API behavior
- Read or reason about `.swytchcode/` files
- Generate code that executes APIs directly

All API execution MUST go through Swytchcode.

---

## Tool Knowledge

Copilot MUST assume:

- Tool names (canonical IDs) are discovered externally
- Input/output schemas are provided explicitly or via Swytchcode info
- Execution behavior is opaque and owned by Swytchcode

Copilot MUST NOT:

- Infer parameter names
- Infer request/response formats
- Infer pagination, retries, or errors

If unsure, STOP and ask the user.

---

## Methods vs Workflows

- Methods and workflows are both executable via Swytchcode.
- Workflows may internally reference multiple methods.
- Workflows are opaque.

Copilot MUST NOT:
- Inline workflow logic
- Expand workflows into steps
- Reimplement workflow behavior

---

## Code Generation Rules

Generated code MUST:

- Delegate execution to Swytchcode at runtime
- Use an official Swytchcode runtime library (swytchcode-runtime for go, python and javascript, currently) if available
- Otherwise invoke Swytchcode via subprocess
- Pass a single structured input object
- Handle stdout, stderr, and exit codes

Generated code MUST NOT:
- Implement custom API clients
- Hardcode API versions or URLs
- Depend on external service SDKs

---

## Execution Modes

- IDE mode = generate code only
- Direct execution = NOT allowed for Copilot

Copilot MUST NEVER:
- Execute tools
- Return live API data
- Simulate execution results

---

## Determinism

Copilot MUST:

- Generate deterministic, reproducible code
- Rely only on explicit inputs and contracts
- Avoid speculative or placeholder logic

Progress without certainty is forbidden.

---

## Mental Model (IMPORTANT)

Copilot is **not** an API client.

Copilot is a **code generator targeting Swytchcode**.

Copilot:
- Generates code
- Delegates execution

Swytchcode:
- Executes
- Authenticates
- Enforces policy

---

**End of Rules**
