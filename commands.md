# CLI Verification Commands

This file is an internal checklist for verifying the **real** Swytchcode CLI against a backend environment (local or staging). For full user-facing documentation, see the markdown files under `docs/` (architecture, execution model, config spec, CLI reference, MCP, and install/upgrade).

---

## 1. Build

```bash
cd /path/to/swytchcode/shell
go build -o swytchcode ./cmd/swytchcode/
```

Confirm binary exists:
```bash
./swytchcode --version
```

---

## 2. Auth modes

| Mode | How | Who |
|------|-----|-----|
| **Service token** | `SWYTCHCODE_TOKEN` env var → `Authorization: Bearer <token>` | Agents, CI/CD |
| **User login** | Firebase JWT in `~/.swytchcode/auth.json` via `swytchcode login` | Human developers |

`check` accepts either. `inspect` uses service token for the first request (proposals list) but
requires user login for the second request (proposal detail via `appAuthMiddleware`). `upgrade`
requires user login only.

### Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success — no breaking proposals / command completed normally |
| `1` | Breaking (`major`) proposals detected — use as CI gate |
| `2` | CLI error — auth failure, network error, exec limit (429), missing env var |

---

## 3. Service token setup (agents / CI)

```bash
export SWYTCHCODE_API_URL=http://localhost:80
export SWYTCHCODE_TOKEN=<value of INTERNAL_AGENT_TOKEN from backend .env>
export SWYTCHCODE_PROJECT_UUID=<your project UUID>
```

---

## 4. User login flow (human developers)

```bash
./swytchcode login
# Visit the URL printed, complete login in browser
# CLI saves session to ~/.swytchcode/auth.json automatically
```

Verify session:
```bash
./swytchcode whoami
```

Expected output:
```
email:         user@example.com
customer_uuid: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
session:       valid (expires in 59 minutes)
```

Log out:
```bash
./swytchcode logout
# → Logged out.
./swytchcode whoami
# → Not logged in.
```

Auto-open browser:
```bash
./swytchcode login --open
```

### Token auto-refresh

If `~/.swytchcode/auth.json` contains a `refresh_token`, the CLI automatically refreshes
the access token before it expires (60-second safety buffer). No user action required.

If refresh fails (backend returns 400/401), the session file is deleted and the CLI
prompts re-login:
```
session expired — run `swytchcode login` again
```

---

## 5. Verify backend is reachable

```bash
curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer $SWYTCHCODE_TOKEN" \
  "$SWYTCHCODE_API_URL/v2/cli/proposals/check?project_uuid=$SWYTCHCODE_PROJECT_UUID"
```

Expected: `200`
If `401`: token is wrong or not set.
If `connection refused`: backend is not running (`docker-compose ps`).

---

## 6. check — clean state

Before any proposals exist:

```bash
./swytchcode check
echo "exit code: $?"
```

Expected output:
```
All integrations up to date
exit code: 0
```

---

## 7. check — with a major proposal

After TinyFish scan has run (or after seeding the DB manually — see `demo_todo.md` Step 4):

```bash
./swytchcode check
echo "exit code: $?"
```

Expected output:
```
[!] stripe       v1.0.0 -> v2.0.0   major    Breaking changes — new auth flow required
exit code: 1
```

Non-major proposals print without color and do not set exit code 1:
```
[!] twilio       v3.1.0 -> v3.2.0   minor    Added new messaging endpoints
exit code: 0
```

---

## 8. check — missing project UUID

```bash
unset SWYTCHCODE_PROJECT_UUID
./swytchcode check
echo "exit code: $?"
```

Expected output:
```
Error: SWYTCHCODE_PROJECT_UUID is not set
exit code: 2
```

Re-export after test:
```bash
export SWYTCHCODE_PROJECT_UUID=<your project UUID>
```

---

## 9. inspect

Show full detail for a single library's proposal. **Requires user login** (not a service token —
the second request hits `appAuthMiddleware`).

```bash
./swytchcode inspect stripe
```

Internally makes two requests:
1. `GET /v2/cli/proposals/check` — resolves library name to proposal UUID
2. `GET /v2/app/proposals/:uuid` — fetches full detail (Firebase JWT only)

Expected output (when a proposal exists):
```
stripe   v1.0.0 -> v2.0.0   [major]   confidence: 0.95
────────────────────────────────────────────────────────
Summary:  Breaking changes in auth flow — createClient() removed
```

When no proposal exists:
```
No proposals found for stripe
```

If not logged in:
```
Error: not logged in — run `swytchcode login`
exit code: 2
```

---

## 10. check — exec limit (429)

When the project has exceeded its monthly CLI execution quota:

```bash
./swytchcode check
echo "exit code: $?"
```

Expected output:
```
Error: monthly CLI executions used: 501 / 500 — upgrade your plan: https://app.swytchcode.com/billing
exit code: 2
```

Exit code `2` signals a CLI-level error (not a proposal finding) — CI pipelines should
treat this as a hard failure and surface it for human review.

---

## 11. upgrade

Approve a pending proposal (requires user login — not a service token):

```bash
./swytchcode upgrade stripe
```

Expected prompt for a major change:
```
Approve stripe v1.0.0 → v2.0.0 [major]? This is a BREAKING change. (y/N)
```

Type `y` to confirm:
```
Upgrade submitted for stripe — spec fetch and library reprocessing started
```

If not logged in:
```
Error: not logged in — run `swytchcode login` (service tokens cannot approve upgrades)
```

---

## 12. Watch backend logs during a scan

In a separate terminal tab, tail the AI module to see TinyFish activity:

```bash
docker-compose logs -f swytchcode-ai
```

Expected lines when a proposal is created:
```
[TinyFish] Agent reviewing: library=<uuid> impact=major confidence=0.90
[TinyFish] Agent decision: approve — integrating TinyFish-detected breaking change
[TinyFish] Proposal created: proposal_uuid=<uuid> library=<uuid> status=applied
```

---

## 13. Full demo dry-run sequence

Run these in order with a timer. Target: under 2:40.

```bash
# 1. Show pinned integration (local list)
./swytchcode list integrations

# 2. Check for updates (should be clean before v2 release exists)
./swytchcode check

# 3. (Switch to browser — show GitHub releases page with v2.0.0 already published)
# (Switch to terminal with docker logs tailing — wait for TinyFish scan to fire)

# 4. After scan completes, run check again
./swytchcode check

# 5. Inspect the breaking proposal
./swytchcode inspect stripe

# 6. Approve it (requires user login)
./swytchcode upgrade stripe
```

---

## 14. Clear stale proposals (reset between test runs)

If you need to reset the demo state, run this SQL against the dev database:

```sql
DELETE FROM integration_proposals
WHERE project_uuid = 'YOUR_PROJECT_UUID';
```

Then re-run the scan manually:
```bash
docker exec swytchcode-ai-service python3 -c \
  "import asyncio; from ai.tinyfish.scanner import run_full_scan; asyncio.run(run_full_scan())"
```

---

## 15. Rebuild after any Go changes

```bash
go build -o swytchcode ./cmd/swytchcode/ && echo "build ok"
```

After a rebuild, telemetry events will appear in backend logs (`POST /v2/cli/telemetry/batch`)
when you run `check`, `inspect`, or `upgrade`. No output is shown in the CLI terminal — failures
are silently discarded.
