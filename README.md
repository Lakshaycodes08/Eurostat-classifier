## Swytchcode Kernel (Go skeleton)

Swytchcode is the **execution kernel** for tools. Editors, agents, and languages are **guests** that must call `swytch exec` instead of doing their own SDK logic.

This repo contains a **minimal, opinionated Go skeleton** for that kernel, ready for a Go developer to extend.

**📖 Documentation:**
- **[DESIGN.md](./DESIGN.md)** — Architectural decisions and design principles (source of truth for design discussions)
- **[CONTRIBUTING.md](./CONTRIBUTING.md)** — Developer guide, implementation roadmap, and contribution guidelines

Swytchcode is not a library.  
It is a kernel.  
All languages, editors, and agents are guests.

**`tooling.json` defines what is trusted.**  
**Wrekenfiles define what is possible.**

**Positioning:** Swytchcode is a deterministic execution layer between LLMs/apps and real-world SDKs/APIs — not an agent framework, not observability tooling. **"LLMs ask. Swytchcode executes."**

---

### Repository layout

This project is structured as a single Go module with a `swytchcode` CLI:

- **`cmd/swytchcode/`**: CLI entrypoint
- **`internal/cli/`**: Cobra commands (`init`, `get`, `exec`, `rm`, `upgrade`, `list`, `describe`, `mode`)
- **`internal/kernel/`**: Execution kernel (deterministic, non-interactive)
- **`internal/wreken/`**: Wrekenfile loading and validation
- **`internal/tooling/`**: `tooling.json` contract loader
- **`internal/editors/`**: Init-time only editor configuration writers
- **`internal/util/`**: Shared helpers (interactive detection, JSON IO, filesystem, env)

The `.swytchcode/` directory is created by `swytchcode init`.
It may be uncommitted in local development, but must exist in CI and production.

---

### `tooling.json` vs Wrekenfiles (important)

Swytchcode intentionally separates **execution contracts** from **execution implementations**.

#### `tooling.json` (authoritative contract)
- Defines **what tools are allowed** in this project.
- Defines the **canonical input/output schema** for each tool.
- Stores the **execution mode** (`production` or `sandbox`).
- Is **committed**, reviewed, and stable.
- Is the **only agent-facing contract**.

If it affects *what a tool is allowed to do*, it belongs in `tooling.json`.

Example structure:
```json
{
  "mode": "production",
  "tools": {
    "stripe.createCustomer": {
      "library": "stripe",
      "operation": "createCustomer"
    }
  }
}
```

#### Wrekenfiles (implementation details)
- Define **how an SDK method is executed**.
- Encode SDK calls, auth mapping, retries, and metadata.
- Are **library-specific** and may change independently.
- Must **conform to** the I/O contract defined in `tooling.json`.

If it affects *how the SDK is called*, it belongs in a Wrekenfile.

The kernel enforces the boundary:
- Inputs are validated against `tooling.json`.
- SDK outputs are normalized to the shape declared in `tooling.json`.
- Extra fields are dropped; missing required fields cause failure.

Wrekenfiles must never define public I/O schemas.

---

### Verified tools vs Raw methods

Swytchcode intentionally exposes two execution surfaces.

#### Verified tools (tooling.json)
Verified tools are explicitly declared in `tooling.json`.

Properties:
- Reviewed and intentional
- Stable input/output contracts
- Safe for CI, production, and agents
- Suggested by editors and agents by default

`tooling.json` defines what is **trusted**.

This surface may include:
- Single SDK methods
- Curated SDK methods
- Verified multi-step workflows (explicitly declared)

---

#### Raw methods (Wrekenfiles)
Raw methods are defined in Wrekenfiles but **not promoted** into `tooling.json`.

Properties:
- Fully executable
- Discoverable
- Not guaranteed stable
- Not allowed in CI or agents by default

Wrekenfiles define what is **possible**.

Raw methods are intended for:
- Exploration
- Advanced users
- Rapid iteration before promotion

Execution of raw methods is always **explicitly opt-in**.

---

### CI and `.swytchcode/` policy

`.swytchcode/` is **execution metadata**, not user state.

#### Local development
- `.swytchcode/` may be uncommitted.
- Developers may experiment freely using `init` and `get`.

#### CI / Docker / production
- `.swytchcode/` **must exist**.
- If `.swytchcode/` is missing, `swytchcode exec` must fail.
- CI must never infer or auto-download integrations.
- Mode must be explicitly set during `init`: `swytchcode init --mode=production --editor=none --non-interactive`.

