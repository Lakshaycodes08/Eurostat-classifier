## Swytchcode Kernel (Go skeleton)

Swytchcode is the **execution kernel** for tools. Editors, agents, and languages are **guests** that must call `swytch exec` instead of doing their own SDK logic.

This repo contains a **minimal, opinionated Go skeleton** for that kernel, ready for a Go developer to extend.

Swytchcode is not a library.  
It is a kernel.  
All languages, editors, and agents are guests.

**`tooling.json` defines what is trusted.**  
**Wrekenfiles define what is possible.**

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

Wrekenfiles describe **individual SDK methods**, not workflows.

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

- **Single execution path**: All execution flows through `swytch exec`.
- **Deterministic**: Same JSON input → same JSON output, no hidden state.
- **Exec is never interactive**: No prompts, no TTY detection, no prose.
- **Setup only is interactive**: `init` and `get` may prompt on a TTY, and must be scriptable via flags.
- **Editor‑agnostic at runtime**: Editor rules are for authoring only; `exec` ignores them.
- **Env‑only auth**: Secrets come from environment variables only.
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
    - Write editor‑specific configs via `internal/editors/*` (Cursor, VS Code, Claude), if editor ≠ `none`.

- **`swytchcode get <library>`**
  - **Purpose**: Fetch and manage Wrekenfiles (e.g. `stripe`, `openai`).
  - **Human / TTY**:
    - With no args, `swytchcode get` may prompt to select a library and confirm overwrites.
  - **CI / non‑interactive**:
    - Should be usable as: `swytchcode get stripe --yes --non-interactive`.
    - If overwrite is needed and `--yes` is missing → fail, do not prompt.
  - **Responsibilities**:
    - Resolve library → registry endpoint.
    - Fetch Wrekenfile JSON/YAML.
    - Validate schema.
    - Save to `.swytchcode/wrekenfiles/<library>.json`.
    - Never execute tools or touch `tooling.json`.

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
  - **Purpose**: Discover available methods without executing anything.
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
  - **Purpose**: Inspect a tool or raw method without execution.
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
    - Schema validation and env‑based auth checks.
    - SDK invocation layer and error mapping to exit codes.
  - Implement `internal/wreken/*` and `internal/tooling/*` loaders/validators.

- **Quality gates**
  - Ensure `echo '{"tool":"x.y","args":{}}' | swytchcode exec` works:
    - In a Docker container.
    - In GitHub Actions or GitLab CI.
    - With no TTY, no `$HOME`, and no editor present.

Once these pieces are in place, your thin clients (Cursor/VS Code/agents) should be <50 LOC and defer all execution to this kernel.

