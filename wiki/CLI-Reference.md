# CLI Reference

Full command list and behavior. For a short table, see [Pages → CLI](https://swytchcode.gitlab.io/cli/docs/cli.html).

## Commands

| Command | Purpose |
|--------|---------|
| `swytchcode -v` / `--version` | Show version |
| `swytchcode init` | One-time setup: `.swytchcode/`, `tooling.json`, editor rules (Cursor / Claude) |
| `swytchcode get <project>` | Fetch integration bundle; does not add to tooling.json |
| `swytchcode bootstrap` | Fetch all integrations in tooling.json (CI-friendly) |
| `swytchcode list` | List tools and integrations from tooling.json + local integrations |
| `swytchcode list methods [pattern]` | List methods (optional filter by canonical_id / project) |
| `swytchcode list workflows [pattern]` | List workflows (optional filter) |
| `swytchcode list integrations` | List fetched integrations only |
| `swytchcode search [keyword]` | Search remote registry |
| `swytchcode add [spec] <canonical_id>` | Add a tool to tooling.json |
| `swytchcode info <canonical_id>` | Show tool info (resolved inputs/output) |
| `swytchcode exec [canonical_id]` | **Single execution path** – CLI args or JSON stdin. `--demo` flag (or shorthand `swytchcode <canonical_id>`) runs without setup. |
| `swytchcode demo list` | List all tools that have a demo available |
| `swytchcode doctor` | Local diagnostics; `--json`; exits 1 on error-level checks |
| `swytchcode mcp serve` | Start MCP server (stdio/HTTP) |
| `swytchcode mcp status` / `mcp stop` | Daemon status / stop |

## swytchcode exec

- **Input:** CLI args or JSON on stdin: `{"tool":"<canonical_id>","args":{...}}`.
- **Output:** JSON by default; `--raw` for raw stdout/stderr; `--dry-run` for no execution.
- **Exit codes:** Non-zero on resolution failure (tool not found, integration not installed) or tool execution failure. Exact codes are implementation-defined; non-zero means failure.
- **Flags:** `--json`, `--raw`, `--dry-run`, `--demo`, `--allow-raw`, body/header/param passthrough as needed.
- **Base URL:** Resolved from `manifest.json` (`sandbox_endpoint` / `production_endpoint`). Must be **`https://`** or **`http://`** on loopback only (`localhost`, `127.0.0.1`, `::1`). Same in CI/Docker. See [docs/config-spec.md](https://gitlab.com/swytchcode/cli/-/blob/main/docs/config-spec.md) and repo README.

Document exact exit codes and JSON request/response schema in this page as you stabilize them (e.g. from `internal/kernel/errors.go` and executor).
