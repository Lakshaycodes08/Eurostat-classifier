# Swytchcode Design Document

This document captures the architectural decisions and design principles for Swytchcode. Treat this as the authoritative source of truth for design discussions.

**Last updated:** Based on architectural decisions finalized during initial design phase.

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
- Is **committed**, reviewed, and stable.
- Is the **only agent-facing contract**.

**Rule:** If it affects *what a tool is allowed to do*, it belongs in `tooling.json`.

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
4. Team promotes it into `tooling.json` (via `propose` → `apply`)
5. Method becomes verified, agent-safe, and CI-safe

### Propose → Review → Apply

**Agents may propose; they may not apply.**

1. **Propose:** `swytchcode propose stripe customers.search`
   - Generates a **patch** (e.g. `.swytchcode/proposals/stripe.customers.search.json`)
   - Does **not** edit `tooling.json`

2. **Review:** Human or CI validates the proposal
   - Schema correctness
   - Naming rules
   - Auth compatibility
   - No forbidden patterns

3. **Apply:** `swytchcode apply proposals/stripe.customers.search.json`
   - **Only command** that mutates `tooling.json`
   - Requires clean validation

**Invariant:** `tooling.json` is write-protected; all changes go through proposals. Agents prepare; humans approve; kernel enforces.

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

- **Local development:** Optional OAuth login may store tokens in a local auth store (e.g. `~/.swytchcode/auth/<provider>.json`).
- **CI and production:** Authentication is **env-only**. No local auth store.
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
- CI must never infer or auto-download integrations.

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

| Command        | Human (TTY)        | CI / No TTY                    |
| -------------- | ------------------ | ------------------------------- |
| `swytchcode init` | ✅ Interactive     | ❌ Error unless flags provided  |
| `swytchcode get`  | ✅ Optional prompts | ❌ Non-interactive              |
| `swytchcode exec` | ❌ Never           | ❌ Never                        |

**Rule:** Only setup commands are interactive. Execution is never interactive.

---

## Key Invariants

1. **`swytchcode get` installs capability; it never grants permission.** Get must never modify `tooling.json` or enable tools.

2. **`tooling.json` defines what is trusted. Wrekenfiles define what is possible.**

3. **Exec never initiates authentication.** No browser, no login prompts during exec.

4. **Same binary everywhere.** Behavior by TTY + flags only, not environment type.

5. **If Swytchcode works in CI, it will work in production.** If it only works in an IDE, it is broken.

---

## Summary

Swytchcode is friendly during setup and ruthless during execution.

The kernel is the execution authority. Everything else is a guest.
