# Swytchcode CLI – Refactor Proposals

This document summarizes concrete refactor ideas based on the current codebase and the hardcoded inventory. It’s organized by area and is intended as a roadmap; none of these changes are required for correctness right now, but they would improve maintainability and configurability.

## 1. Configuration & environment

### 1.1 Small config layer

**Problem:** Core URLs and tokens are spread across `internal/constants/constants.go`, CLI commands, and auth helpers:

- `constants.RegistryURL` (registry base URL).
- Default `SWYTCHCODE_API_URL` = `https://api-v2.swytchcode.com` in several commands and `auth.go`.
- `constants.MCPBearerToken` used directly by `internal/mcp/transport.go`.

**Proposal:**

- Introduce a lightweight `internal/config` package that centralizes:
  - `RegistryBaseURL()` – returns the registry URL, resolving in order:
    1. Build-time constant (production).
    2. Optional env override (`SWYTCHCODE_REGISTRY_URL`) for dev/staging.
  - `BackendAPIURL()` – returns the CLI backend base URL, resolving in order:
    1. `SWYTCHCODE_API_URL` env var (already used).
    2. Build-time default (current `https://api-v2.swytchcode.com`).
  - `MCPBearer()` – returns the MCP HTTP bearer token, resolving in order:
    1. Env var (e.g. `SWYTCHCODE_MCP_BEARER`).
    2. Build-time default (current `MCPBearerToken` value).

- Update:
  - `internal/registry/config.go` to call `config.RegistryBaseURL()`.
  - `auth.go`, `check.go`, `login.go`, `inspect.go`, `upgrade.go` to use `config.BackendAPIURL()` instead of re-implementing the default in each file.
  - `internal/mcp/transport.go` to use `config.MCPBearer()` instead of hardcoding `MCPBearerToken`.

**Benefits:** Single place to adjust URLs and tokens; clearer dev/staging overrides; smaller changes when domains move.

## 2. Install scripts (especially Windows)

### 2.1 Improve Windows install robustness

**Current behavior (`install.ps1`):**

- Assumes:
  - Modern PowerShell with `Invoke-WebRequest`, `Expand-Archive`, `Get-FileHash`.
  - Execution policy allows running downloaded scripts.
- Fails if:
  - TLS/SSL or proxies block `Invoke-WebRequest`.
  - Execution policy blocks piped `iex`.
  - PATH update fails (user shell not reloaded or PATH env is unusual).

**Proposed improvements:**

- **More explicit error messaging:**
  - Distinguish:
    - Download failure (network/TLS/404).
    - Checksum mismatch.
    - Archive extraction failure.
    - PATH update failure.
  - Print a single summary line at the end: “Installed to `<path>`; open a new terminal or run `setx PATH` … if not on PATH”.

- **Execution policy / non-piped mode:**
  - Document a non-piped usage:
    - `powershell -NoProfile -ExecutionPolicy Bypass -File install.ps1` (for users who download the script first).
  - Optionally detect when running under restricted policy and suggest manual steps:
    - On `Invoke-WebRequest` or `iex` failures, print guidance: “If execution policy blocks this, download install.ps1 and run with `-ExecutionPolicy Bypass` or install manually from Releases.”

- **PowerShell version compatibility:**
  - Keep dependencies minimal:
    - `Invoke-WebRequest` (available in Windows PowerShell 5+).
    - `Expand-Archive`, `Get-FileHash` (standard in PS 5+; for older versions, document manual install as fallback).

These changes can remain in `install.ps1` without changing the contract with users (same one-liner command).

### 2.2 Align URLs and docs

- Ensure:
  - `install.sh` / `install.ps1` default to GitLab Releases (`https://gitlab.com/swytchcode/cli/-/releases`) and optionally allow `BASE_URL`/`ReleaseBase` overrides.
  - `pages/index.html` and `README.md` reference the **canonical** install URLs (`https://cli.swytchcode.com/install.sh` / `.ps1`) and Releases page.
  - Wiki pages that still point at `https://swytchcode.gitlab.io/cli/...` are updated to either `https://cli.swytchcode.com/...` or the GitLab Releases page.

## 3. CLI surface and consistency

### 3.1 Command help & flags

**Problem:** As commands were added (`check`, `login`, `logout`, `whoami`, `inspect`, `upgrade`), help texts and flag behavior may not all follow the same conventions (e.g. non-interactive usage, JSON output).

**Proposal:**

- Standardize:
  - Each command’s `Short` and `Long` description uses the same style and mentions relevant env vars (esp. `SWYTCHCODE_API_URL`, `SWYTCHCODE_TOKEN`, `SWYTCHCODE_PROJECT_UUID`).
  - Commands used in CI (e.g. `check`, `bootstrap`, `exec`) clearly document non-interactive usage and exit codes.
  - If any commands need JSON output modes (beyond `exec`), add `--json` consistently and document it.

- Align README’s “Commands at a glance” and the new `docs/cli-reference.md` with the actual CLI help text (source of truth is the code).

### 3.2 Exit codes and error messages

**Problem:** Exit codes are centralized in `internal/kernel/errors.go` for `exec`, but other commands (`check`, `login`, etc.) exit directly with numeric codes.

**Proposal (lightweight):**

- Add a short section in `docs/cli-reference.md` summarizing:
  - Exec exit codes from `errors.go`.
  - High-level behavior for other commands (e.g. `check` exits with 1 on breaking changes, 2 on errors).
- Optionally, add small constants in packages where exit codes are reused (e.g. for `check` error vs breaking change) to avoid magic numbers.

## 4. Docs structure & single source of truth

### 4.1 Prefer `docs/` as the technical source, mirror to Wiki

**Current split:**

- `wiki/` contains conceptual docs (execution model, tooling.json spec, integrations, FAQ, roadmap).
- README contains a lot of CLI detail.
- GitLab Wiki will mirror the repo manually.

**Proposal:**

- Use `docs/` as the primary technical documentation source:
  - `docs/architecture.md`, `docs/execution-model.md`, `docs/config-spec.md`, `docs/cli-reference.md`, `docs/mcp-and-integrations.md`, `docs/install-upgrade.md`.
- Keep Wiki pages in sync by copy/paste or by simple automation later.
- Keep README as a high-level entrypoint:
  - Short install section.
  - “Commands at a glance”.
  - Links into `docs/` and the public Wiki.

This reduces duplication and makes it easier to reason about behavior when refactoring.

## 5. Summary of priority changes

If you want to keep changes focused, the highest-impact refactors are:

1. **Config layer:** Introduce `internal/config` and route registry/back-end URLs and MCP bearer token through it, with env var overrides.
2. **Windows install hardening:** Improve `install.ps1` error messages and document a non-piped install path; add a small “Windows troubleshooting” section to `docs/install-upgrade.md`.
3. **URL consolidation:** Align install URLs across `install.sh`, `install.ps1`, `pages/index.html`, README, and wiki; update any lingering `swytchcode.gitlab.io` links.
4. **CLI & docs sync:** Standardize help texts and document exit codes and behaviors in the new CLI reference.

