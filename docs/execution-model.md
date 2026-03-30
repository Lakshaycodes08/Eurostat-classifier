# Execution model

How **`swytchcode exec`** turns a canonical tool ID into an HTTP call and JSON output. For module layout and diagrams, see [architecture.md](architecture.md).

## Single entrypoint

Only **`kernel.Execute`** ([`internal/kernel/executor.go`](../internal/kernel/executor.go)) runs methods and workflows. The CLI and MCP both funnel into it. The kernel does **not** call the registry for **single-method** execution (offline, deterministic); **workflows** may fetch definitions and bundles from the registry when not fully defined locally.

## Request shape

- **Stdin (JSON):** `{ "tool": "<canonical_id>", "args": { ... } }` ([`ExecRequest`](../internal/kernel/executor.go)).
- **CLI mode:** Cobra builds the same structure and passes it to the kernel as JSON.

Stdin is read in full, then unmarshaled; invalid JSON produces a structured stderr error (with optional hints for Windows / `&` in payloads — see [windows-guide.md](windows-guide.md)).

## Pipeline (single method)

1. Resolve **project root** ([`util.ProjectRoot`](../internal/util/fs.go)).
2. **Resolve tool** from `tooling.json` ([`ResolveTool`](../internal/kernel/resolver.go)) — uses [`util.LoadToolingJSON`](../internal/util/tooling.go) for a single read/parse path.
3. **Load bundle** ([`LoadIntegrationBundle`](../internal/kernel/bundle.go)): Wrekenfile + metadata under `.swytchcode/integrations/`.
4. **Resolve method** in the Wreken `METHODS` section ([`ResolveMethod`](../internal/kernel/bundle.go)).
5. **Base URL** from `manifest.json` ([`GetBaseURL`](../internal/kernel/manifest.go)) and **validate** scheme/host ([`ValidateExecutionBaseURL`](../internal/kernel/base_url_validate.go)).
6. **Validate inputs** against the tool schema ([`ValidateInput`](../internal/kernel/validator.go)).
7. **Build request** ([`BuildRequest`](../internal/kernel/request.go)) — JSON or form encoding from `Content-Type`.
8. **Execute HTTP** ([`ExecuteHTTP`](../internal/kernel/http_exec.go)) — retries, timeout, and idempotency from manifest `execution_policy` (see [config-spec.md](config-spec.md)).
9. **Normalize output** to stdout JSON (or raw / dry-run).

## Workflows

If the tool type is **workflow**, the kernel either runs **local steps** from `tooling.json` or fetches the workflow definition from the registry, then runs steps sequentially ([`chain.go`](../internal/kernel/chain.go)). Failed workflows return a **non-zero** exit code while still emitting step JSON.

## Trust and policy

- **`tooling.json`** is the allow-list of tools.
- **Wrekenfiles** define what HTTP calls look like.
- **Agents** should not bypass `exec` or reinterpret policy.

See [security.md](security.md) and the kernel section of [architecture.md](architecture.md).
