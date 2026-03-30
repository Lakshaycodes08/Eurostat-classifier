# Execution Model

## Single entrypoint

**`swytchcode exec`** is the only integration entrypoint. No other code path executes methods or workflows. Editors and agents must call `exec`; they must not run tools directly or interpret policy themselves.

## Why exec is the only path

- **Authority** – One place to enforce tooling.json and resolve canonical IDs.
- **Determinism** – Same request → same behavior; no prompt-based branching.
- **Failure semantics** – Exit codes and JSON errors are defined in one place; agents stop on "tool not found" or "integration not installed."

## Apps must not interpret policy

Agents and editors should be **read-only** over the tool list. They may call `swytchcode list` and `swytchcode info` to discover what is available, but they must not:

- Retry or fall back to other tools when exec returns an error.
- Decide on their own that a tool is "allowed" without it being in tooling.json.
- Execute anything outside of `swytchcode exec`.

## Retries and idempotency

Retries and idempotency belong **inside** Swytchcode (or inside the tool implementation), not in the agent. If exec fails, the agent should surface the error and stop, not improvise.

## Where requests go (base URL rules)

The kernel resolves `sandbox_endpoint` / `production_endpoint` from `manifest.json` and validates the scheme/host: **`https://`** anywhere, or **`http://`** only on loopback (`localhost`, `127.0.0.1`, `::1`). The same rules apply in CI and containers. Details: [docs/config-spec.md](https://gitlab.com/swytchcode/swytchcode-cli/-/blob/main/docs/config-spec.md) (manifest section).

## Integration not found

If an integration is not installed or a tool is not in tooling.json, exec fails explicitly. Agents must **stop** and report; they must not try alternative execution paths.