In CI, Swytchcode never infers integrations.

If `.swytchcode/` is missing, execution fails.  
If a tool is not declared in `tooling.json`, execution fails.  
If a raw method is used without explicit opt-in, execution fails.  
If mode is not set, defaults to `production` (strict policies enforced).

Swytchcode does not guess in CI.
If execution matters, it must be declared and reviewable.

---

### No workflows (by design)

Wrekenfiles describe **individual SDK methods**, not workflows. This restriction applies to Wrekenfiles only. Verified workflows may exist as first-class tools in `tooling.json`, with explicit I/O contracts and transparent kernel implementation.

Swytchcode intentionally does not own workflows:
- No hidden sequences
- No implicit multi-step behavior
- No business logic encoded in specs

Workflows emerge through **composition**:
- In application code
- In agent reasoning
- In CI scripts

If a multi-step operation must exist, it should be exposed explicitly
as a single tool in `tooling.json`, implemented transparently by the kernel.

Nothing is hidden inside Wrekenfiles.

---

### Promotion model

The expected lifecycle of a method is:

1. Method exists in a Wrekenfile
2. User discovers it via `swytchcode list`
3. User experiments locally using raw execution
4. Team promotes it into `tooling.json`
5. Method becomes verified, agent-safe, and CI-safe

Swytchcode does not auto-promote anything.

---

### Non‑negotiable principles

- **Single execution path**: All execution flows through `swytch exec`. Only `exec` is for integration; commands like `list`, `describe` are diagnostic/setup only — agents must not rely on them for execution.
- **Deterministic**: Same JSON input → same JSON output, no hidden state.
- **Exec is never interactive**: No prompts, no TTY detection, no prose.
- **Setup only is interactive**: `init` and `get` may prompt on a TTY, and must be scriptable via flags.
- **Editor‑agnostic at runtime**: Editor rules are for authoring only; `exec` ignores them.
- **Env‑only auth**: Secrets come from environment variables only. **Exec never initiates auth** (no browser, no login prompts during exec).
- **Stable exit codes**: Locked contract for automation (see below).

If an implementation choice violates these, it is wrong.

---

### What NOT to add (explicitly forbidden)

The following are non-negotiable:

- ❌ No auto-promotion
- ❌ No inference from imports
- ❌ No "helpful" fallback to raw
- ❌ No workflows hidden in Wrekenfiles
- ❌ No agent auto-discovery of raw methods

---

### Design decisions and reference (for implementation and discussion)

The following are locked design decisions. When implementing or discussing later, treat this section as the source of truth.

#### 1. One true integration command

- **`swytchcode exec`** is the **only** command that application code, agents (Cursor, Copilot, etc.), or automations are allowed to call for execution.
- All other commands are **diagnostic or setup only**: `list`, `describe`, `plan`, `dry-run`, `explain` (if added) are for humans debugging, inspecting, or learning. **Agents must never rely on them for integration.**
- App code does not orchestrate retries, policies, tool selection, or fallbacks. Agents don’t “reason about tools” — they execute via `exec`.

#### 2. Thin clients and how projects call Swytchcode

- Other projects **never** integrate Swytchcode logic; they **shell out** to `swytchcode exec` through a **thin client** that looks native to their language.
- **Canonical path:** App code → Thin client (serialize → invoke CLI → deserialize) → `swytchcode exec` → tooling.json + Wrekenfile → real SDKs/APIs. There is no alternate path.
- Thin client does **only**: serialize input to JSON, invoke `swytchcode exec`, read stdout/exit code, deserialize output. It does **not**: decide which tool to call, retry, interpret failures, apply policy, or chain calls.
- Thin clients can be implemented in any language; the kernel is language-agnostic. Target: thin client **&lt;50 LOC** per language.

#### 3. Installation model (CLI first)

- The **Swytchcode CLI** and **thin clients** are **separately installed**.
- **Recommended (Option 1):** CLI is primary; users install the CLI explicitly (e.g. `curl -fsSL ... | sh`). Thin clients assume `swytchcode` is on PATH. If CLI is missing, thin client fails with a clear error.
- **Do not** default to thin clients auto-installing the CLI (Option 2). That leads to version fragmentation and hidden binaries; CLI-first keeps one kernel per machine, one upgrade surface, and auditability. Optional “install CLI from client” can be layered on later as opt-in.

