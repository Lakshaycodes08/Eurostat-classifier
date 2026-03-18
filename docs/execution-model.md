# Swytchcode CLI – Execution Model

Swytchcode treats AI as a **planner**, not an executor. Only `swytchcode exec` runs tools. This document explains how execution works end-to-end.

## Single entrypoint: `swytchcode exec`

- CLI: `swytchcode exec <canonical_id> [flags]`
- JSON stdin: `{"tool":"<canonical_id>","args":{...}}`
- MCP: tools like `swytchcode_exec` call into the same kernel function.

In all cases, execution flows into [`kernel.Execute`](internal/kernel/executor.go), which implements a strict pipeline and never calls the registry at runtime.

## Why exec is the only path

From `internal/kernel/executor.go`:

- **Authority** – One place to:
  - Enforce `tooling.json` (what is trusted).
  - Resolve canonical IDs to integrations and modes.
  - Validate inputs and build HTTP requests.
- **Determinism** – Same `ExecRequest` → same behavior:
  - No prompt-based branching.
  - No hidden retries.
  - No network calls to the registry during execution.
- **Failure semantics** – Exit codes and JSON errors are defined centrally in [`internal/kernel/errors.go`](internal/kernel/errors.go).

Agents and editors should be **read-only** over the tool list: they may use `swytchcode list` and `swytchcode info`, but must not execute tools directly or interpret policy.

## ExecRequest input

The kernel expects a JSON object on stdin:

```json
{
  "tool": "api.cluster.create",
  "args": {
    "name": "my-cluster",
    "region": "eu-west-1"
  }
}
```

In CLI mode, `internal/cli/exec.go` converts flags into this shape:

- `--body file.json` → `args["body"] = <parsed JSON>`.
- `--input key=value` → `args["key"] = "value"`.
- `--param key=value` → `args["params"][key] = "value"`.
- `--header key=value` → `args["headers"][key] = "value"`.

## Execution pipeline

The execution pipeline (from `internal/kernel/executor.go`) is:

1. **Parse request** – Read JSON from stdin into `ExecRequest`.
2. **Detect project root** – Use `util.ProjectRoot()` if not provided explicitly.
3. **Enforce raw policy** – If `tool` starts with `raw.`, require `--allow-raw`.
4. **Resolve tool** – Call `ResolveTool` to read `.swytchcode/tooling.json` and find the tool entry.
5. **Load integration bundle** – Call `LoadIntegrationBundle` to read Wrekenfile + methods/workflows from `.swytchcode/integrations/...`.
6. **Resolve method/workflow** – Call `ResolveMethod` to locate the specific method/workflow in the Wreken `METHODS` section.
7. **Get base URL** – Call `GetBaseURL` to read `.swytchcode/integrations/manifest.json` and select the correct endpoint for the tool’s mode (`production` or `sandbox`).
8. **Validate input** – Call `ValidateInput` to check that `args` match the input schema.
9. **Build HTTP request** – Call `BuildRequest` to produce an HTTP request object from the Wreken spec, base URL, and args.
10. **Execute or dry-run**:
    - If `--dry-run`: output a description of the request without calling the target API.
    - Otherwise: execute the HTTP request (`ExecuteHTTP`) and capture the response.
11. **Normalize output**:
    - Raw: `OutputRawResponse` (for debugging / non-JSON APIs).
    - JSON: `OutputJSONResponse` includes response details and request URL.

At no point does the kernel call the registry or modify `.swytchcode/`; it only reads local files.

## Tool resolution via tooling.json

`ResolveTool` in [`internal/kernel/resolver.go`](internal/kernel/resolver.go):

- Reads `.swytchcode/tooling.json`.
- Looks up the tool in `tooling.tools[canonical_id]`.
- Builds a `Tool` struct with:
  - `CanonicalID` – the requested ID (e.g. `api.cluster.create`).
  - `Integration` – string like `project.library@version`.
  - `Type` – `"method"` or `"workflow"`.
  - `Summary`, `Desc` – description fields.
  - `Inputs` – input schema (with STRUCTs resolved earlier when the tool was added).
  - `Mode` – top-level `mode` from `tooling.json` (defaults to `"production"`).

