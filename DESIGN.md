<!-- Authoritative design document: architectural decisions, invariants, and boundaries for the Swytchcode kernel. -->

# Swytchcode Design Document

This document captures the architectural decisions and design principles for Swytchcode. Treat this as the authoritative source of truth for design discussions.

**Last updated:** Reflects integration version pinning, bootstrap, config, and registry-usage invariants.

---

## Core Philosophy

**Swytchcode is not a library. It is a kernel.**

Swytchcode is a deterministic execution layer between LLMs/apps and real-world SDKs/APIs — not an agent framework, not observability tooling.

**"LLMs ask. Swytchcode executes."**

All languages, editors, and agents are **guests** that must call `swytchcode exec` instead of doing their own SDK logic.

---

## Fundamental Principles

### 1. Single Execution Path

- **`swytchcode exec`** is the **only** command that application code, agents (Cursor, Copilot, etc.), or automations are allowed to call for execution.
- All other commands (`list`, `describe`, `plan`, `dry-run`, `explain`) are **diagnostic or setup only** — for humans debugging, inspecting, or learning. **Agents must never rely on them for integration.**
- App code does not orchestrate retries, policies, tool selection, or fallbacks. Agents don't "reason about tools" — they execute via `exec`.

### 2. Deterministic Execution

- Same JSON input → same JSON output, no hidden state.
- No prompts, no TTY detection, no prose during `exec`.
- Stable exit codes (locked contract: 0=Success, 1=Invalid input, 2=Tool not found, 3=Auth error, 4=SDK failure, 5=Internal error).

### 3. Editor-Agnostic Runtime

- Editor rules exist only for authoring (e.g. `.cursor/rules/swytchcode.mdc`).
- `exec` must not depend on Cursor / VS Code configs.
- Editor configs affect IDEs only; runtime ignores them.

### 4. CI/Docker First

- Non-interactive by default.
- Environment-variable based auth only.
- Works without TTY, without `$HOME`, without editor.
- If Swytchcode works in CI, it will work in production. If it only works in an IDE, it is broken.

---

## Architecture Boundaries

### tooling.json vs Wrekenfiles

Swytchcode intentionally separates **execution contracts** from **execution implementations**.

#### tooling.json (authoritative contract)

- Defines **what tools are allowed** in this project.
- Defines the **canonical input/output schema** for each tool.
- Stores the **execution mode** (`production` or `sandbox`).
- Stores **integration version pins** (`integrations: { <name>: { version: "<exact>" } }`). No ranges, no `"latest"`. The registry decides what versions exist; the project decides which versions it trusts.
- Stores **registry_url** (base URL for the registry API). Set by init only when absent; overridable at runtime by `SWYTCHCODE_REGISTRY_URL` (env wins; override is never persisted).
- Stores **version** (tooling.json schema version only; kernel-owned, set once by init).
- Is **committed**, reviewed, and stable.
- Is the **only agent-facing contract**.

**Rule:** If it affects *what a tool is allowed to do*, it belongs in `tooling.json`.

**Kernel-owned fields:** `version`, `mode`, and `registry_url` are owned by the kernel. Proposals must not include them; apply rejects any proposal that does. Init never overwrites an existing `version` or `registry_url`.

#### Wrekenfiles (implementation details)

- Define **how an SDK method is executed**.
- Encode SDK calls, auth mapping, retries, and metadata.
- Are **library-specific** and may change independently.
- Must **conform to** the I/O contract defined in `tooling.json`.

**Rule:** If it affects *how the SDK is called*, it belongs in a Wrekenfile.

**Critical boundary:** Wrekenfiles must never define public I/O schemas. The kernel enforces:
- Inputs are validated against `tooling.json`.
- SDK outputs are normalized to the shape declared in `tooling.json`.
- Extra fields are dropped; missing required fields cause failure.

#### Installed versions (wreken manifest)

- **Path:** `.swytchcode/wrekenfiles/manifest.json`.
- **Purpose:** Records which integration version is installed for each library (e.g. `{"stripe": "2025-01-10"}`).
- **Updated by:** `get` and `upgrade` when they write a Wrekenfile; `bootstrap` when it installs.
- **Used by:** `bootstrap` to decide “fetch” vs “version mismatch → fail”. Enables deterministic, reproducible installs.

