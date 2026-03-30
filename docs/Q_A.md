# Developer Q&A: Swytchcode CLI Behavior

Based on full codebase exploration of the `discovery` branch.

---

## Q1: Integration downloaded for a specific version, upgrade detected — how do we update integration code?

**A multi-step process:**

1. `swytchcode check` — detects a pending TinyFish proposal (exits code `1` if breaking/major changes; safe as a CI gate)
2. `swytchcode inspect <library>` — shows version diff, impact level, confidence score, and a human-readable summary of what changed in the API
3. `swytchcode diff <library>` — shows method-level signature changes: added/removed methods, new/removed/changed input fields, breaking flag on removals. Use this to assess impact before approving.
4. `swytchcode upgrade <library>` — approves the proposal server-side (requires **user login**, not just a service token — the backend records who approved). Triggers backend reprocessing.
   - With `--apply`: automatically runs `swytchcode get` + re-adds all affected methods after approval.
5. If NOT using `--apply`: manually run `swytchcode get <project> --yes` then `swytchcode add <canonical_id>` for each affected method.

**What's automated with `--apply`:** After `upgrade --apply`, the CLI re-fetches the integration bundle and re-adds all methods for that library in a single step. No `swytchcode migrate` or schema migration exists — tooling.json is updated by re-adding tools.

---

## Q2: A library version is updated in the backend — how will swytchcode shell handle it? How are we notified and code updated?

**No automatic notification exists. All updates are pull-based.**

**Notification path:**
- `swytchcode check [--library=X]` polls `GET /v2/cli/proposals/check` — use this in CI pipelines (exit code `1` on breaking changes)
- `swytchcode inspect <library>` gives human-readable details on what changed

**Code update path** (same as Q1 steps 3–5 above).

**Stale detection (implemented):** At `swytchcode add` time, a `method_hash` (SHA-256 of the wrekenfile entry) is stored per tool in `tooling.json`. `swytchcode sync` re-computes hashes and warns `"⚠ method X has changed — run swytchcode add X to refresh"` for any tool whose hash no longer matches the local wrekenfile. This catches local drift but not backend-side changes (phase 2, requires backend endpoint).

**What's still missing:** No push notifications, no background polling, no backend-driven stale detection yet. `tooling.json` keeps pinning the old version and `exec` keeps using the old wrekenfile until you explicitly run `swytchcode add` again. `swytchcode sync` refreshes workflow definitions and warns on hash mismatches, but does not update method version pins.

---

## Q3: `swytchcode exec` calls a method not in tooling.json and the integration doesn't exist locally — how is it handled?

**Fully handled with structured errors (exit code `2`):**

| Condition | Error message |
|---|---|
| `tooling.json` missing | `tooling.json not found; run 'swytchcode init' first` |
| Tool not in `tooling.json` | `tool "X" not found in tooling.json — run 'swytchcode add X' to add it` |
| Wrekenfile missing on disk | `integration stripe.stripe@v2.1 not installed. Run: swytchcode get stripe` |
| Method not in wrekenfile | `method "X" not found in wrekenfile` |

All errors are written as `{"error": "..."}` JSON to stderr — machine-readable for MCP agents and CI pipelines.

---

## Q4: Someone tries to integrate a workflow/method not existing in a specific version — is there version validation?

**Soft validation exists (warning, not blocking).**

`swytchcode add <canonical_id>` searches all locally installed wrekenfiles for the canonical ID. If found locally, it also calls `GET /v2/cli/integrations/{project}/{library}/{version}/methods` to check whether the method is known in that version on the backend.

- If the method is not in the backend-returned list → prints `"⚠ method X not found in backend version Y — it may not exist or may have been removed"` (non-blocking)
- If the backend endpoint isn't live yet (404/network error) → validation is silently skipped so offline/pre-deployment use still works

There is no semver range checking, deprecation blocking, or compatibility matrix. The primary guard is still a local wrekenfile existence check — version validation is a best-effort warning.

**What's still missing:** `swytchcode exec` at runtime does existence-only checks, not version compatibility checks. No enforcement that the installed version satisfies a declared constraint.

---

