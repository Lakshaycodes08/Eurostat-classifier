# Editor rules and instructions

This directory is the **human-editable copy** of the editor rule templates. The canonical source used by `swytchcode init` is **`internal/editors/templates/`** (embedded in the binary). When you change rules here, copy the same content into `internal/editors/templates/` so init installs the updated text. When a user runs `swytchcode init --editor=<cursor|claude|vscode>` in their repo, the CLI writes the embedded templates into their project.

## Intent

- **Swytchcode is the sole authority** for external API execution. Agents must use MCP tools or `swytchcode exec`, never raw HTTP or direct config edits.
- **`.swytchcode` is not for hand editing.** The only way to change it via MCP is **swytchcode_add** (or `swytchcode add` CLI). Do not expose tooling.json, Wrekenfiles, or integration files as MCP resources.
- **Snake_case only.** We expose and document only the MCP tools that exist: `swytchcode_list`, `swytchcode_get`, `swytchcode_add`, `swytchcode_exec`. No `swytchcode_describe` or other unimplemented tools.

## Where each template goes (in the user’s repo)

| Editor   | Source (this repo)              | Destination (user repo)           |
|----------|----------------------------------|-----------------------------------|
| Cursor   | `editors/cursor/swytchcode.mdc`  | `.cursor/rules/swytchcode.mdc`     |
| Claude   | `editors/claude/CLAUDE.md`       | **`CLAUDE.md`** (repo root)       |
| VS Code  | Copilot: `editors/vscode/copilot/swytchcode.md` | `.github/instructions/swytchcode.md` |
| VS Code  | Claude: same as Claude           | **`CLAUDE.md`** (repo root)       |

So for VS Code, init can write both `.github/instructions/swytchcode.md` (Copilot) and `CLAUDE.md` (Claude in VS Code) from the same templates.

## MCP tools we expose (no resources)

- **swytchcode_list** — list available integrations
- **swytchcode_get** — fetch integration bundles
- **swytchcode_add** — add a tool to tooling (only MCP command that modifies `.swytchcode`)
- **swytchcode_exec** — execute a tool by canonical_id

Do not expose MCP **resources** that read tooling.json, manifest.json, wrekenfiles, methods.json, or workflows.json.

## Developer checklist (implement in shell)

1. **Init**
   - `swytchcode init --editor=cursor` → copy `editors/cursor/swytchcode.mdc` → `.cursor/rules/swytchcode.mdc`
   - `swytchcode init --editor=claude` → copy `editors/claude/CLAUDE.md` → `<projectRoot>/CLAUDE.md`
   - `swytchcode init --editor=vscode` → copy Copilot template → `.github/instructions/swytchcode.md`, and copy `editors/claude/CLAUDE.md` → `<projectRoot>/CLAUDE.md`

2. **MCP server**
   - Expose only the four tools above (snake_case). Do not expose resources for `.swytchcode` or integrations.

3. **Runtime**
   - Helpers (Go/TS/Python) are thin wrappers around `swytchcode exec`; no auth or schema logic in helpers.