---

## Registry: Who Uses It, Who Does Not

**Invariant: `swytchcode exec` must never call the registry.** CI determinism, offline execution, and security boundaries depend on it.

| Command            | Uses registry |
|--------------------|---------------|
| `get`              | ✅            |
| `upgrade`          | ✅            |
| `list`             | ❌ (local only) |
| `describe`         | ❌            |
| `add workflow`     | ✅            |
| `add integration`  | ❌ (writes tooling.json only) |
| `apply`            | ❌            |
| **`exec`**         | **❌ never**  |
| **`bootstrap`**    | ✅ (exact versions only) |

**Rule:** If exec ever needs registry access, the architecture has broken.

---

## Integration Version Pinning (Determinism)

### No implicit “latest”

- **No “latest” at exec or bootstrap time. Ever.** Implicit “current” versions would destroy CI reproducibility, auditability, and rollbacks.
- Versions are **exact and explicit** in `tooling.json` (e.g. `"2025-01-10"`). No semver ranges, no operators.

### How versions enter tooling.json (explicit only)

1. **`swytchcode add integration <name>@<version>`** — Pins the integration version. Does **not** fetch; reviewable and commit-worthy.
2. **`swytchcode apply`** — If a proposal includes an `integrations` block, apply merges it into tooling.json. **Every integration in the proposal must have an explicit `version`**; otherwise apply fails.
3. **Manual edit** — Always allowed.

### Bootstrap

- **Command:** `swytchcode bootstrap`.
- **Behavior:** Reads `integrations` from tooling.json. For each integration: if not installed → fetch **exact version** from registry and write Wrekenfile + manifest; if installed but version mismatch → **fail** (no silent upgrade).
- **Never** installs “latest”. **Never** mutates tooling.json. **Never** runs during exec.
- **Invariant:** *tooling.json pins what is trusted. The registry supplies how it works. `bootstrap` reconciles the two. `exec` only executes.*

### get vs bootstrap

- **`get`:** For exploration and discovery. Fetches latest available, installs locally, updates manifest. **Does not** modify tooling.json. Not authoritative for version pinning.
- **`bootstrap`:** For deterministic install. Uses only versions declared in tooling.json; fails on mismatch. Authoritative for “what is installed” in CI and production.

---

## Config and Env Overrides

### Registry base URL

- **Stored in:** `tooling.json` as `registry_url`.
- **Override:** `SWYTCHCODE_REGISTRY_URL` overrides at runtime when set.
- **Rule:** Environment variable overrides **must not be silently persisted**. The CLI never rewrites `tooling.json` with the env value.

### Visibility of overrides

- **`swytchcode config`** — Outputs effective configuration as JSON. For `registry_url`, shows `effective` (URL in use) and `source` (`"env"` | `"tooling"` | `"default"`). Ensures overrides are visible and avoids “why is this pointing somewhere else?” incidents.

---

## Verified Tools vs Raw Methods

Swytchcode intentionally exposes two execution surfaces.

### Verified Tools (tooling.json)

**Properties:**
- Reviewed and intentional
- Stable input/output contracts
- Safe for CI, production, and agents
- Suggested by editors and agents by default

`tooling.json` defines what is **trusted**.

**May include:**
- Single SDK methods
- Curated SDK methods
- Verified multi-step workflows (explicitly declared)

### Raw Methods (Wrekenfiles)

**Properties:**
- Fully executable
- Discoverable
- Not guaranteed stable
- Not allowed in CI or agents by default

Wrekenfiles define what is **possible**.

**Intended for:**
- Exploration
- Advanced users
- Rapid iteration before promotion

**Execution:** Always **explicitly opt-in** via `raw.*` namespace + `--allow-raw` flag.

**Critical rule:** There must be no silent fallback from verified → raw.

**CI rule:** Raw methods are not allowed in CI by default. In CI, `--allow-raw` must never be used. If `--allow-raw` is present in a CI context, `exec` must fail unless an explicit override is provided (e.g. `--ci-allow-raw`, default false). Even if that override is never implemented, the intent is clear: raw ≠ CI; no accidental enabling via copied commands.

---

## Thin Client Model

### How Projects Call Swytchcode

