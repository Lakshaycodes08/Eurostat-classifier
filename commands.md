# Swytchcode CLI Commands Reference

Complete reference for all available `swytchcode` commands.

## Table of Contents

- [Setup Commands](#setup-commands)
- [Discovery Commands](#discovery-commands)
- [Authorization Commands](#authorization-commands)
- [Execution Commands](#execution-commands)
- [Configuration Commands](#configuration-commands)

---

## Setup Commands

### `swytchcode init`

Initialize Swytchcode in the current project. Creates `.swytchcode/` directory, `tooling.json`, and editor-specific configuration files.

**Usage:**
```bash
swytchcode init [flags]
```

**Flags:**
- `--editor <value>`: Set editor (`cursor | vscode | claude | none`)
- `--mode <value>`: Set execution mode (`production | sandbox`)
- `--non-interactive`: Disable prompts (required for CI)

**Behavior:**
- **Interactive mode** (default): Prompts for editor and mode selection
- **Non-interactive mode**: Requires `--editor` and `--mode` flags

**Example:**
```bash
# Interactive
swytchcode init

# Non-interactive (CI)
swytchcode init --editor=cursor --mode=production --non-interactive
```

**Responsibilities:**
- Creates `.swytchcode/` directory
- Creates `tooling.json` with mode configuration
- Writes editor-specific config files (Cursor, VS Code, Claude)
- Sets up project structure for Swytchcode

---

### `swytchcode get <library>`

Fetch and install integration capability (Wrekenfile) for a library. **Exploratory only** — does not modify `tooling.json` or enable tools.

**Usage:**
```bash
swytchcode get [library] [flags]
```

**Flags:**
- `--yes`: Auto-confirm overwrite in non-interactive mode
- `--non-interactive`: Disable prompts (suitable for CI)

**Behavior:**
- **Interactive mode**: If no library specified, prompts to select from available integrations
- **Non-interactive mode**: Requires library name and `--yes` for overwrites

**Example:**
```bash
# Interactive selection
swytchcode get

# Specific library
swytchcode get stripe

# Non-interactive (CI)
swytchcode get stripe --yes --non-interactive
```

**Important:**
- **`swytchcode get` is exploratory only.** Nothing fetched by get is ever trusted until pinned in `tooling.json` and installed via `bootstrap`.
- Never modifies `tooling.json`
- Never enables tools
- Installs Wrekenfile to `.swytchcode/wrekenfiles/<library>.yaml`

---

### `swytchcode bootstrap`

Install exact integration versions declared in `tooling.json`. Reads integration versions from `tooling.json`, downloads exact versions from the registry. Fails if installed version does not match.

**Usage:**
```bash
swytchcode bootstrap
```

**Behavior:**
- Reads `tooling.json` to get required integration versions
- Downloads exact versions from registry
- Fails if installed version doesn't match `tooling.json`
- Never modifies `tooling.json`
- Never installs "latest"

**Example:**
```bash
swytchcode bootstrap
```

**Important:**
- Requires `project_uuid` in `tooling.json`
- Deterministic: always installs exact versions
- Safe for CI and production

---

### `swytchcode rm <library>`

Remove a Wrekenfile spec for a library.

**Usage:**
```bash
swytchcode rm <library> [flags]
```

**Flags:**
- `--yes`: Auto-confirm deletion
- `--non-interactive`: Disable prompts (suitable for CI)

**Example:**
```bash
swytchcode rm stripe --yes --non-interactive
```

**Responsibilities:**
- Removes `.swytchcode/wrekenfiles/<library>.yaml`
- Removes associated proposals for the library
- Fails if Wrekenfile doesn't exist (deterministic error)

---

### `swytchcode upgrade <library>`

Refresh an existing Wrekenfile spec from the registry. **Local convenience only; never used in CI or automation.**

**Usage:**
```bash
swytchcode upgrade <library> [flags]
```

**Flags:**
- `--yes`: Auto-confirm overwrite
- `--non-interactive`: Disable prompts (suitable for CI)

**Example:**
```bash
swytchcode upgrade stripe --yes --non-interactive
```

**Important:**
- **Do not use in CI.** For reproducible builds, pin versions in `tooling.json` and use `bootstrap`
- Use `add integration <name>@<version>` to change versions
- Fetches latest version from registry

---

## Discovery Commands

### `swytchcode list <library>`

List available verified tools and raw methods for a library. **Diagnostic/setup only** — not for integration.

**Usage:**
```bash
swytchcode list <library> [flags]
```

**Flags:**
- `--raw`: Show only raw methods
- `--verified`: Show only verified tools

**Output:** JSON to stdout
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

**Example:**
```bash
swytchcode list stripe
swytchcode list stripe --raw
swytchcode list stripe --verified
```

**Behavior:**
- Reads `tooling.json` + Wrekenfile
- Never executes anything
- Safe for CI
- Default output is JSON only (no prose)

---

### `swytchcode describe <tool>`

Describe a verified tool or raw method without execution. **Diagnostic only** — not for integration.

**Usage:**
```bash
swytchcode describe <tool>
```

**Examples:**
```bash
# Verified tool
swytchcode describe stripe.createCustomer

# Raw method
swytchcode describe raw.stripe.customers.search
```

**Behavior:**
- Verified tool → shows I/O schema from `tooling.json`
- Raw method → shows metadata from Wrekenfile
- No SDK calls
- No side effects
- Output is JSON

---

## Authorization Commands

### `swytchcode validate <proposal_file>`

Validate a proposal file for correctness. Full validation with no side effects.

**Usage:**
```bash
swytchcode validate <proposal_file>
```

**Example:**
```bash
swytchcode validate .swytchcode/proposals/stripe.customers.search.json
```

**Behavior:**
- Validates structural correctness
- Checks kernel-owned fields (no `version`, `mode`, `registry_url`)
- Validates integration compatibility
- Produces structured errors or success
- Exit code 0 if valid, non-zero if invalid
- **No side effects** — never modifies `tooling.json`

---

### `swytchcode apply <proposal_file>`

Apply a validated proposal to `tooling.json`. **Only command that mutates `tooling.json` from proposals.**

**Usage:**
```bash
swytchcode apply <proposal_file>
```

**Example:**
```bash
swytchcode apply .swytchcode/proposals/stripe.customers.search.json
```

**Behavior:**
- Reuses same validation as `validate`
- Merges tools and optional integrations into `tooling.json`
- Archives proposal to `.swytchcode/proposals/applied/`
- Fails if proposal is invalid
- Requires explicit user action (never auto-applied)

---

### `swytchcode add integration <name>@<version>`

Pin an integration version in `tooling.json`. Does not fetch the integration.

**Usage:**
```bash
swytchcode add integration <name>@<version>
```

**Example:**
```bash
swytchcode add integration stripe@2025-01-10
```

**Behavior:**
- Adds integration version pin to `tooling.json`
- Does not fetch or install the integration
- Use `bootstrap` to install after pinning
- Requires exact version (no "latest")

---

### `swytchcode add workflow <workflow_id>`

Add a verified workflow to `tooling.json`.

**Usage:**
```bash
swytchcode add workflow <workflow_id>
```

**Example:**
```bash
swytchcode add workflow stripe.customer-onboarding
```

**Behavior:**
- Fetches workflow definition from registry
- Merges workflow tools into `tooling.json`
- Requires `project_uuid` in `tooling.json`
- Verified workflows are trusted (no proposal step)

---

## Execution Commands

### `swytchcode exec`

Execute a tool via the Swytchcode kernel. **Single execution path** — all tools must be executed through this command.

**Usage:**
```bash
swytchcode exec [flags]
```

**Flags:**
- `--allow-raw`: Allow execution of raw methods (required for tools starting with `raw.`)

**Input:** JSON from stdin
```json
{
  "tool": "stripe.createCustomer",
  "args": {
    "email": "user@example.com",
    "name": "John Doe"
  }
}
```

**Output:** JSON to stdout
```json
{
  "customer_id": "cus_1234567890",
  "email": "user@example.com"
}
```

**Example:**
```bash
# Verified tool
echo '{"tool":"stripe.createCustomer","args":{"email":"user@example.com"}}' | swytchcode exec

# Raw method (requires --allow-raw)
echo '{"tool":"raw.stripe.customers.search","args":{"query":"email:user@example.com"}}' | swytchcode exec --allow-raw
```

**Behavior:**
- **Never prompts** — fully non-interactive
- **Never calls registry** — uses only local `tooling.json` and Wrekenfiles
- Reads JSON from stdin, writes JSON to stdout
- Stable exit codes (see below)
- Requires `--allow-raw` for raw methods

**Exit Codes:**
- `0`: Success
- `1`: Invalid input
- `2`: Tool not found
- `3`: Auth error (missing env vars)
- `4`: SDK failure
- `5`: Internal error

**Important:**
- **In CI, `--allow-raw` must never be used.** Raw methods are not allowed in CI by default.

---

## Configuration Commands

### `swytchcode config`

Show effective configuration (registry URL and source). Makes env overrides visible.

**Usage:**
```bash
swytchcode config
```

**Output:** JSON to stdout
```json
{
  "registry_url": {
    "effective": "https://api.swytchcode.com",
    "source": "env"
  },
  "mode": "production",
  "version": "1.0.0"
}
```

**Source values:**
- `"env"`: Overridden by `SWYTCHCODE_REGISTRY_URL` environment variable
- `"tooling"`: From `tooling.json`
- `"default"`: Default value (`https://localhost`)

**Example:**
```bash
swytchcode config
```

**Use case:** Debug "why is this pointing somewhere else?" or confirm no override is active.

---

### `swytchcode mode [production|sandbox]`

Set or display the execution mode for the project.

**Usage:**
```bash
# Display current mode
swytchcode mode

# Set mode
swytchcode mode production
swytchcode mode sandbox
```

**Modes:**
- `production`: Use production credentials and enforce strict policies
- `sandbox`: Use sandbox/test credentials and allow experimental features

**Behavior:**
- Mode is stored in `tooling.json`
- Affects credential selection and policy enforcement
- Default mode is `production` if not explicitly set
- Can be set during `swytchcode init` or changed later

**Example:**
```bash
swytchcode mode          # Display current mode
swytchcode mode sandbox  # Set to sandbox mode
```

---

## Command Groups

Commands are organized by role:

| Role | Commands |
|------|----------|
| **Setup** | `init`, `get`, `bootstrap`, `rm`, `upgrade` |
| **Discovery** | `list`, `describe` |
| **Authorization** | `validate`, `apply`, `add integration`, `add workflow` |
| **Execution** | `exec` |
| **Configuration** | `config`, `mode` |

---

## Common Patterns

### Initializing a Project
```bash
swytchcode init --editor=cursor --mode=production --non-interactive
```

### Adding an Integration
```bash
# 1. Pin version in tooling.json
swytchcode add integration stripe@2025-01-10

# 2. Install the integration
swytchcode bootstrap
```

### Adding a Workflow
```bash
swytchcode add workflow stripe.customer-onboarding
```

### Validating and Applying a Proposal
```bash
# 1. Validate (no side effects)
swytchcode validate .swytchcode/proposals/my-workflow.json

# 2. Apply (mutates tooling.json)
swytchcode apply .swytchcode/proposals/my-workflow.json
```

### Executing a Tool
```bash
# Verified tool
echo '{"tool":"stripe.createCustomer","args":{"email":"user@example.com"}}' | swytchcode exec

# Raw method (local dev only)
echo '{"tool":"raw.stripe.customers.search","args":{"query":"test"}}' | swytchcode exec --allow-raw
```

---

## Notes

- All commands that modify state require explicit confirmation or flags in non-interactive mode
- Commands that read from `tooling.json` will fail if it doesn't exist (run `swytchcode init` first)
- `project_uuid` must be present in `tooling.json` for registry operations
- In CI, use `--non-interactive` flag and provide all required parameters via flags
- Raw methods (`raw.*`) require `--allow-raw` flag and are **not allowed in CI**