#### 4. Same kernel everywhere (local, CI, prod)

- **One binary** for local dev, GitHub Actions, Docker, and production. No “CI mode” or “dev mode” binary.
- Behavior is determined by **TTY + flags only** (e.g. `util.IsInteractive()`). No environment guessing (e.g. `CI=true`). In CI there is no TTY, so init/get require flags; exec is never interactive anywhere.
- If Swytchcode works in CI, it will work in production. If it only works in an IDE, it is broken.

#### 5. Method naming (canonical IDs vs titles)

- **Executable tool names** are **canonical IDs**: lowercase, dot-separated, stable (e.g. `stripe.createCustomer`, `openai.responses.create`). No spaces, no punctuation except `.` and `_`. Used in `exec`, CI, thin clients, and agents.
- **Titles and descriptions** are **metadata only** (e.g. `"title": "Create Stripe Customer"`). They are for discovery and editors; they are never parsed, matched, or executed. Titles can change; IDs must not.

#### 6. Promotion and proposals (propose → review → apply)

- **Promotion into `tooling.json` is always explicit and human-controlled.** No auto-promotion.
- **Agents may propose; they may not apply.** Recommended flow: **propose → review → apply.**
  - **Propose:** e.g. `swytchcode propose stripe customers.search` generates a **patch** (e.g. `.swytchcode/proposals/stripe.customers.search.json`), not an edit to `tooling.json`.
  - **Review:** Human or CI validates the proposal (schema, naming, auth).
  - **Apply:** e.g. `swytchcode apply proposals/stripe.customers.search.json` is the only command that mutates `tooling.json`.
- **Invariant:** `tooling.json` is write-protected; all changes go through proposals. Agents prepare; humans approve; kernel enforces.

#### 7. Verified vs custom workflows

- **Verified workflows** (backend-owned, curated) may be **applied directly** (e.g. `swytchcode add workflow stripe.customer-onboarding`). They are trusted; no proposal step.
- **Custom workflows** (user-assembled, agent-assembled, or backend-generated but not in the verified catalog) are **never auto-applied**. They must be written as **proposals** (e.g. `.swytchcode/proposals/workflow.<name>.json`). Only after explicit **apply** do they become part of `tooling.json`.
- **Rule:** Custom workflows are always proposed, never applied automatically — even if machine-generated.

#### 8. IDE: accepting proposals (no auto-apply)

- In the IDE (Cursor, VS Code, Claude Code), when a proposal exists, the UX must **show → explain → require explicit action**.
- **Show a diff** (what would change in `tooling.json`), not just a yes/no prompt. User must see side effects, auth scope, and schema.
- **Explicit acceptance:** e.g. button “Apply workflow” or command “Swytchcode: Apply Proposal”, which runs `swytchcode apply ...`. No silent or automatic apply. No “approve?” dialogs that auto-apply on yes.
- Agents may generate proposals and ask the user to approve; agents must not apply proposals or modify `tooling.json` directly.

#### 9. Authentication (exec never initiates auth)

- **Authentication is configured explicitly and consumed deterministically. `exec` never initiates authentication.**
- All secrets from **environment variables**; Wrekenfiles declare which env var(s) each operation needs. Missing → exit code 3.
- Optional **interactive auth** (e.g. `swytchcode auth login github`) is **local dev only**; never usable in CI. Tokens from login are stored in a local auth store; in CI/prod only env vars are used.
- OAuth refresh (if supported) must be non-interactive and explicit (e.g. `--allow-auth-refresh`); in CI default is no refresh, expired token → fail.

#### 10. HTTP and API usage in the kernel

- Use **Go’s standard `net/http`** with a **single shared `*http.Client`** per process (custom `Transport` for connection pooling). No heavy third-party REST client in the kernel; retries, auth, and policy live in the kernel layer, not in the HTTP layer.
- All API calls (registry, SDK) must be **timeout-bound and non-interactive**.

---

### Commands and behavior (developer contract)