Other projects **never** integrate Swytchcode logic; they **shell out** to `swytchcode exec` through a **thin client** that looks native to their language.

**Canonical path:**
```
App Code → Thin Client (serialize → invoke CLI → deserialize) 
         → swytchcode exec 
         → tooling.json + Wrekenfile 
         → real SDKs/APIs
```

There is no alternate path.

### Thin Client Responsibilities

**Does only:**
- Serialize input to JSON
- Invoke `swytchcode exec`
- Read stdout/exit code
- Deserialize output

**Does not:**
- Decide which tool to call
- Retry
- Interpret failures
- Apply policy
- Chain calls

**Target:** Thin client **<50 LOC** per language.

---

## Installation Model

### CLI First (Recommended)

- The **Swytchcode CLI** and **thin clients** are **separately installed**.
- CLI is primary; users install the CLI explicitly (e.g. `curl -fsSL ... | sh`).
- Thin clients assume `swytchcode` is on PATH. If CLI is missing, thin client fails with a clear error.

**Why:** One kernel per machine, one upgrade surface, auditability. Prevents version fragmentation and hidden binaries.

**Do not** default to thin clients auto-installing the CLI. Optional "install CLI from client" can be layered on later as opt-in.

---

## Same Kernel Everywhere

- **One binary** for local dev, GitHub Actions, Docker, and production.
- No "CI mode" or "dev mode" binary.
- Behavior is determined by **TTY + flags only** (e.g. `util.IsInteractive()`).
- No environment guessing (e.g. `CI=true`).

**Rule:** In CI there is no TTY, so `init`/`get` require flags; `exec` is never interactive anywhere.

---

## Method Naming

### Canonical IDs (for execution)

- **Format:** lowercase, dot-separated, stable (e.g. `stripe.createCustomer`, `openai.responses.create`).
- **Constraints:** No spaces, no punctuation except `.` and `_`.
- **Used in:** `exec`, CI, thin clients, agents.

### Titles/Descriptions (metadata only)

- **Purpose:** Discovery and editors.
- **Never:** Parsed, matched, or executed.
- **Rule:** Titles can change; IDs must not.

---

## Promotion and Proposals

### Promotion Flow

**Promotion into `tooling.json` is always explicit and human-controlled.** No auto-promotion.

**Lifecycle:**
1. Method exists in a Wrekenfile
2. User discovers it via `swytchcode list`
3. User experiments locally using raw execution (`raw.*` + `--allow-raw`)
4. Team promotes it into `tooling.json` (IDE generates proposal → `validate` → `apply`)
5. Method becomes verified, agent-safe, and CI-safe

### Validate → Apply

**IDEs generate proposals; only the kernel validates and applies.**

1. **Proposal:** IDE (or user) writes a file to `.swytchcode/proposals/<name>.json` (no CLI command for generation).

2. **Validate:** `swytchcode validate <proposal>`
   - Proof — no side effects. Full validation (structure, kernel-owned fields, integrations with version).
   - Produces structured errors or success.

3. **Review:** Human or CI reviews (schema, naming, auth, no forbidden patterns).

4. **Apply:** `swytchcode apply <proposal>`
   - **Authorization** — only command that mutates `tooling.json` from proposals.
   - Fails if proposal is invalid (same checks as validate). Merges **tools** and optional **integrations** (with explicit versions). Archives to `proposals/applied/`.

**IDE-first:** Proposal generation is IDE-owned (IDEs generate proposal files). The kernel owns **validate** and **apply**.

**Invariant:** `tooling.json` is write-protected; all changes go through validation and explicit apply (or explicit `add integration` / `add workflow`). Agents/IDEs prepare; humans approve; kernel enforces.

**Backend boundary:** The backend never receives, stores, validates, or applies proposal files. Proposal files are local, project-scoped artifacts.

---

## Verified vs Custom Workflows

### Verified Workflows

- **Source:** Backend catalog, curated, versioned, reviewed.
- **Application:** May be applied directly (e.g. `swytchcode add workflow stripe.customer-onboarding`).
- **Trust:** Trusted; no proposal step.

### Custom Workflows

- **Definition:** User-assembled, agent-assembled, or backend-generated but not in verified catalog.
- **Application:** **Never auto-applied**. Must be written as **proposals** (e.g. `.swytchcode/proposals/workflow.<name>.json`).
- **Rule:** Custom workflows are always proposed, never applied automatically — even if machine-generated.

