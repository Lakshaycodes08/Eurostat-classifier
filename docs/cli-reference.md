# Swytchcode CLI – Command Reference

This document summarizes the Swytchcode CLI surface, with a focus on inputs, outputs, and behavior. For detailed architecture and execution semantics, see `docs/architecture.md` and `docs/execution-model.md`.

## Commands at a glance

| Command | Purpose |
|--------|---------|
| `swytchcode -v` / `--version` | Show CLI version. |
| `swytchcode init` | Initialize `.swytchcode/`, create `tooling.json`, install editor rules (Cursor / Claude). |
| `swytchcode get <project>` | Fetch integration bundles (Wrekenfiles, methods, workflows) for a project; does **not** modify `tooling.json`. |
| `swytchcode bootstrap` | Fetch all integrations declared in `tooling.json.integrations` (CI-friendly). |
| `swytchcode list` | List tools and integrations from local state (`tooling.json` + `.swytchcode/integrations`). |
| `swytchcode list methods [pattern]` | List local methods (optional filter by canonical ID / project). |
| `swytchcode list workflows [pattern]` | List local workflows (optional filter). |
| `swytchcode list integrations` | List installed integrations only. |
| `swytchcode search [keyword]` | Search the remote registry for integrations. |
| `swytchcode add [spec] <canonical_id>` | Add a tool to `tooling.json` from fetched integrations. |
| `swytchcode info <canonical_id>` | Show detailed info about a tool (resolved inputs/output). |
| `swytchcode exec [canonical_id]` | **Single execution path** – run a tool via the kernel. Shorthand: `swytchcode <canonical_id>` (omit `exec`). |
| `swytchcode demo list` | List all tools that have a demo available (no setup required). |
| `swytchcode mcp serve` | Start MCP server (stdio/HTTP). |
| `swytchcode mcp status` / `swytchcode mcp stop` | Daemon status / stop. |
| `swytchcode sync [project]` | Re-fetch workflow/method list from backend; updates local files without touching `tooling.json`. Warns on stale method hashes. |
| `swytchcode discover <intent>` | Semantic search: find methods and workflows by natural-language description. |
| `swytchcode plan <canonical_id>` | Preview ordered steps of a workflow before running it. |
| `swytchcode doctor` | Local diagnostics: `tooling.json`, bundles, `manifest.json`, HTTPS base URLs, auth; `--json` for machines; exits **1** if any error-level check fails. |
| `swytchcode check` | Check for integration update proposals from the backend. |
| `swytchcode login` / `logout` / `whoami` | Manage CLI auth sessions. |
| `swytchcode inspect <library>` | Show full proposal detail for a library (requires login). |
| `swytchcode upgrade <library> [--apply]` | Approve a pending integration update proposal (requires login). `--apply` also refreshes local bundle and tooling.json. |
| `swytchcode diff <library>` | Show method-level signature diff for a pending upgrade proposal (requires login or token). |

## swytchcode doctor

- **Flags:** `--json` — emit the full report as JSON (`ok`, `checks[]` with `id`, `status`, `message`).
- **Exit code:** `0` if no check has status `error`; `1` otherwise. Warnings do not fail the command.
- **Scope:** Read-only; no registry or execution calls. Validates integration bundles with the same load path as `exec`.

See [docs/security.md](security.md) and [docs/windows-guide.md](windows-guide.md) for related operational guidance.

## swytchcode exec

### Demo mode

Run any tool without a project or API keys:

```bash
# Shorthand (no 'exec' needed) — auto-detects demo mode
npx swytchcode stripe.create_payment

# Explicit demo flag
swytchcode exec stripe.create_payment --demo
```

When no `.swytchcode/` project is found, demo mode activates automatically. The CLI calls the Swytchcode demo API, which uses shared sandbox credentials (e.g. Stripe test keys) and returns a real sandbox response. No `tooling.json`, no auth required.

```
✔  Payment created successfully (demo)

→ payment_id: pi_3QxR...
→ amount: $20.00
→ currency: USD
→ status: requires_payment_method
```

After a successful demo run, the CLI prints an upgrade prompt to stderr:

```
────────────────────────────────────────
To use real integrations:
  swytchcode init
────────────────────────────────────────
```

### Input

- CLI args:

```bash
swytchcode exec api.cluster.create \
  --body request.json \
  --input env=prod \
  --param debug=true \
  --header X-Trace-Id=123
```

- JSON stdin:

```bash
echo '{"tool":"api.cluster.create","args":{"name":"my-cluster","region":"eu-west-1"}}' \
  | swytchcode exec
```

Arguments:

- `tool` – canonical ID (string, required).
- `args` – JSON object with:
  - Arbitrary fields matching the integration’s input schema.
  - Optional `body`, `params`, `headers` when using HTTP features.

### Output