- **`swytchcode init`**
  - **Purpose**: One‑time project setup, `.swytchcode/` creation, editor rules, and mode configuration.
  - **Behavior** (interactive by default):
    - When running on a TTY without flags, `init` interactively prompts for:
      - Editor selection (`cursor | vscode | claude | none`)
      - Mode selection (`production | sandbox`)
    - Example interactive session:
      ```
      $ swytchcode init
      
      Which editor do you use?
        1) cursor
        2) vscode
        3) claude
        4) none
      Select [1-4]: 1
      
      Which execution mode do you want to use?
        1) production
        2) sandbox
      Select [1-2]: 1
      
      Swytchcode initialized for project at /path/to/project
      ```
    - If `--non-interactive` is set, prompts are disabled and flags are required.
  - **CI / non‑interactive**:
    - Use: `swytchcode init --editor=cursor --mode=production --non-interactive`.
    - If non‑interactive and `--editor` is missing → error (exit code 1).
    - If non‑interactive and `--mode` is missing → error (exit code 1).
    - All required parameters must be provided via flags.
  - **Flags**:
    - `--editor`: Set editor (`cursor | vscode | claude | none`)
    - `--mode`: Set execution mode (`production | sandbox`)
    - `--non-interactive`: Disable prompts (required for CI)
  - **Responsibilities**:
    - Detect project root.
    - Create `.swytchcode/`.
    - Write `tooling.json` with mode configuration.
    - Write editor‑specific configs via `internal/editors/*` (Cursor, VS Code, Claude), if editor ≠ `none`. For Cursor: create **`.cursor/rules/swytchcode.mdc`** with **JSON content** (not prose), instructing Cursor to always call the thin client, never plan tools, and always defer execution to `swytchcode exec`.

- **`swytchcode get <library>`**
  - **Purpose**: Fetch and install integration capability (Wrekenfile) for a library. **Does not enable tools** — that is promotion into `tooling.json`.
  - **Rule**: **get installs capability; it never grants permission.** Get must **never** modify `tooling.json`.
  - **Invariant:** `swytchcode get` installs integration capability only. It must never modify `tooling.json` or enable tools.
  - **Human / TTY**:
    - With no args, `swytchcode get` may prompt to select a library and confirm overwrites.
  - **CI / non‑interactive**:
    - Should be usable as: `swytchcode get stripe --yes --non-interactive`.
    - If overwrite is needed and `--yes` is missing → fail, do not prompt.
  - **Responsibilities**:
    - Resolve library → registry endpoint.
    - Fetch Wrekenfile JSON/YAML (and optional manifest/metadata).
    - Validate schema.
    - Save to `.swytchcode/wrekenfiles/<library>.json` (or `.swytchcode/integrations/<library>/wreken.yaml` per your layout).
    - Never execute tools, never touch `tooling.json`, never run OAuth or fetch secrets.

- **`swytchcode rm <library>`**
  - **Purpose**: Delete a local Wrekenfile spec for a library.
  - **Human / TTY**:
    - May prompt to confirm deletion (once implemented).
    - For now, requires an explicit `--yes` to proceed.
  - **CI / non‑interactive**:
    - Use as: `swytchcode rm stripe --yes --non-interactive`.
    - Must never block or prompt; fails fast without `--yes`.
  - **Responsibilities**:
    - Remove `.swytchcode/wrekenfiles/<library>.json` if it exists.
    - Treat missing specs as a deterministic error (no “best effort”).

- **`swytchcode upgrade <library>`**
  - **Purpose**: Refresh an existing Wrekenfile spec from the registry.
  - **Human / TTY**:
    - May prompt before overwriting the existing spec (once implemented).
    - For now, requires `--yes` to indicate an intentional upgrade.
  - **CI / non‑interactive**:
    - Use as: `swytchcode upgrade stripe --yes --non-interactive`.
    - Must never prompt and must be safe to run repeatedly.
  - **Responsibilities**:
    - Verify an existing spec is present.
    - Fetch latest Wrekenfile as in `get`.
    - Validate and overwrite the local spec atomically.

- **`swytchcode list <library>`**
  - **Purpose**: Discover available methods without executing anything. **Diagnostic/setup only** — not for integration; agents must not rely on this for execution.
  - **Output**: JSON to stdout:
    ```json
    {
      "verified": [
        "stripe.createCustomer",
        "stripe.attachPaymentMethod"
      ],
      "raw": [
        "customers.search",
        "customers.update",
        "subscriptions.resume"
      ]
    }
    ```
  - **Flags**:
    - `--raw`: Show only raw methods
    - `--verified`: Show only verified tools
  - **Behavior**:
    - Reads `tooling.json` + Wrekenfile
    - Never executes anything
    - Safe for CI
    - Default output is JSON only (no prose)

