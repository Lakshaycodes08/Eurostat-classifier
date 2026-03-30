# Troubleshooting

Quick-reference for diagnosing `swytchcode exec` failures. For detailed debug techniques, see [debugging.md](debugging.md). For auth-specific issues, see [auth-guide.md](auth-guide.md).

## Exit code reference

| Code | Constant | Category | Retryable | Typical cause |
|---|---|---|---|---|
| 0 | `ExitCodeOK` | — | — | Success |
| 1 | `ExitCodeInvalidInput` | `validation` or `internal` | No | Malformed JSON, missing required field, schema validation failure |
| 2 | `ExitCodeToolNotFound` | `not_found` | No | Canonical ID not in `tooling.json`; bundle or method missing from disk |
| 3 | `ExitCodeAuthError` | `auth` | No | Token rejected by the API (HTTP 401/403) |
| 4 | `ExitCodeSDKFailure` | `network` or `rate_limit` | Usually yes | Network error, timeout, rate limiting, HTTP 5xx from API |
| 5 | `ExitCodeInternalError` | `internal` | No | Project root not found, JSON encoding failure |

Errors are written to stderr as JSON: `{ "error": "...", "category": "...", "retryable": true|false }`.

---

## By category

### `auth` — exit 3

The API rejected the credentials.

- **Token missing** — the `Authorization` header was not passed in args. Run `swytchcode info <canonical_id>` to see the required header.
- **Wrong format** — `Bearer` prefix missing, or token includes/excludes prefix incorrectly.
- **Token expired** — Firebase ID tokens expire after ~1 hour. Run `swytchcode login` to refresh, or use a `swy_key_*` service token for long-lived sessions.
- **Wrong key for mode** — sandbox key used against a production endpoint or vice versa. Check `mode` in `.swytchcode/tooling.json`.
- **Key revoked** — check the service dashboard.

See [auth-guide.md](auth-guide.md) for full credential patterns.

---

### `validation` — exit 1

Input was rejected before the HTTP call.

- **Missing required field** — run `swytchcode info <canonical_id>` to see which `inputs` are required.
- **Wrong type** — e.g. string where integer expected.
- **Malformed JSON in `--body`** — parse error in your payload. On Windows, `&` characters in JSON must be escaped or passed via a file (see [windows-guide.md](windows-guide.md)).
- **Tool not specified** — `tool` field is empty. Specify the canonical ID.
- **`--allow-raw` missing** — raw method execution requires the `--allow-raw` flag.

---

### `not_found` — exit 2

The tool, bundle, or method could not be found locally.

- **Canonical ID not in `tooling.json`** — run `swytchcode add <canonical_id>` to add it, then retry.
- **Bundle not downloaded** — run `swytchcode get <project>` to fetch the integration bundle.
- **Method name changed** — run `swytchcode check` to see if a breaking upgrade is pending; run `swytchcode diff <library>` to preview changes.
- **Workflow step missing** — if a workflow step references an unknown library, re-run `swytchcode get` to refresh all bundles.

---

### `network` — exit 4 (retryable)

The HTTP call failed due to a transient network issue.

- **Endpoint unreachable** — check internet connectivity; the API may be down.
- **Timeout** — the request exceeded the execution policy timeout. Run `swytchcode inspect <library>` to see the configured timeout.
- **TLS/cert error** — check that system root CAs are up to date.
- **Connection reset / EOF** — often transient; retry with back-off.

`retryable: true` — safe to retry after a brief wait.

---

### `rate_limit` — exit 4 (retryable)

The API returned HTTP 429 (Too Many Requests).

- **Back off and retry** — the error message usually includes a `Retry-After` duration.
- **Check API plan limits** — you may have exceeded your tier's rate limit.
- **Batch requests** — if running many sequential calls, add a delay between them.

`retryable: true`.

---

### `internal` — exit 5

A problem with the local setup, not the API.

- **Project root not found** — run `swytchcode init` to create `.swytchcode/` in the project root.
- **Corrupted bundle** — delete `.swytchcode/integrations/<library>/` and re-run `swytchcode get`.
- **`tooling.json` malformed** — validate the JSON; run `swytchcode doctor` to diagnose.
- **Execution policy error** — malformed `manifest.json`. Re-fetch with `swytchcode get`.

---

## Canonical ID suffix (`_xxxx`)

Canonical IDs sometimes end in a `_` followed by 4 hex characters (e.g. `stripe.create_checkout_session_ec30`). This is a **disambiguation hash** added when multiple methods share the same base name within a library.

- **Always use the full canonical ID** including the suffix in `swytchcode exec`, `swytchcode add`, and generated code.
- The API response includes `canonical_id_has_suffix: true` and `canonical_id_base` to help you identify the human-readable base name.
- Do not strip the suffix — it is part of the stable identifier.

---

## Windows-specific issues

See [windows-guide.md](windows-guide.md) for full details. Common problems:

- **`&` in JSON payload** — use `--body <file>` to pass the payload from a file instead of inline shell string.
- **Binary output to terminal** — always use `--output <file>` when the response may be binary (PDF, image).
- **Path separators** — use forward slashes in file paths even on Windows, or quote paths containing backslashes.

---

## `swytchcode doctor`

Run `swytchcode doctor` (or MCP `swytchcode_doctor`) to get a checklist of common setup issues: missing `tooling.json`, unauthenticated session, stale bundles, broken registry connectivity.

```bash
swytchcode doctor
# or for machine-readable output:
swytchcode doctor --json
```
