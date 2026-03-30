# Debugging Swytchcode Executions

This guide covers the debug flags and structured output available from `swytchcode exec`. For a full flag reference, see [cli-reference.md](cli-reference.md). For error resolution, see [troubleshooting.md](troubleshooting.md).

## Flags

### `--dry-run`

Prints the HTTP request that would be sent — method, URL, headers, body — without making the call. Useful to verify payload shape before hitting a live API.

```bash
swytchcode exec stripe.checkout_session_create \
  --body request.json --dry-run
```

Output goes to stdout as JSON. No HTTP call is made and no exit-code other than 0 is returned (unless the tool or bundle cannot be resolved).

MCP equivalent: `swytchcode_exec` with `dry_run: true`.

---

### `--verbose`

Emits two JSON lines to **stderr**, one before and one after the HTTP call:

```json
{"verbose":"request","method":"POST","url":"https://api.stripe.com/v1/checkout/sessions","headers":{"Authorization":"[REDACTED]","Content-Type":"application/x-www-form-urlencoded"}}
{"verbose":"response","status":200,"content_type":"application/json","body_bytes":1240}
```

Sensitive header values (`Authorization`, `X-Api-Key`, `token`, `secret`, `password`) are automatically redacted.

To capture verbose output without mixing with the result:

```bash
swytchcode exec stripe.checkout_session_create \
  --body request.json --verbose 2>debug.log
cat debug.log
```

MCP equivalent: `swytchcode_exec` with `verbose: true`. The verbose lines appear in the tool's stderr capture and are included in the combined output returned to the MCP client.

---

### `--output <file>`

When the API returns a binary `Content-Type` (e.g. `application/pdf`, `image/png`), writes the raw response bytes to `<file>` instead of stdout. Stdout receives a JSON summary:

```json
{ "saved_to": "invoice.pdf", "bytes": 42301, "content_type": "application/pdf" }
```

If a binary response arrives and `--output` is not set, the CLI exits with an error rather than writing binary data to the terminal.

MCP equivalent: not exposed — binary output over MCP is returned as base64 in the tool result.

---

## Structured errors

All errors from `swytchcode exec` are written to **stderr** as JSON:

```json
{ "error": "execution failed: 429 Too Many Requests", "category": "rate_limit", "retryable": true }
```

| Field | Values | Notes |
|---|---|---|
| `error` | string | Human-readable error message |
| `category` | `auth` \| `validation` \| `not_found` \| `network` \| `rate_limit` \| `internal` | Machine-readable category |
| `retryable` | `true` / `false` | `true` = transient, safe to retry |

---

## Exit codes

| Code | Constant | Typical cause |
|---|---|---|
| 0 | `ExitCodeOK` | Success |
| 1 | `ExitCodeInvalidInput` | Malformed JSON, missing required field, validation failure |
| 2 | `ExitCodeToolNotFound` | Canonical ID not in `tooling.json`, bundle missing |
| 3 | `ExitCodeAuthError` | Auth token rejected by the API (HTTP 401/403) |
| 4 | `ExitCodeSDKFailure` | HTTP call failed (network error, rate limit, API returned error) |
| 5 | `ExitCodeInternalError` | Project root not found, JSON encoding failure |

Exit codes are stable and part of the public contract. They map to the `category` field in the structured error JSON, but the relationship is not always 1:1 (e.g. exit 4 can have category `network` or `rate_limit`). Always read `category` from stderr JSON for reliable categorisation.

---

## Common debug workflow

1. **Verify the payload first** — run with `--dry-run` and confirm method, URL, headers, and body are correct before making a live call.
2. **Check the auth header** — in the dry-run output look for `Authorization` (shown as `[REDACTED]`). If the header is absent, your args are missing the auth field. Run `swytchcode info <canonical_id>` to see the expected auth header name.
3. **Run verbose** — if the call reaches the API but returns an unexpected response, use `--verbose` to see the response status and content-type, then inspect `debug.log`.
4. **Check the exit code and structured error** — read `category` and `retryable` from stderr JSON. `retryable: true` → wait and retry. `retryable: false` → fix the input or auth before retrying.
5. **Check for updates** — if the API returns 404 or the schema seems wrong, run `swytchcode check` to see if a breaking integration change has been detected.

---

## MCP debugging

When using `swytchcode_exec` from an IDE agent:

1. Set `dry_run: true` first to inspect the planned request.
2. If the call fails, the tool result `isError` will be `true` and the content will contain the stderr JSON. Parse the `category` field to decide how to respond:
   - `auth` — inform the user to check/update their API key in `.env`.
   - `validation` — fix the args: missing field, wrong type, or the canonical ID is not in `tooling.json`.
   - `not_found` — run `swytchcode_add` to add the canonical ID before retrying.
   - `network` / `rate_limit` — these are retryable; back off and retry.
   - `internal` — run `swytchcode_doctor` to diagnose the project setup.
3. Set `verbose: true` to capture request/response details for harder-to-diagnose failures.