## Q5: Discovery lists only certain results — how do I search further with a specific library/integration?

**Current options:**

1. **Scope to a project:**
   ```bash
   swytchcode discover "charge a customer" --project stripe
   ```
   The `--project` flag scopes semantic search to a single project. There is no `--library` scoping flag yet.

2. **Increase result count** (default is 5):
   ```bash
   swytchcode discover "charge a customer" --top 20
   ```

3. **Browse locally installed methods:**
   ```bash
   swytchcode list
   swytchcode list --filter stripe
   ```
   Lists all tools in `tooling.json` (already-added ones only — not a discovery search).

4. **MCP tool `swytchcode_discover`** accepts the same `intent`, `project_name`, `library`, and `top_k` params for AI agent use.

**Library scoping (implemented):**
```bash
swytchcode discover "charge a customer" --library payments
```
The `--library`/`-l` flag scopes semantic search to a specific library within a project. When combined with `--project`, it narrows results to that project's library. The filter is passed as `library_name` in the backend `POST /v2/cli/discover` payload (backend filtering support pending).

**What's still missing:** No way to browse all available methods for a library without running `swytchcode get` first. `DISCOVERY_FUTURE.md` notes a public hosted discovery API for external agents as future work.

---

## Q6: If I don't provide login or `SWYTCHCODE_TOKEN`, will `exec` work?

**Yes — `exec` works without authentication for local single-method execution.**

From `internal/cli/exec.go`:
```go
token, fromSession, _ := auth.ResolveToken()
if token == "" {
    telemetry.MaybeHintNoAuth()
}
```

A missing token only prints a telemetry hint. Execution proceeds regardless.

**However:** If the *downstream API itself* requires authentication (e.g., Stripe needs `Authorization: Bearer <key>`), you must supply it manually via `--header Authorization=Bearer <yourtoken>`. Swytchcode does not inject API credentials automatically.

**Commands that DO require auth:**
- `swytchcode check` and `swytchcode inspect` — need a token or session
- `swytchcode upgrade` — requires a full **user session** (not just a service token) since the backend records the approver's identity
- `swytchcode diff <library>` — requires a token or session (fetches proposal diff from the registry)

---

## Q7: If I upgrade swytchcode CLI, will existing workflows/integrations work?

**Yes — CLI upgrades are backward-compatible with existing local bundles today.**

`tooling.json` stores the CLI version at `init` time and never overwrites it on subsequent commands:
```go
if _, hasVersion := tooling["version"]; !hasVersion {
    tooling["version"] = constants.Version
}
```
The version field is informational only — not validated or enforced at exec time. Local wrekenfiles, `manifest.json`, and `tooling.json` are format-stable.

**Potential risk:** If a future CLI upgrade changes the `tooling.json` schema or wrekenfile parsing in a breaking way, existing integrations could break. There is no schema versioning on wrekenfiles and no `swytchcode migrate` command. This is a gap to watch as the format evolves.

---

## Q8: How to fetch newer workflows for an integration?

**Two commands serve different purposes:**

### `swytchcode sync [project_name]` — for workflow definition changes
- Re-fetches workflow list from the backend, compares against local `workflows.json`
- Re-downloads all integration bundles (wrekenfile.yaml, methods.json, workflows.json)
- Updates `manifest.json`
- Does **NOT** touch `tooling.json` — warns `"run swytchcode add <id> to refresh"` for changed workflows already in tooling

After sync, re-add to pick up changes:
```bash
swytchcode add <new_workflow_canonical_id>   # add a new workflow to tooling.json
swytchcode add <existing_workflow_id>        # re-add a changed workflow to refresh its step schema
```

### `swytchcode get <project> --yes` — for version upgrades
- Full re-download of all bundles for a project (overwrites existing)
- Use this after `swytchcode upgrade` when a library version has changed

---

## Q9: A workflow is independent of library_uuids (multi-lib) — why is it shown with libraries?

**This is a known UX tension — the "owner library" is a simplification.**

Every workflow must have an owner project/library in the data model for organizational purposes. `swytchcode add` records `"integration": "project.library@version"` as the owner in `tooling.json`, and discovery results return a `library` field pointing to the owning library.

