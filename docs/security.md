# Security model and trust boundaries

Swytchcode is a **local execution kernel**: `swytchcode exec` resolves tools from your project’s `tooling.json` and integration bundles on disk. This page summarizes trust boundaries, transport rules, and environment flags. For configuration detail, see [config-spec.md](config-spec.md) and [architecture.md](architecture.md).

## What is trusted

- **tooling.json** — Allow-list of canonical tool IDs, workflow definitions, and execution mode. Only listed tools run.
- **Local Wrekenfiles** — Define HTTP shape (method, path, headers). The kernel builds requests from this contract plus caller-supplied args.
- **manifest.json** — Per-integration base URLs and optional `execution_policy` (retries, timeouts, idempotency).

## What is not trusted as “implicit policy”

- **Agents and editors** must not execute APIs directly or bypass `swytchcode exec`. They should treat `tooling.json` as read-only for policy.
- **User/agent-supplied args** (body, headers, query params) are passed through after schema validation; secrets belong in env or headers the caller provides, not in committed files unless you accept that risk.

## Registry vs execution

| Surface | Registry / cloud API | Tool execution (`exec`) |
|--------|----------------------|-------------------------|
| When used | `get`, `bootstrap`, `search`, workflow fetch, auth commands | After bundles are on disk |
| Network | Yes | Single-method: **no** registry at runtime; workflows may fetch definitions |

## Integration base URLs (HTTPS)

The kernel rejects non-loopback **`http://`** base URLs. **`https://`** is required for remote hosts; **`http://`** is allowed only for **`localhost`**, **`127.0.0.1`**, and **`::1`**. Same rules in CI and containers. See [config-spec.md](config-spec.md) (manifest → HTTPS and HTTP base URLs).

## `SWYTCHCODE_INSECURE=1`

- Disables TLS certificate verification for shared HTTP clients (registry, execution, auth-related calls, telemetry).
- Intended **only** for local development with self-signed certificates.
- Outside CI, the CLI prints a **one-time stderr warning**.
- When **`CI`**, **`GITHUB_ACTIONS`**, or **`GITLAB_CI`** is truthy, **registry** HTTP requests **fail** if this variable is set (execution URL rules are unchanged; arbitrary `http://` non-loopback URLs remain invalid).

## Authentication

- **`SWYTCHCODE_TOKEN`** — Service token from the environment (CI/agents).
- **`~/.swytchcode/auth.json`** — User session from `swytchcode login`.
- **`exec`** does not require auth unless you rely on registry-backed workflows or your target API needs credentials in args/headers.

## MCP and editors

The MCP HTTP server binds to **loopback** only. It is **not** the integration base URL for tools; execution targets come from `manifest.json`. See [mcp-and-integrations.md](mcp-and-integrations.md).

## Wrekenfile signatures

**Not implemented yet** in the CLI. When added, they will be an additional integrity check on bundle content; today, trust comes from the registry at `get` time and your version pins in `tooling.json`.

## Diagnostics

Run **`swytchcode doctor`** (or MCP **`swytchcode_doctor`**) for a local checklist: tooling and manifest parse, bundles on disk, base URL validation, auth posture, and `SWYTCHCODE_INSECURE` / CI interaction.
