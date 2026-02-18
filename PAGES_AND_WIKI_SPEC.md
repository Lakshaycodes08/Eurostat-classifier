# GitLab Pages + Wiki Redesign – Discussion & Recommendations

This doc responds to your content spec: validates the structure, calls out one important implication, recommends what you might have missed (commands, concepts), and suggests rollout order.

---

## 1. Structure – Validated with One Caveat

### Pages structure you proposed

```
/ → index.html (Landing)
/install/ → index.html, install.sh, install.ps1
/docs/ → overview, concepts, cli, security
/examples/ → cursor, frontend-agent
/roadmap.html
```

**Recommendation:** Keep it. One caveat: **install script URLs**.

Right now the one-liner is:

- `curl -fsSL https://swytchcode.gitlab.io/cli/install.sh | sh`

If scripts live under **/install/** you have two options:

| Option | URL | Pros | Cons |
|--------|-----|------|------|
| **A) Scripts at root** | `.../install.sh` | No change to one-liner; docs and CI already use this | Slightly less “clean” URL tree |
| **B) Scripts under /install/** | `.../install/install.sh` | Nice hierarchy; install page and scripts colocated | **All existing one-liners and README/SETUP_INSTRUCTIONS must be updated**; CI must copy to `public/install/` |

**Recommendation:** Either keep scripts at **root** (`public/install.sh`, `public/install.ps1`) and add **/install/index.html** as the install *docs* page only, or move to **/install/** and update everywhere (README, SETUP_INSTRUCTIONS, INSTALL_SCRIPT_PLAN, CI, any external links). If you move, do it in one change and document the new URLs.

---

## 2. What You Might Have Missed

### 2.1 Full CLI surface (for Pages “CLI” + Wiki “CLI Reference”)

Your spec calls out `swytch exec` and exit codes. For completeness, document the full set so nothing is missing:

| Command | Purpose (for docs) |
|--------|---------------------|
| `swytchcode -v` / `--version` | Version |
| `swytchcode init` | One-time project setup: `.swytchcode/`, `tooling.json`, editor rules (Cursor / Claude) |
| `swytchcode get <project>` | Fetch integration bundle (Wrekenfile, methods, workflows); does **not** add to tooling.json |
| `swytchcode bootstrap` | Fetch all integrations listed in `tooling.json` (CI-friendly) |
| `swytchcode list` | List tools + integrations from tooling.json + local integrations |
| `swytchcode list methods [pattern]` | List methods (optional filter by canonical_id / project) |
| `swytchcode list workflows [pattern]` | List workflows (optional filter) |
| `swytchcode list integrations` | List fetched integrations only |
| `swytchcode search [keyword]` | Search **remote** registry (no keyword = all) |
| `swytchcode add [spec] <canonical_id>` | Add a tool to `tooling.json` |
| `swytchcode info <canonical_id>` | Show tool info (resolved inputs/output) |
| `swytchcode exec [canonical_id]` | **Single execution path** – CLI args or JSON stdin; `--json`, `--raw`, `--dry-run` |
| `swytchcode mcp serve` | Start MCP server (stdio/HTTP); exposes init, bootstrap, list, search, get, add, info, exec |
| `swytchcode mcp status` / `mcp stop` | Daemon status / stop |

**Recommendation:**

- **Pages /docs/cli.html:** Short table like above + “Details in Wiki → CLI Reference”.
- **Wiki CLI Reference:** Full spec: inputs, outputs, exit codes, failure modes, JSON schema for `exec` (stdin + stdout).

### 2.2 Concepts to name explicitly (Overview + Wiki Core Concepts)

So your “how Swytchcode thinks” and “why it’s different” stay accurate:

- **Execution authority** – Only `swytchcode exec` runs tools; agents/editors call it, they don’t run arbitrary code.
- **tooling.json** – Single source of truth for *what is allowed*; human-readable, machine-enforced.
- **Wrekenfile** – Defines *what is possible* (METHODS, WORKFLOWS, STRUCTS) for an integration; fetched by `get` / `bootstrap`.
- **Canonical ID** – Stable identifier for a method/workflow (e.g. `api.cluster.create`); used in add/list/exec.
- **Methods vs workflows** – Methods = single callable; workflows = multi-step; both are tools and both go through `exec`.
- **Deterministic JSON** – Exec input/output is JSON; no prompt-based execution; failures are explicit (exit codes, structured error).
- **Registry vs local** – `search` hits the registry; `get`/`bootstrap`/`list`/`exec` use **local** data only (no runtime registry calls).
- **Integrations** – Project-scoped bundles (e.g. `weaviate@lyrid.v1`); live under `.swytchcode/integrations/` after `get`.

**Recommendation:**  
In **/docs/overview** (and **Wiki → Core Concepts**), define these terms and add one line: *“Swytchcode treats AI as a planner, not an executor.”*

