# Swytchcode Wiki – Home

## What Swytchcode is

Swytchcode is a **secure execution layer** for AI agents and CLIs. Only `swytchcode exec` runs tools; editors and agents call it – they don't run arbitrary code.

- **tooling.json** defines what is *allowed* (the contract).
- **Wrekenfiles** define what is *possible* (METHODS, WORKFLOWS, STRUCTS) for each integration.

## What it is not

- Not an agent runtime that interprets prompts to decide what to run.
- Not a service that executes code in the cloud.
- Not a replacement for your editor – it sits between the editor/agent and the tools.

## Design philosophy

- **Explicit over implicit** – No hidden execution paths.
- **Deterministic over probabilistic** – Same inputs, same behavior.
- **Contracts over prompts** – tooling.json and Wrekenfiles are the contract; agents don't interpret policy.

## Install and quick start

**Install (one command):** [Pages → Install](https://swytchcode.gitlab.io/cli/install/)

```bash
curl -fsSL https://swytchcode.gitlab.io/cli/install.sh | sh
```

Then in your project: `swytchcode init`, `swytchcode get <project>`, `swytchcode add <canonical_id>`, `swytchcode exec <canonical_id>`.

For depth and specs, use the sidebar: **Core Concepts**, **CLI Reference**, **Integrations**, **Security Model**.
