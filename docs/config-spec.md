# Swytchcode CLI – tooling.json & manifest spec

Swytchcode uses two primary configuration files:

- `tooling.json` – project-level contract for what tools are allowed.
- `manifest.json` – registry-derived metadata for integrations (endpoints, versions, counts).

This document describes their structure and how the CLI and kernel use them.

## tooling.json

### Location

- Path: `.swytchcode/tooling.json`
- Created by: `swytchcode init`

### Top-level fields

- `version` – Schema version (string).
- `mode` – Execution mode: `"production"` or `"sandbox"`.
- `integrations` – Map of integration specs to fetch. Keys are `project.library`, values are objects with `version`.
- `tools` – Map of canonical IDs (e.g. `api.cluster.create`) to tool entries.

Example:

```json
{
  "version": "1",
  "mode": "production",
  "integrations": {
    "weaviate.lyrid": { "version": "v1" }
  },
  "tools": {
    "api.cluster.create": {
      "integration": "weaviate.lyrid@v1",
      "type": "method",
      "summary": "Create a cluster",
      "desc": "Creates a cluster in the configured region.",
      "inputs": {
        "name": { "type": "string" },
        "region": { "type": "string" }
      }
    }
  }
}
```

### Semantics

- **integrations**:
  - Declares which bundles (and versions) are used in this project.
  - `swytchcode bootstrap` reads `integrations` and ensures the corresponding bundles exist under `.swytchcode/integrations/{project}/{library}/{version}/` by calling the registry.

- **tools**:
  - Allow-list of tools that can be executed.
  - Each key is a canonical ID (e.g. `api.cluster.create`).
  - Each value references an integration (`project.library@version`) and includes metadata:
    - `type`: `"method"` or `"workflow"`.
    - `summary` / `desc`: human-readable descriptions.
    - `inputs`: resolved input schema (STRUCTs expanded into concrete fields).

**Important:** A tool must be in `tools` *and* its integration must be installed locally for `swytchcode exec` to run it successfully.

### How the CLI uses tooling.json

- `swytchcode init`:
  - Creates a minimal `tooling.json` with default `version`, `mode`, and empty `integrations`/`tools` maps.

- `swytchcode get`:
  - Does **not** modify `tooling.json`; it only fetches bundles.

- `swytchcode add`:
  - Reads integration bundles and Wrekenfiles.
  - Adds selected tools (methods/workflows) to `tools` with resolved input/output schemas.

- `swytchcode bootstrap`:
  - Reads `integrations` to know which bundles to ensure are present locally.

- `swytchcode list` / `swytchcode info`:
  - Use both `tooling.json` and `.swytchcode/integrations` to provide local state and detailed tool metadata.

### How the kernel uses tooling.json

[`internal/kernel/resolver.go`](internal/kernel/resolver.go):

- `ResolveTool`:
  - Reads `.swytchcode/tooling.json`.
  - Loads top-level `mode` (default `"production"`).
  - Looks up the requested canonical ID in `tools`.
  - Builds a `Tool` struct with:
    - `CanonicalID`
    - `Type`
    - `Integration` (`project.library@version`)
    - `Summary`, `Desc`, `Inputs`
    - `Mode` (copied from top-level `mode`)

If the tool is missing, it returns an error instructing the user to run `swytchcode add <canonical_id>`.

## manifest.json

### Location

- Path: `.swytchcode/integrations/manifest.json`
- Managed by: `internal/manifest/manifest.go`, updated by `RunGet` and `RunBootstrap` in `internal/commands`.

### Structure

`manifest.json` is a JSON object mapping `"project.library"` to an entry:

```json
{
  "weaviate.lyrid": {
    "version": "v1",
    "sandbox_endpoint": "http://localhost:8080",
    "production_endpoint": "https://api.weaviate.lyrid.dev",
    "methods": 42,
    "workflows": 3,
    "auth": {
      "type": "api_key",
      "header": "Authorization"
    }
  }
}
```

Fields (from `internal/manifest/manifest.go`):

- `version` – Integration version (string, required).
- `sandbox_endpoint` – Base URL for sandbox mode.
- `production_endpoint` – Base URL for production mode.
- `methods` – Number of methods for this integration (int).
- `workflows` – Number of workflows (int).
- `auth` – Optional auth metadata (arbitrary JSON).

### How manifest.json is written

`manifest.UpdateEntry` in `internal/manifest/manifest.go`:

- Reads existing `manifest.json` (or initializes an empty map).
- Updates/creates an entry for a given `projectLibrary` key:
  - Version, endpoints, counts, auth.
- Writes the updated map back to `manifest.json`.

`RunGet` and `RunBootstrap`:

- After fetching bundles and listing workflows/methods:
  - Compute `methodsCount` and `workflowsCount`.
  - Determine `sandboxEndpoint` and `productionEndpoint` from bundle metadata (falling back to `http://localhost` when endpoints are missing).
  - Call `manifest.UpdateEntry(...)` to persist the data.

### How the kernel uses manifest.json

[`internal/kernel/manifest.go`](internal/kernel/manifest.go):

- `GetBaseURL(projectRoot, integration, mode)`:
  - Reads the manifest via `manifest.Read(projectRoot)`.
  - Looks up the `project.library` entry corresponding to the tool’s integration.
  - Chooses:
    - `SandboxEndpoint` if mode is `"sandbox"`.
    - `ProductionEndpoint` otherwise.
  - Returns an error if:
    - `manifest.json` is missing or malformed.
    - The integration key is missing.
    - The chosen endpoint is empty.

The base URL returned by `GetBaseURL` is then combined with the path from the Wreken method/workflow definition to form the final HTTP request URL.

## Relationship between tooling.json and manifest.json

- `tooling.json`:
  - Declares which integrations and tools are **allowed** in the project.
  - Owned by the user/project; edited by commands like `init` and `add`.

- `manifest.json`:
  - Tracks what the registry knows about those integrations (endpoints, versions, counts, auth).
  - Owned by the registry/CLI; maintained by `get` and `bootstrap`.

During execution:

- `ResolveTool` uses `tooling.json` to find which integration and mode to use.
- `GetBaseURL` uses `manifest.json` to determine where to send the request for that integration and mode.

Together, they form the core contract between the project, the registry, and the kernel.