**Note:** Verified workflows may exist as first-class tools in `tooling.json`, with explicit I/O contracts and transparent kernel implementation. The "no workflows" restriction applies to Wrekenfiles only.

---

## IDE: Accepting Proposals

### UX Pattern: Show → Explain → Require Explicit Action

**Not:** Modal popups, silent acceptance, "approve?" yes/no prompts.

**Required:**
1. **Show a diff** (what would change in `tooling.json`)
   - Workflow name, description
   - Input/output schema
   - Side effects (write ops, auth scope)

2. **Explicit acceptance action**
   - Button: "Apply workflow" or command: "Swytchcode: Apply Proposal"
   - Runs `swytchcode apply proposals/workflow.<name>.json`
   - No magic, no hidden state

3. **Record acceptance**
   - `tooling.json` updated
   - Proposal file deleted or archived
   - Change is git-diffable
   - CI will now allow it

**Agents may:** Generate proposals, explain proposals, point to risks, ask user to approve.

**Agents may not:** Apply proposals, bypass review, modify `tooling.json` directly.

---

## Authentication Model

### Core Principle

**Authentication is configured explicitly and consumed deterministically. `exec` never initiates authentication.**

### Local Dev vs CI/Production

- **Local development:** Optional OAuth login (e.g. `swytchcode auth login`) may store tokens in a local auth store. This is **optional and local-dev only**; the kernel’s core execution path does not depend on it.
- **CI and production:** Authentication is **env-only**. No local auth store; OAuth login must never be used in CI.
- **Exec:** Never initiates authentication in any environment.

### Auth Sources

- **All secrets from environment variables.** Wrekenfiles declare which env var(s) each operation needs (e.g. `STRIPE_API_KEY`, `GITHUB_TOKEN`). Missing → exit code 3.
- **No config-file secrets.** No `.env` files, no `~/.swytchcode/credentials` (except optional local OAuth store for dev).
- **No interactive auth during exec.** No browser-based OAuth, no paste-this-token prompts during `exec`.

### OAuth

- **User/browser OAuth:** User completes OAuth **outside** Swytchcode. Access token provided via env vars. Kernel only uses token from env; does not run OAuth flow.
- **Machine-to-machine:** Kernel may obtain tokens using client ID/secret from env (non-interactive, HTTP only, timeout-bound).
- **Refresh tokens:** If Wrekenfile specifies refresh, kernel may refresh using refresh token from env (non-interactive, HTTP-only).

### Other Auth Types

- **API keys:** Env var per key; name defined in Wrekenfile.
- **Bearer tokens:** Env var holds the token.
- **Basic auth:** Username/password from two env vars.

---

## HTTP and API Usage

### HTTP Client

- **Use Go's standard `net/http`** with a **single shared `*http.Client`** per process (custom `Transport` for connection pooling).
- **Thin wrapper only:** Retries, backoff, auth, and policy live in the kernel layer. The HTTP layer must remain a thin wrapper over `net/http`.
- **Do not introduce** a third-party REST client into the kernel unless this decision is explicitly revisited.

**Rule:** All API calls (registry, SDK) must be timeout-bound and non-interactive.

---

## CI and .swytchcode/ Policy

### Local Development

- `.swytchcode/` may be uncommitted.
- Developers may experiment freely using `init` and `get`.

### CI / Docker / Production

- `.swytchcode/` **must exist**.
- If `.swytchcode/` is missing, `swytchcode exec` must fail.
- CI must never infer or auto-download integrations. Required integrations are declared in `tooling.json` (`integrations`); `swytchcode bootstrap` installs those exact versions.

**In CI, Swytchcode never infers integrations.**

**Failures:**
- If `.swytchcode/` is missing → execution fails
- If a tool is not declared in `tooling.json` → execution fails
- If a raw method is used without explicit opt-in → execution fails
- If mode is not set → defaults to `production` (strict policies enforced)

**Rule:** If execution matters, it must be declared and reviewable.

---

## No Workflows (by Design)

Wrekenfiles describe **individual SDK methods**, not workflows.

**This restriction applies to Wrekenfiles only.** Verified workflows may exist as first-class tools in `tooling.json`, with explicit I/O contracts and transparent kernel implementation.

