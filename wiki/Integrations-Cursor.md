# Integrations: Cursor

Cursor uses Swytchcode so the agent delegates execution to `swytchcode exec` instead of running tools directly.

## Setup

1. Install Swytchcode: [Install](https://swytchcode.gitlab.io/cli/install/)
2. In the project: `swytchcode init --editor=cursor`
3. Add tools: `swytchcode add <canonical_id>`

## What gets installed

- **`.cursor/rules/swytchcode.mdc`** – Rule that tells Cursor to use `swytchcode exec` for tool execution. The agent reads tooling.json and integrations; it calls `swytchcode exec` with the chosen canonical_id and args.

## Behavior

- Cursor (and its MCP client, if used) can call `swytchcode list`, `swytchcode info`, and `swytchcode exec`.
- Only tools in `tooling.json` can be executed. The agent is read-only over the allow-list; it does not interpret policy.
- On "tool not found" or "integration not installed", exec fails and the agent should surface the error, not retry with something else.