**The architectural reality:** Multi-library workflows have steps from different libraries, each routed via `library_uuid` per step at execution time. The workflow is not actually bound to a single library — but it appears under one for navigational purposes. A workflow labeled under `stripe` might also call `sendgrid` steps internally. The `steps[]` array in `tooling.json` shows the real per-step `integration` breakdown.

`DISCOVERY_FUTURE.md` mentions a "capability graph" as future work that would better represent multi-library relationships. Today, the owner-library label is a pragmatic simplification.

---

## Q10: If a method/workflow is renamed/edited in the backend, will it impact existing integrations via `swytchcode exec`?

**Depends on whether it's a method or a workflow:**

### Single-method execution — NOT automatically impacted
- `exec` reads only local files: `tooling.json` + the pinned wrekenfile version. Backend changes are invisible until you re-run `swytchcode get` and `swytchcode add`.
- If the canonical ID was **renamed** on the backend, the old ID continues working locally until you manually remove it.

### Workflow execution — partially impacted immediately
- `exec <workflow_id>` calls `GET /workflows/{canonical_id}` from the registry **at runtime** (workflows are always fetched live).
- If the canonical ID was **renamed**, exec fails immediately with a registry 404.
- If only step **definitions changed** (same canonical_id), the updated steps are used immediately on next exec.

### What's implemented
- **`method_hash`**: stored in `tooling.json` at `add` time (SHA-256 of the wrekenfile method entry). `swytchcode sync` re-computes hashes and warns `"⚠ method X has changed — run swytchcode add X to refresh"` for any mismatch. This catches local drift (e.g. if a wrekenfile is manually edited or re-fetched from a different version).

### What's still missing
- No rename tracking or redirect mechanism — a renamed canonical_id breaks existing exec calls with no helpful error pointing to the new name. (`GET /v2/cli/canonical-id/resolve` endpoint contract is documented in `BACKEND_API_CONTRACTS.md`; CLI degrades gracefully when backend endpoint isn't live yet.)
- No backend-driven stale detection — hash comparison is local-only today. Phase 2 (`GET /v2/cli/methods/hashes`) will compare against backend hashes when that endpoint is live.
- `swytchcode sync` warns on workflow step changes and method hash mismatches but does not update method version pins.

---

## Q11: Can tool execution use `http://` (non-HTTPS) endpoints? What about localhost, CI, and Docker?

**Execution base URLs** (`sandbox_endpoint` / `production_endpoint` in `manifest.json`) are validated by [`ValidateExecutionBaseURL`](internal/kernel/base_url_validate.go):

| URL | Allowed? |
|-----|----------|
| `https://` on any host | Yes |
| `http://` on `localhost`, `127.0.0.1`, or `::1` | Yes |
| `http://` on any other hostname or IP | No |

**CI (GitHub Actions, GitLab CI, etc.)** and **Docker** use the same rules. HTTPS to a reachable API works as usual. `http://127.0.0.1` is valid when a mock or API listens on loopback **in the same container/job** as the CLI. `http://localhost` inside a container is that container’s loopback, not the host or another Compose service — use HTTPS (or a loopback proxy) for those.

**`SWYTCHCODE_INSECURE=1`:** Skips TLS certificate verification only; it does **not** bypass the scheme/host rules above. Registry traffic is **blocked** in CI when `CI` / `GITHUB_ACTIONS` / `GITLAB_CI` is truthy and `SWYTCHCODE_INSECURE=1` is set.

See [docs/config-spec.md](config-spec.md) (manifest section) and the README “Base URL Resolution”.

---

## Q12: How do I sanity-check my project before CI or after clone?

Run **`swytchcode doctor`** (or MCP **`swytchcode_doctor`** with `json: true`). It verifies `tooling.json`, installed bundles, `manifest.json`, HTTPS/loopback rules on manifest endpoints, auth posture, and `SWYTCHCODE_INSECURE` vs CI. Exit code **1** if any **error**-level check fails (warnings alone do not fail). See [docs/cli-reference.md](cli-reference.md) and [docs/security.md](security.md).
