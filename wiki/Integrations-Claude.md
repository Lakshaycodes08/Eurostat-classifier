# Integrations: Claude

Claude uses Swytchcode so execution goes through `swytchcode exec` instead of ad-hoc tool runs.

## Setup

1. Install Swytchcode: [Install](https://swytchcode.gitlab.io/cli/install/)
2. In the project: `swytchcode init --editor=claude`
3. Add tools: `swytchcode add <canonical_id>`

## What gets installed

- **`CLAUDE.md`** at repo root – Instructions for Claude to use the tool list and call `swytchcode exec` for execution. Claude reads tooling.json and integrations; it invokes `swytchcode exec` with the chosen canonical_id and args.

## Behavior

- Claude is given the contract (tooling.json + available tools) and the single entrypoint: `swytchcode exec`.
- Only tools in `tooling.json` can be executed. Claude does not interpret policy; it requests execution and respects exit codes and JSON output.
- On failure, Claude should report the error and stop, not improvise or retry with a different tool.