- **`swytchcode describe <tool>`**
  - **Purpose**: Inspect a tool or raw method without execution. **Diagnostic only** — not for integration.
  - **Examples**:
    - `swytchcode describe stripe.createCustomer` (verified tool)
    - `swytchcode describe raw.stripe.customers.search` (raw method)
  - **Behavior**:
    - Verified tool → show I/O schema from `tooling.json`
    - Raw method → show metadata from Wrekenfile
    - No SDK calls
    - No side effects

- **`swytchcode mode [production|sandbox]`**
  - **Purpose**: Set or display the execution mode for the project.
  - **Modes**:
    - `production`: Use production credentials and enforce strict policies
    - `sandbox`: Use sandbox/test credentials and allow experimental features
  - **Usage**:
    - `swytchcode mode` → Display current mode (defaults to `production`)
    - `swytchcode mode production` → Set mode to production
    - `swytchcode mode sandbox` → Set mode to sandbox
  - **Behavior**:
    - Mode is stored in `tooling.json` (not a separate config file)
    - Affects credential selection and policy enforcement
    - Default mode is `production` if not explicitly set
    - Can be set during `swytchcode init` or changed later with this command
  - **CI / non‑interactive**:
    - Use: `swytchcode mode production` or `swytchcode mode sandbox`
    - No prompts, deterministic behavior

- **`swytchcode exec`**
  - **Purpose**: The **only** execution path for tools.
  - **Input**: JSON on stdin, e.g.

    ```json
    {
      "tool": "stripe.createCustomer",
      "args": {
        "email": "test@example.com"
      }
    }
    ```

  - **Behavior** (always non‑interactive):
    - Read and parse stdin JSON.
    - If tool starts with `raw.`:
      - Require `--allow-raw` flag
      - If missing → fail with exit code 1
      - Resolve directly via Wrekenfile (bypass `tooling.json`)
    - Else (verified tool):
      - Load `tooling.json`
      - Resolve tool → Wrekenfile
      - Validate input schema
    - Load env‑based credentials.
    - Apply policy (retries, idempotency).
    - Execute SDK call.
    - Normalize and write JSON output to stdout on success.
    - Write JSON error to stderr on failure.
  - **Flags**:
    - `--allow-raw`: Required for executing raw methods (disabled by default)
  - **Executing raw methods (explicit opt-in)**:
    Raw methods may be executed by explicitly opting in.

    Example:
    ```json
    {
      "tool": "raw.stripe.customers.search",
      "args": { "query": "email:'test@example.com'" }
    }
    ```

    This requires:
    ```bash
    swytchcode exec --allow-raw
    ```

    Rules:
    - Raw execution is disabled by default
    - CI must never allow raw execution implicitly
    - Agents must never use raw methods unless explicitly configured
    - There must be no silent fallback from verified → raw
  - **Forbidden**:
    - Prompts, spinners, or human‑oriented prose.
    - Any branching on editor or language.
    - Reading editor config files at runtime.
    - Auto-promotion of raw methods to verified
    - Silent fallback from verified to raw methods

---

### Kernel responsibilities and exit codes

The kernel lives under `internal/kernel/` and must own:

- **Execution orchestration** (no retries in clients).
- **Idempotency and policy** (e.g. safe retries).
- **Error mapping** into the following **stable exit codes**:

| Code | Meaning                    |
| ---- | -------------------------- |
| 0    | Success                    |
| 1    | Invalid input              |
| 2    | Tool not found             |
| 3    | Auth missing/invalid       |
| 4    | SDK execution failure      |
| 5    | Internal error             |

This contract is for CI, Docker images, and agents, and must not change casually.

---

### API and data fetching

The kernel and CLI need to pull data from APIs (e.g. Wrekenfile registry, upstream services).

#### Recommended HTTP client

- **Prefer the standard library** `net/http` for minimal dependencies and good efficiency (connection reuse via `http.DefaultClient` or a shared `*http.Client` with a single `Transport`).
- Use a **single shared `*http.Client`** per process (do not create a new client per request) so connection pooling and keep-alives work.
- Set explicit **timeouts** (e.g. `Client.Timeout`) and consider **context** for cancellation.

Retries, backoff, auth, and policy live in the kernel layer. The HTTP layer must remain a thin wrapper over `net/http`. Do not introduce a third-party REST client into the kernel unless this decision is explicitly revisited.

