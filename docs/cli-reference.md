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
| `swytchcode exec [canonical_id]` | **Single execution path** – run a tool via the kernel. |
| `swytchcode mcp serve` | Start MCP server (stdio/HTTP). |
| `swytchcode mcp status` / `swytchcode mcp stop` | Daemon status / stop. |
| `swytchcode check` | Check for integration update proposals from the backend. |
| `swytchcode login` / `logout` / `whoami` | Manage CLI auth sessions. |
| `swytchcode inspect` | Inspect account/project and integration usage via the backend. |
| `swytchcode upgrade` | Check or trigger CLI upgrade behavior via the backend. |

## swytchcode exec

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
- `--raw`: write raw HTTP response to stdout/stderr instead of normalized JSON.
- `--dry-run`: do not execute; print a representation of the request that would be sent.

Errors are written to stderr as:

```json
{ "error": "message" }
```

### Exit codes (from `internal/kernel/errors.go`)

- `0` – Success.
- `1` – Invalid input (bad JSON, missing `tool`, validation error, invalid flags).
- `2` – Tool not found (missing in `tooling.json` or bundle).
- `3` – Auth error (reserved for auth-related exec failures).
- `4` – SDK failure (network/HTTP errors when calling the target API).
- `5` – Internal error (unexpected conditions, project root detection errors, etc).

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
- Installs editor rules:
  - Cursor: `.cursor/rules/swytchcode.mdc`
  - Claude: `CLAUDE.md`

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
```

- Reads only local state:
  - `.swytchcode/integrations` (Wrekenfiles, methods.json, workflows.json).
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
```

- Reads integration bundles and Wrekenfiles from `.swytchcode/integrations`.
- Resolves the requested method/workflow and its STRUCTs into concrete input/output schemas.
- Adds an entry in `tooling.json.tools` for that canonical ID.

If the canonical ID exists in multiple integrations, `add` may require an explicit `project@library.version` to disambiguate.

## info

```bash
swytchcode info <canonical_id>
```

- Shows rich information for a tool:
  - Source integration.
  - Summary/description.
  - Resolved input and output schemas (STRUCTs expanded).
- Uses both `tooling.json` and integration artifacts (`wrekenfile.yaml`, `methods.json`, `workflows.json`) to compute the result.

## MCP commands

- `swytchcode mcp serve`
  - Starts the MCP server (stdio by default, HTTP when configured).
  - Exposes tools that mirror CLI operations (init, get, bootstrap, list, info, exec, etc.).

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

See backend-specific docs for exact payloads and behavior. From the CLI’s perspective, they:

- Resolve project UUIDs and tokens via `internal/auth`.
- Call backend endpoints for account/project/introspection and plan/usage info.
- Exit with non-zero codes on auth/network/server errors, printing clear messages on stderr.

