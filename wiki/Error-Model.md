# Error Model

- **Tool not found** – Canonical ID not in tooling.json or not resolvable. Fatal; exec exits non-zero.
- **Integration not installed** – Referenced integration not present under `.swytchcode/integrations/`. Fatal. Agent should run `swytchcode get <project>` or `bootstrap`.
- **Execution failure** – Tool ran but returned an error. Exit code and stderr/JSON carry the reason.
- **No hidden retries** – Swytchcode does not retry on failure. Callers (agents) should not retry with different tools; they should surface the error.

Exit codes and JSON schema are documented in [CLI Reference](CLI-Reference) and the `swytchcode exec` section.