Rule: **No interactive or blocking behavior** when fetching data. All API calls must be timeout-bound and non-interactive.

---

### Authentication (OAuth and others)

In CI and production, auth is **env-only**. In local dev, optional OAuth login may use a local auth store. The kernel never prompts for secrets during `exec` and never opens browsers.

#### Principles

- **All secrets come from environment variables.**  
  Wrekenfiles declare which env var(s) each operation needs (e.g. `STRIPE_API_KEY`, `GITHUB_TOKEN`). The kernel reads them via `os.Getenv` / `util.GetEnvRequired`. Missing or empty → exit code 3 (auth error).
- **Local dev vs CI/production:**  
  In local development, optional OAuth login may store tokens in a local auth store (e.g. `~/.swytchcode/auth/<provider>.json`). In CI and production, authentication is **env-only**. `swytchcode exec` never initiates authentication in any environment.
- **No interactive auth during exec.**  
  No browser-based OAuth, no paste-this-token prompts during `exec`. The kernel runs non-interactively. Optional `swytchcode auth login` (if implemented) is for local dev only and must never be used in CI.

#### OAuth

- **User / browser OAuth:**  
  The user completes OAuth (e.g. device flow, or login in a browser) **outside** Swytchcode. The resulting **access token** (and optionally refresh token) is then provided to the kernel via **environment variables** (e.g. `GITHUB_TOKEN`, `SLACK_USER_TOKEN`). The kernel only uses whatever token is in env; it does not run the OAuth flow.
- **Machine-to-machine (client credentials, etc.):**  
  If the Wrekenfile supports it, the kernel can **obtain** tokens using client ID and secret from env (e.g. `STRIPE_CLIENT_ID`, `STRIPE_CLIENT_SECRET`). That flow must be non-interactive (HTTP only), timeout-bound, and cache tokens in memory for the process lifetime if desired. No browser, no user interaction.
- **Refresh tokens:**  
  If a Wrekenfile specifies refresh behavior, the kernel may refresh using a refresh token from env, again in a non-interactive, HTTP-only way.

#### Other auth types

- **API keys:** Env var per key; name defined in Wrekenfile.
- **Bearer tokens:** Same; env var holds the token.
- **Basic auth:** Username/password from two env vars if needed.

Summary: **Env-only, Wrekenfile-defined names, no prompts, no browser.** OAuth tokens are either supplied in env after the user completes the flow elsewhere, or obtained via machine-to-machine flows using env-supplied client credentials.

---

### Developer roadmap (what to implement next)

- **CLI wiring**
  - Complete remaining TODOs in `internal/cli/init.go`:
    - Interactive prompts for editor and mode are implemented.
    - Create `.swytchcode/` and `tooling.json` with mode configuration (done).
    - Delegate editor configuration to `internal/editors/*` (done).
    - Interactive mode works by default, with `--non-interactive` for CI (done).
  - Extend `internal/cli/get.go` to:
    - Support optional interactive prompts on TTY.
    - Implement non‑interactive `--yes` behavior.
  - Complete `internal/cli/mode.go`:
    - Integrate mode into kernel execution logic (credential selection, policy enforcement).
    - Ensure mode affects SDK calls and environment variable selection.

- **Kernel and contracts**
  - Flesh out `internal/kernel/*`:
    - Tool resolution from `tooling.json` + Wrekenfiles.
    - Schema validation and env‑based auth checks (see **Authentication** above).
    - SDK invocation layer and error mapping to exit codes.
  - Implement `internal/wreken/*` and `internal/tooling/*` loaders/validators.
  - For registry/API calls: use a single shared `*http.Client` and stdlib only; see **API and data fetching** above.

- **Promotion and proposals** (see **Design decisions and reference**)
  - Implement `swytchcode propose <library> <method>` to generate proposal files under `.swytchcode/proposals/` without modifying `tooling.json`.
  - Implement `swytchcode apply proposals/<name>.json` as the only command that mutates `tooling.json` after validation.
  - Support verified workflows from backend (direct add) vs custom workflows (proposals only, explicit apply).

- **Quality gates**
  - Ensure `echo '{"tool":"x.y","args":{}}' | swytchcode exec` works:
    - In a Docker container.
    - In GitHub Actions or GitLab CI.
    - With no TTY, no `$HOME`, and no editor present.

Once these pieces are in place, your thin clients (Cursor/VS Code/agents) should be <50 LOC and defer all execution to this kernel.

