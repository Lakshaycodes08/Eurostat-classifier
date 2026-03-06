# Hardcoded values inventory

This file catalogs notable hardcoded values in the Swytchcode CLI codebase and where they appear. It’s grouped by area so we can decide what should remain fixed and what should be configurable.

## 1. Core configuration and API endpoints

- **`internal/constants/constants.go`**
  - `Version = "1.0.2"` – build-time constant overridden by Goreleaser via `-X` in `.goreleaser.yml`.
  - `RegistryURL = "https://api-v2.swytchcode.com"` – default registry base URL used by the registry client.
  - `MCPBearerToken = "swytchcode-mcp-token"` – fixed bearer token for MCP HTTP transport (marked as temporary).

- **`internal/registry/config.go`**
  - `ConfigFromProjectRoot` returns a config with `BaseURL` set from `constants.RegistryURL` (no env override).

- **`internal/cli/check.go`, `internal/cli/login.go`, `internal/cli/inspect.go`, `internal/cli/upgrade.go`, `internal/auth/auth.go`**
  - All default `SWYTCHCODE_API_URL` to `https://api-v2.swytchcode.com` when the env var is empty.
  - Auth refresh in `auth.go` also defaults to `https://api-v2.swytchcode.com` if `SWYTCHCODE_API_URL` is unset.

**Implication:** `https://api-v2.swytchcode.com` is the baked-in production backend. Dev/staging require setting `SWYTCHCODE_API_URL` or building with different constants.

## 2. Localhost and sandbox defaults

- **`internal/kernel/request_test.go`**
  - Uses `http://localhost` as the base URL in tests for request-building logic.

- **`internal/commands/get.go` / `internal/commands/bootstrap.go`**
  - Comment: “Use endpoints directly from bundle (use http://localhost if empty)”.
  - If bundle endpoints are empty, sandbox and production endpoints fall back to `http://localhost`.

**Implication:** A missing endpoint in the bundle produces a localhost base URL. This is convenient for local dev but should be clearly documented and may deserve a configurable default or validation rule.

## 3. GitLab project / releases / CI

- **`.goreleaser.yml`**
  - `release.gitlab.owner = "swytchcode"` and `name = "cli"` – hardcoded GitLab project for releases.

- **`.gitlab-ci.yml`**
  - Uses `https://gitlab.com/swytchcode/cli` in:
    - CI/CD settings link for `GITLAB_TOKEN`.
    - `git remote set-url origin "https://oauth2:${GITLAB_TOKEN}@gitlab.com/swytchcode/cli.git"`.

- **`install.sh` / `install.ps1`**
  - Default `RELEASE_BASE` / `$ReleaseBase` = `https://gitlab.com/swytchcode/cli/-/releases`.
  - Uses `/permalink/latest/downloads` or `/vX.Y.Z/downloads` for artifacts and `checksums.txt`.

**Implication:** The CLI and installer are tightly bound to the `swytchcode/cli` GitLab project. Forks or mirrors must override `BASE_URL` / `ReleaseBase` and CI config.

## 4. Public domains and marketing links

- **`pages/index.html`**
  - Install one-liners:
    - `curl -fsSL https://cli.swytchcode.com/install.sh | sh`
    - `irm https://cli.swytchcode.com/install.ps1 | iex`
  - Links:
    - `https://swytchcode.com` – main site
    - `https://gitlab.com/swytchcode/cli/-/releases` – releases
    - `https://gitlab.com/swytchcode/cli/-/wikis/home` – wiki
    - `https://docs.swytchcode.com` – docs
    - `https://blog.swytchcode.com/` – blog
    - `https://wreken.com` – Wreken spec

- **`README.md`, `wiki/*.md`**
  - Older references to `https://swytchcode.gitlab.io/cli/...` remain in some wiki pages and FAQ entries.
  - README points to:
    - `https://cli.swytchcode.com/install.sh` / `.ps1` for install.
    - `https://cli.swytchcode.com/` for Pages.
    - `https://swytchcode.com` for the main site.

**Implication:** These are expected to be stable marketing/docs URLs; they should be updated only when domains move. The main risk is inconsistency (old `swytchcode.gitlab.io` links vs new `cli.swytchcode.com`), not configurability.

## 5. MCP and telemetry

- **`internal/mcp/transport.go`**
  - Authorization header for HTTP transport is always `Bearer ` + `constants.MCPBearerToken`.

- **`internal/cli/check.go`, `internal/cli/inspect.go`, `internal/cli/upgrade.go` and `internal/telemetry/telemetry.go`**
  - Telemetry uses `SWYTCHCODE_API_URL` (default `https://api-v2.swytchcode.com`) and includes `constants.Version` in events.

**Implication:** MCP HTTP transport uses a fixed bearer token; for production this should likely become configurable (env var or config file) and documented.

## 6. Legacy docs (optional cleanup)

- **`cli.md`, `commands.md`, `CLI_CHANGES.md`** (if still intended to ship)
  - Contain examples referencing:
    - `https://app.swytchcode.com/cli-auth?...`
    - `SWYTCHCODE_API_URL=http://localhost:80`
    - `https://app.swytchcode.com/billing`

These appear to be early/auxiliary docs; if they are not meant for end users anymore, they can be archived or moved under a `legacy/` docs section.

---

This inventory is the basis for refactor proposals. The main candidates for change are:

- Making registry and backend URLs configurable via a small config layer (with clear env var overrides).
- Moving `MCPBearerToken` out of code into configuration.
- Consolidating public URLs so install scripts, Pages, README, and wiki all point to the same canonical endpoints.