### 2.3 Integrations (Wiki) – Align with current product

You listed Cursor, Claude, MCP. Current codebase has:

- **Cursor** – `.cursor/rules/swytchcode.mdc` (from `init`); agents use MCP or rules that call `swytchcode exec`.
- **Claude** – `CLAUDE.md` at repo root (from `init`); same idea: use tooling.json + exec.
- **MCP** – `swytchcode mcp serve` exposes all main commands; Cursor/other MCP clients talk to Swytchcode via MCP.

**Recommendation:**  
Wiki → Integrations: one page per (Cursor, Claude, MCP) explaining how each uses Swytchcode (read-only tool list + `exec` as sole execution path). Remove “Copilot” from any new copy; you already dropped it from the app.

### 2.4 Security page – Optional extra bullets

On top of what you had:

- **No runtime registry** – Exec uses only tooling.json + local integrations; no “phone home” during execution.
- **Explicit allow-list** – Only tools in `tooling.json` can be executed.
- **No hidden retries** – Failures surface as exit codes / JSON; no silent retries or agent-side interpretation of policy.

---

## 3. How Pages + Wiki Work Together (your table – kept)

| Pages | Wiki |
|-------|------|
| Public, polished | Detailed, evolving |
| Install + overview | Specs + philosophy |
| Entry point | Deep reference |
| Minimal | Exhaustive |

- **Pages → Wiki:** “Full CLI reference and design rationale → Wiki.”
- **Wiki → Pages:** “Install and quick start → Pages.”

---

## 4. Rollout Recommendation

**Phase 1 (non-negotiable)**  
- Pages: **Landing (/)**, **Install (/install)** – with a clear decision: scripts at root vs under `/install/` and update all references.  
- Pages: **/docs/overview** (concepts + “AI as planner, not executor”).  
- Wiki: **Home** (what it is / isn’t, philosophy, link to Pages install).  
- Wiki: **Core Concepts** (execution model, tooling.json, Wrekenfile, canonical ID, methods vs workflows, determinism).

**Phase 2**  
- Pages: **/docs/security** (trust, no hidden execution, explicit policy).  
- Pages: **/docs/cli** (command table + link to Wiki for full reference).  
- Wiki: **CLI Reference** (full spec: exec, exit codes, JSON schema, all commands).

**Phase 3**  
- Pages: **/docs/concepts** (optional if overview is enough), **/examples** (cursor, frontend-agent), **/roadmap**.  
- Wiki: **Integrations** (Cursor, Claude, MCP), **Security Model**, **FAQ**, **Roadmap & Philosophy**.

This keeps “install + overview + security” first, then fills in CLI and examples, then deep Wiki and roadmap.

---

## 5. Summary

- **Structure:** Your Pages + Wiki layout is good; decide **install script URL** (root vs `/install/`) and update docs + CI in one go if you move.
- **Coverage:** Add the **full command list** and **concepts** (tooling.json, Wrekenfile, canonical ID, methods vs workflows, registry vs local, no Copilot) so Pages and Wiki stay in sync with the product.
- **Order:** Landing + Install + Overview (+ optional Security) first; then CLI docs and Wiki depth; then examples and roadmap.

If you want, next step can be: (1) a concrete change list for CI and repo (where scripts live, what to copy into `public/`), and (2) minimal HTML/markdown outlines for `index.html`, `install/index.html`, and `docs/overview.html` so you can paste in your copy.