**Swytchcode intentionally does not own workflows:**
- No hidden sequences
- No implicit multi-step behavior
- No business logic encoded in specs

**Workflows emerge through composition:**
- In application code
- In agent reasoning
- In CI scripts

If a multi-step operation must exist, it should be exposed explicitly as a single tool in `tooling.json`, implemented transparently by the kernel.

**Nothing is hidden inside Wrekenfiles.**

---

## What NOT to Add (Explicitly Forbidden)

- ❌ No implicit “latest” at exec or bootstrap (versions must be explicit in tooling.json)
- ❌ No registry calls from `exec`
- ❌ No persisting env overrides into tooling.json
- ❌ No auto-promotion
- ❌ No inference from imports
- ❌ No "helpful" fallback to raw
- ❌ No workflows hidden in Wrekenfiles
- ❌ No agent auto-discovery of raw methods
- ❌ No language SDK wrappers
- ❌ No plugin system
- ❌ No background daemons
- ❌ No config auto-magic
- ❌ No interactive exec
- ❌ No client-side retries

**If it feels "helpful", it's probably wrong.**

---

## Exit Code Contract

**Locked contract** (must not change casually):

| Code | Meaning                    |
| ---- | -------------------------- |
| 0    | Success                    |
| 1    | Invalid input              |
| 2    | Tool not found             |
| 3    | Auth missing/invalid       |
| 4    | SDK execution failure      |
| 5    | Internal error             |

This contract is for CI, Docker images, and agents.

---

## Command Interaction Matrix

| Command                  | Human (TTY)        | CI / No TTY                    |
| ------------------------ | ------------------ | ------------------------------- |
| `swytchcode init`        | ✅ Interactive     | ❌ Error unless flags provided  |
| `swytchcode get`         | ✅ Optional prompts | ❌ Non-interactive              |
| `swytchcode bootstrap`   | ❌ No prompts      | ❌ No prompts                  |
| `swytchcode config`      | ❌ No prompts      | ❌ No prompts                  |
| `swytchcode add integration` | ❌ No prompts  | ❌ No prompts                  |
| `swytchcode validate`        | ❌ No prompts  | ❌ No prompts                  |
| `swytchcode apply`          | ❌ No prompts  | ❌ No prompts                  |
| `swytchcode exec`           | ❌ Never       | ❌ Never                        |

**Rule:** Only setup commands (init, get) are interactive. Execution, validate, apply, and bootstrap/config are never interactive.

---

## Key Invariants (canonical list)

The list below is the **canonical** set of non-negotiable principles. README and CONTRIBUTING reference this section to avoid drift.

1. **`swytchcode get` installs capability; it never grants permission.** Get must never modify `tooling.json` or enable tools. **`swytchcode get` is exploratory only.** Nothing fetched by get is ever trusted until pinned in `tooling.json` and installed via `bootstrap`.

2. **`tooling.json` defines what is trusted. Wrekenfiles define what is possible.**

3. **tooling.json pins what is trusted. The registry supplies how it works. `bootstrap` reconciles the two. `exec` only executes.** No implicit “latest” at exec or bootstrap.

4. **`swytchcode exec` must never call the registry.** All data comes from local tooling.json and Wrekenfiles only.

5. **Env overrides are visible and never persisted.** `SWYTCHCODE_REGISTRY_URL` overrides `registry_url` at runtime only; the CLI never writes that value into `tooling.json`. Use `swytchcode config` to see effective config and source.

6. **Proposals cannot mutate `version`, `mode`, or `registry_url`.** Those fields are kernel-owned. Apply rejects any proposal that includes them.

7. **Exec never initiates authentication.** No browser, no login prompts during exec.

8. **Same binary everywhere.** Behavior by TTY + flags only, not environment type.

9. **If Swytchcode works in CI, it will work in production.** If it only works in an IDE, it is broken.

---

## Summary

Swytchcode is friendly during setup and ruthless during execution.

The kernel is the execution authority. Everything else is a guest.

**Determinism guarantee:** Integration versions are explicit in `tooling.json`. `bootstrap` installs those exact versions. `exec` uses only local files and never calls the registry. No implicit “latest” — CI, audits, and rollbacks stay reproducible.
