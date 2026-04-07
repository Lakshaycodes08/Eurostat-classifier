# Security Model

- **Local execution** – Tools run on your machine. No execution is delegated to remote services.
- **No runtime registry** – `swytchcode exec` uses only tooling.json and local integrations. No "phone home" during execution.
- **Explicit allow-list** – Only tools in `tooling.json` can be executed.
- **Integration base URLs** – Tool HTTP calls require **`https://`** unless the base URL is **`http://`** on loopback only (`localhost`, `127.0.0.1`, `::1`). See [Config spec → manifest](https://gitlab.com/swytchcode/cli/-/blob/main/docs/config-spec.md) (HTTPS and HTTP base URLs).
- **`SWYTCHCODE_INSECURE=1`** – Disables TLS certificate verification for HTTP clients (local dev / self-signed only). Registry traffic is **blocked** in CI when `CI`, `GITHUB_ACTIONS`, or `GITLAB_CI` is set; a warning is printed outside CI. This flag does not allow arbitrary non-loopback `http://` execution URLs.
- **Retries** – Kernel applies manifest `execution_policy` (retries, timeouts, idempotency); agents should not improvise retries on top of exec failures.
- **AI as planner, not executor** – Agents call `swytchcode exec`; they don't run arbitrary code or interpret policy.

Consolidated doc in-repo: [docs/security.md](https://gitlab.com/swytchcode/cli/-/blob/main/docs/security.md). Run **`swytchcode doctor`** (or MCP **`swytchcode_doctor`**) for a quick local posture check.

See also [Pages → Security](https://swytchcode.gitlab.io/cli/docs/security.html).
