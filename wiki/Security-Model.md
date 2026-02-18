# Security Model

- **Local execution** – Tools run on your machine. No execution is delegated to remote services.
- **No runtime registry** – `swytchcode exec` uses only tooling.json and local integrations. No "phone home" during execution.
- **Explicit allow-list** – Only tools in `tooling.json` can be executed.
- **No hidden retries** – Failures surface as exit codes / JSON; no silent retries or agent-side interpretation of policy.
- **AI as planner, not executor** – Agents call `swytchcode exec`; they don't run arbitrary code or interpret policy.

See also [Pages → Security](https://swytchcode.gitlab.io/cli/docs/security.html).