- Default: JSON object on stdout describing the response, including:
  - Request URL (so you can verify base URL and path).
  - Response status, headers, and body (normalized where possible).
- `--demo`: run in demo mode — no project setup or API keys required. Calls the Swytchcode demo API using shared sandbox credentials. Activates automatically when no `.swytchcode/` project is detected.
- `--raw`: write raw HTTP response to stdout/stderr instead of normalized JSON.
- `--dry-run`: do not execute; print a representation of the request that would be sent.
- `--verbose`: log request and response details to stderr before and after the HTTP call. Output is two JSON lines (one per event) with `"verbose":"request"` and `"verbose":"response"`. Sensitive header values (`Authorization`, `X-Api-Key`) are redacted automatically.

```bash
swytchcode exec stripe.checkout_session_create \
  --body request.json --verbose 2>debug.log
```

- `--output <file>`: write the response body to a file instead of stdout when the API returns a binary `Content-Type` (e.g. `application/pdf`, `image/png`). stdout still receives a JSON summary:

```json
{ "saved_to": "invoice.pdf", "size_bytes": 45231, "content_type": "application/pdf", "status_code": 200 }
```

If the response is binary and `--output` is omitted, the CLI exits with an error rather than writing binary data to the terminal.

Errors are written to stderr as:

```json
{ "error": "message", "category": "network", "retryable": true }
```

`category` values: `auth` | `validation` | `not_found` | `network` | `rate_limit` | `internal`. `retryable` is `true` when the error is transient (network failures, rate limiting).

### Exit codes (from `internal/kernel/errors.go`)

- `0` – Success **or API-level error** (4xx/5xx from the target API). Check `status_code` in the JSON output to distinguish success from API failure.
- `1` – Invalid input (bad JSON, missing `tool`, validation error, invalid flags).
- `2` – Tool not found (missing in `tooling.json` or bundle).
- `3` – Auth error (reserved for auth-related exec failures).
- `4` – SDK failure (network/HTTP errors when calling the target API — DNS, timeout, connection refused).
- `5` – Internal error (unexpected conditions, project root detection errors, etc).

**API errors vs CLI errors:** The CLI exits 0 whenever it successfully received an HTTP response from the target API, regardless of the HTTP status code. The status code and error body are always available in the JSON output on stdout. Exit codes 1–5 indicate CLI-level problems (bad input, missing tool, network failure) where no API response was received.

**Checking for API errors in code:**
```js
const result = JSON.parse(stdout);
if (result.status_code >= 400) {
  // API-level error — result.data has the error body
  throw new Error(`API error ${result.status_code}: ${JSON.stringify(result.data)}`);
}
```

**Workflow output:** A workflow always exits 0. Check `result.success` to detect step failures:
```js
const result = JSON.parse(stdout);
if (!result.success) {
  // result.error has the failure message; result.steps includes the failed step's data
  const failedStep = result.steps.find(s => s.failed);
}
```

## swytchcode demo

### demo list

Lists all tools that have a demo available. Fetches live from the Swytchcode demo API — no setup or auth required.

```bash
swytchcode demo list
```

Output groups tools by integration prefix:

```
Available demos:

  Stripe
  → stripe.create_payment

Run any demo:
  npx swytchcode <tool>
```

