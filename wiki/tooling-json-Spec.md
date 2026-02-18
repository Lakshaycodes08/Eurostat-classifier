# tooling.json Spec

`tooling.json` is the **single source of truth** for what is allowed in this project.

## Location

`.swytchcode/tooling.json` (created by `swytchcode init`).

## Top-level fields

- **version** – Schema version.
- **mode** – `production` or `sandbox`.
- **integrations** – Map of integration specs to fetch (e.g. `weaviate@lyrid.v1`). Used by `swytchcode bootstrap`.
- **tools** – Map of canonical IDs to integration refs. Only tools listed here can be executed via `swytchcode exec`.

## Semantics

- **integrations**: Declares which bundles (and versions) are used. `swytchcode get` / `bootstrap` install them under `.swytchcode/integrations/`.
- **tools**: Allow-list. A canonical ID must appear in `tools` and resolve to an installed integration for `exec` to run it.

Human-readable, machine-enforced. No implicit allowlists.
