# Determinism Guarantees

- **Same inputs → same observable behavior.** No prompt-based execution; no hidden branching.
- **Exec input/output** is JSON (or raw when requested). No free-form text interpretation.
- **Failures are explicit** – exit codes and structured error payloads. No silent retries.
- **No agent-side policy interpretation** – if a tool isn't in tooling.json or an integration isn't found, exec fails; agents don't substitute or retry.