If the tool is not present, `ResolveTool` returns an error instructing the user to run `swytchcode add <canonical_id>`.

## Bundles, Wrekenfiles, and manifest

The **registry** is responsible for providing integration bundles. The kernel uses the artifacts on disk:

- **Bundles** (`internal/kernel/bundle.go`):
  - `LoadIntegrationBundle` reads:
    - `wrekenfile.yaml` – Wrekenfile spec with `METHODS`, `WORKFLOWS`, `STRUCTS`.
    - `methods.json` / `workflows.json` – denormalized listings used by `list` and `info`.
  - `ResolveMethod` finds the method or workflow definition in the Wrekenfile for the canonical ID.

- **Manifest** (`internal/kernel/manifest.go` + `internal/manifest/manifest.go`):
  - `.swytchcode/integrations/manifest.json` maps `project.library` to:
    - `sandbox_endpoint`
    - `production_endpoint`
    - `version`, `methods`, `workflows`, and optional `auth`.
  - `GetBaseURL`:
    - Reads `manifest.json`.
    - Picks `SandboxEndpoint` or `ProductionEndpoint` based on the tool’s mode.
    - Errors if the integration or endpoint is missing.

These files are created and maintained by `swytchcode get` and `swytchcode bootstrap` (via `internal/commands`), not by the kernel.

## Error model and exit codes

The kernel’s public contract is defined in [`internal/kernel/errors.go`](internal/kernel/errors.go):

- Exit codes:
  - `0` – OK.
  - `1` – Invalid input (bad JSON, missing `tool`, validation failure, bad flags).
  - `2` – Tool not found (missing in `tooling.json` or integration bundle).
  - `3` – Auth error (reserved for auth-related failures).
  - `4` – SDK failure (e.g. network errors when calling the target API).
  - `5` – Internal error (unexpected conditions, project root detection failure, etc).

- JSON error shape:

```json
{ "error": "message" }
```

All kernel failures write this shape to stderr so callers (CLI, MCP, editors) can parse errors uniformly.

## Policy: apps must not interpret policy

Guidelines for agents and editors:

- Do **not**:
  - Execute tools directly (e.g. by calling the target API themselves).
  - Retry with different tools if `exec` fails.
  - Decide “allowed” tools by inspection of Wrekenfiles; use `tooling.json`.

- Do:
  - Use `swytchcode list` / `swytchcode info` to discover tools.
  - Call `swytchcode exec` for execution.
  - Surface kernel errors to users (using exit codes and JSON error messages).

## Retries and idempotency

Retries and idempotency belong **inside** Swytchcode or the underlying integration:

- If an integration is idempotent, it may handle retries at the HTTP level.
- The kernel does not retry failed requests automatically.
- Agents should not attempt to retry with different tools or different endpoints; they should surface the error and let humans decide.

## Workflow execution

When the tool's `type` is `"workflow"`, the kernel runs each step in the `steps` array sequentially:

1. The first step's args come from the original `ExecRequest.Args`.
2. After each step, the response `data` fields are **merged** into a shared `mergedArgs` map.
3. Each subsequent step starts with `mergedArgs` as its args — so outputs from step N are automatically available as inputs for step N+1.
4. **Path parameter substitution**: `BuildRequest` substitutes `{placeholder}` tokens in the endpoint URL from both `args["params"]` **and** top-level args (i.e. merged outputs from prior steps). This means a step that returns `project_uuid` can be used directly as a URL path parameter `{project_uuid}` in the next step without any manual wiring.

**Output format**: the final output of a workflow exec is a JSON object with a `steps` array, each element containing the result of the corresponding step:

```json
{
  "steps": [
    { "request": {...}, "status_code": 201, "data": { "project_uuid": "abc-123" } },
    { "request": {...}, "status_code": 200, "data": { ... } }
  ]
}
```

## Integration not found / missing tooling

If an integration is not installed or a tool is not present in `tooling.json`:

- `ResolveTool` or `LoadIntegrationBundle` returns an error.
- `Execute`:
  - Writes `{ "error": "..." }` to stderr.
  - Exits with `ExitCodeToolNotFound` (2) or `ExitCodeInternalError` (5), depending on the failure.

Agents should treat this as a **hard stop** (no fallback execution).