See [demo mode in exec](#swytchcode-exec) for how to run a demo tool.

---

## init

```bash
swytchcode init
swytchcode init --editor=cursor --mode=production --non-interactive
```

- Creates `.swytchcode/` and `.swytchcode/integrations/`.
- Writes `tooling.json` with:
  - `version`
  - `mode` (`production` or `sandbox`)
  - empty `integrations` and `tools` maps.
- Installs editor rules and registers the MCP server:
  - **Cursor**: writes `.cursor/rules/swytchcode.mdc`; merges `{"url":"http://localhost:5476/sse"}` into `~/.cursor/mcp.json`
  - **Claude**: writes `CLAUDE.md`; merges `{"type":"sse","url":"http://localhost:5476/sse"}` into `~/.claude/settings.json`
- Starts the MCP HTTP daemon in the background (if not already running) so editor tools are available immediately — **no editor restart required**.

## get

```bash
swytchcode get <project>
```

- Fetches integration bundles (Wrekenfiles, methods, workflows) for `<project>` from the registry.
- Writes them under `.swytchcode/integrations/<project>/<library>/<version>/`.
- Updates `manifest.json` with endpoints and counts.
- Does **not** modify `tooling.json`; use `add` to enable tools.

## bootstrap

```bash
swytchcode bootstrap
```

- Reads `tooling.json.integrations`.
- For each `project.library` with a version:
  - Ensures corresponding bundles are fetched and on disk.
  - Updates `manifest.json`.

Intended for CI to keep `.swytchcode/integrations` in sync with `tooling.json`.

## list

```bash
swytchcode list
swytchcode list methods [pattern]
swytchcode list workflows [pattern]
swytchcode list integrations [pattern]
swytchcode list tooling [pattern]
```

- Reads only local state:
  - `.swytchcode/integrations` (Wrekenfiles, methods.json, workflows.json).
  - `tooling` filter reads from `.swytchcode/tooling.json` — shows only what has been
    explicitly enabled via `swytchcode add`. Use this to verify a canonical ID is
    registered before generating execution code.
- Default output:
  - Human-readable lists of methods, workflows, and integrations.
- `--json` (where supported) returns a machine-readable `ListResult`:
  - Arrays of `{ "canonical_id": "...", "integration": "project.library@version" }`.

## search

```bash
swytchcode search
swytchcode search weaviate
```

- Contacts the registry (using `RegistryURL` / `SWYTCHCODE_API_URL` depending on context).
- Lists available integrations.
- Does not modify local state; use `get` / `bootstrap` to fetch bundles.

## add

```bash
swytchcode add <canonical_id>
swytchcode add <project@library.version> <canonical_id>
swytchcode add --all <project>
```

- Reads integration bundles and Wrekenfiles from `.swytchcode/integrations`.
- Resolves the requested method/workflow and its STRUCTs into concrete input/output schemas.
- Adds an entry in `tooling.json.tools` for that canonical ID.
- Stores a `method_hash` (SHA-256 of the wrekenfile entry) to enable stale detection in `sync`.

If the canonical ID exists in multiple integrations, `add` may require an explicit `project@library.version` to disambiguate.

**Flags:**
- `--all <project>` — Add all methods and workflows for the project. Skips already-present IDs. Prints a summary.
- `--no-auto-install` — Do not auto-download missing library deps for multi-library workflows.

## info

```bash
swytchcode info <canonical_id>
```

- Shows rich information for a tool:
  - Source integration.
  - Summary/description.
  - Resolved input and output schemas (STRUCTs expanded).
- Uses both `tooling.json` and integration artifacts (`wrekenfile.yaml`, `methods.json`, `workflows.json`) to compute the result.

## sync

```bash
swytchcode sync
swytchcode sync <project>
```

- Re-fetches the workflow list from the backend for each installed project (or the named one).
- Compares against local `workflows.json`; if changes are found, re-downloads the full bundle.
- Prints new workflows and warns about updated workflows already in `tooling.json`.
- Does **not** modify `tooling.json`.
- After bundle refresh, re-hashes all method entries against stored `method_hash` values in `tooling.json`. Warns if any differ: `⚠ method X has changed — run: swytchcode add X to refresh tooling.json`.

**Error messages:**
- `"no integrations found — run: swytchcode get <project>"` — No projects installed.
- `"fetch workflows from backend: ..."` — Network or auth error.

## discover

```bash
swytchcode discover "<intent>"
swytchcode discover "<intent>" --project <name>
swytchcode discover "<intent>" --library <name>
swytchcode discover "<intent>" --top 10 --json
```

- Sends a semantic search query to the backend (`POST /v2/cli/discover`).
- Returns ranked methods and workflows matching the plain-English intent.

**Flags:**
- `--project <name>` — Scope to a specific project.
- `--library <name>` / `-l` — Scope to a specific library within a project.
- `--top <n>` — Number of results (default: 5, max: 50).
- `--json` — Output raw JSON.

**Output:** Each result shows canonical ID, type, integration, confidence score, and a ready-to-use `swytchcode exec` snippet.

## plan

```bash
swytchcode plan <canonical_id>
swytchcode plan <canonical_id> --json
```

- Fetches the workflow definition from the registry and prints the ordered step list.
- Shows step name, canonical ID, and integration for each step.
- Does not execute anything.

**Flags:**
- `--json` — Output raw JSON.

**Requirements:** Requires login or `SWYTCHCODE_TOKEN`.

## diff

```bash
swytchcode diff <library>
swytchcode diff stripe.payments
```

- Fetches the pending upgrade proposal diff from the backend (`GET /v2/cli/proposals/diff?library=<library>`).
- Prints a formatted diff of what would change if the proposal were approved.

**Output format:**
```
stripe  v1 → v2

ADDED    stripe.create_payment_link
REMOVED  stripe.legacy_charge  [breaking]
CHANGED  stripe.charge_customer
  + inputs.idempotency_key              (string, optional)
  - inputs.source                       (string) [breaking]
  ~ inputs.amount                       type: int → float

Summary: 1 added, 1 removed, 1 changed

To apply: swytchcode upgrade stripe
```

**Requirements:** Requires login or `SWYTCHCODE_TOKEN`. Depends on backend endpoint `GET /v2/cli/proposals/diff` (see `BACKEND_API_CONTRACTS.md`).

## MCP commands

- `swytchcode mcp serve`
  - Starts the MCP server (stdio by default, HTTP/SSE when `--transport http` is specified).
  - Default HTTP port: **5476** (override with `--port`).
  - HTTP/SSE transport: listens on `127.0.0.1:<port>/sse` and `/message` — localhost only, no auth required.
  - Exposes tools that mirror CLI operations (init, get, bootstrap, list, info, exec, etc.).
  - `swytchcode init --editor=<cursor|claude>` automatically registers the SSE URL in the editor's global MCP config and starts the daemon.

- `swytchcode mcp status`
  - Reports whether the MCP daemon is running.

- `swytchcode mcp stop`
  - Stops the MCP daemon and cleans up PID files.

See `docs/mcp-and-integrations.md` for more detail.

## Auth and backend-related commands

These commands talk to the backend at `SWYTCHCODE_API_URL` (default `https://api-v2.swytchcode.com`) and use `SWYTCHCODE_TOKEN` or `~/.swytchcode/auth.json` for auth:

- `swytchcode login`
- `swytchcode logout`
- `swytchcode whoami`
- `swytchcode check`
- `swytchcode inspect`
- `swytchcode upgrade`
- `swytchcode diff`

See backend-specific docs for exact payloads and behavior. From the CLI's perspective, they:

- Resolve project UUIDs and tokens via `internal/auth`.
- Call backend endpoints for account/project/introspection and plan/usage info.
- Exit with non-zero codes on auth/network/server errors, printing clear messages on stderr.

### Setting SWYTCHCODE_TOKEN

The CLI reads the token only from the **process environment** — it does not load `.env` files.

#### Mac / Linux

| Goal | Command |
|------|---------|
| Current session only | `export SWYTCHCODE_TOKEN=your_token_here` |
| Permanent (Zsh — default on macOS) | `echo 'export SWYTCHCODE_TOKEN=your_token_here' >> ~/.zshrc && source ~/.zshrc` |
| Permanent (Bash) | `echo 'export SWYTCHCODE_TOKEN=your_token_here' >> ~/.bashrc && source ~/.bashrc` |
| Per-directory (direnv) | Add `export SWYTCHCODE_TOKEN=your_token_here` to `.envrc`, then `direnv allow .` |
| From a `.env` file (one-off) | `set -a && source .env && set +a` then run `swytchcode` |

#### Windows

| Goal | Command |
|------|---------|
| Current PowerShell session | `$env:SWYTCHCODE_TOKEN = "your_token_here"` |
| Permanent (PowerShell, user-level) | `[System.Environment]::SetEnvironmentVariable("SWYTCHCODE_TOKEN","your_token_here","User")` |
| Permanent (cmd / setx, user-level) | `setx SWYTCHCODE_TOKEN "your_token_here"` |
| Via GUI | System Properties → Advanced → Environment Variables → User variables → New |

> **Note:** After `setx` or the GUI method, open a **new** terminal window for the change to take effect.

#### Node.js projects (any platform)

If you call `swytchcode` via the `swytchcode-runtime` package and want to load from a `.env` file without exporting manually:

- **Node 20.6+:** `node --env-file=.env src/index.js`
- **dotenv package:** add `require('dotenv').config()` at the top of your entry-point file, then `npm install dotenv`

#### MCP in the IDE

Configure the MCP server's `env` block with `SWYTCHCODE_TOKEN` so the server process inherits it. If you start the MCP server from a terminal, export the token first.

#### CI/CD

Define `SWYTCHCODE_TOKEN` as a secret or CI variable so the job environment has it.

### `SWYTCHCODE_INSECURE` (TLS verification)

When set to `1`, the CLI disables TLS certificate verification for shared HTTP clients (registry, tool execution, auth, telemetry). Use **only** for local development against servers with self-signed certificates.

- Outside CI, a **one-time warning** is printed to stderr.
- If **`CI`**, **`GITHUB_ACTIONS`**, or **`GITLAB_CI`** is truthy (`1`, `true`, `yes`), **registry** `Get`/`Post` calls **fail** when `SWYTCHCODE_INSECURE=1` is set.

This does **not** allow non-loopback **`http://`** integration base URLs. Execution still requires `https://` or `http://` on `localhost` / `127.0.0.1` / `::1` only. See [config-spec.md](config-spec.md) (manifest → HTTPS and HTTP base URLs) and the README “Base URL Resolution” section.

### Telemetry

**Telemetry:** Usage events are sent when authenticated — either via `swytchcode login` (user session) or `SWYTCHCODE_TOKEN` (service token). The backend identifies your account from the bearer token, so server-side exec calls are tracked the same way as interactive ones. When you have no auth, no events are sent and the CLI may print a one-time hint. See `CLI_TELEMETRY.md` in the repo for the full contract (event schema, endpoint, error classification).

