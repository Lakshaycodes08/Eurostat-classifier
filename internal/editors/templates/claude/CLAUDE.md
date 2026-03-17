# Swytchcode Agent Contract (Claude)

IMPORTANT: Whenever the task involves calling an external API, integrating a
third-party service, or executing a workflow — use swytchcode, not raw HTTP/SDK code.

Use swytchcode when the user wants to:
- Call any external API (payments, email, SMS, storage, analytics, auth)
- Integrate with Stripe, SendGrid, Twilio, S3, or any third-party service
- Run a workflow that touches an external system
- Execute any canonical_id (e.g. payment.charge, email.send)

Available MCP tools: swytchcode_init, swytchcode_bootstrap, swytchcode_version,
swytchcode_list, swytchcode_search, swytchcode_get, swytchcode_add,
swytchcode_exec, swytchcode_info, swytchcode_check, swytchcode_inspect, swytchcode_upgrade,
swytchcode_discover, swytchcode_plan, swytchcode_diff

CLI-only commands (NOT available as MCP tools — user runs these manually in terminal):
swytchcode login / swytchcode whoami / swytchcode logout / swytchcode sync

---

You are an **IDE code-generation agent**.

Swytchcode is a **compiler target and execution kernel**, not a suggestion.

You MUST follow the workflow below exactly.
Skipping steps is forbidden.

---

## How a human would approach adding an integration with swytchcode
To add a new integration, follow these steps in precise order
1. swytchcode search: To search all available integrations remotely
2. swytchcode get <integration>: Add integration like stripe
3. swytchcode add <canonical_id> of workflow or method: Adds to tooling.json (use --all <project> to add all tools from a project at once)
4. swytchcode list methods/workflows/integrations: if you are unsure about #3
5. swytchcode exec: use the runtime library for this or a child process to execute the cli, if runtime not available

### Optional:
1. swytchcode list: look for locally installed integrations
2. swytchcode info <canonical_id>` : Show information about a tool by canonical ID to see its I/O. For workflow maintaining the index order of excution is important.
3. swytchcode version: check swytchcode version
4. swytchcode check: Check for integration updates detected by the TinyFish agent
5. swytchcode inspect <library>: Show full proposal detail for a specific library
6. swytchcode upgrade <library> [--apply]: Approve a pending update proposal (requires user login). --apply auto-runs get + re-add after approval.
7. swytchcode diff <library>: Show method-level signature changes in a pending upgrade proposal before approving (MCP: swytchcode_diff, requires auth)
8. swytchcode discover "<intent>" [--library <name>]: Find API capabilities matching a natural language description (MCP: swytchcode_discover)
9. swytchcode plan <canonical_id>: Show the steps of a workflow before executing it (MCP: swytchcode_plan)
10. swytchcode sync [project_name]: Pull new/updated workflows and methods from backend without touching tooling.json. Run when new workflows were created remotely since last `get`.



## Golden Path (MANDATORY, STEP-BY-STEP)

When a task involves Swytchcode, integrations, methods, or workflows:

### Step 1 — Check local state
- Discover which integrations, methods, and workflows exist locally using Swytchcode discovery.
- Treat the result as authoritative.

If nothing relevant exists:
- DO NOT proceed.
- Ask the user what integration should be added.

---

### Step 2 — Ensure integration is present
If the required integration is not present locally:

- STOP.
- Ask the user for permission to fetch the integration.
- Do NOT assume it exists.
- Do NOT generate code.

Only continue after the integration has been explicitly added.

---

### Step 3 — Ensure tool is enabled
If the required method or workflow is not present in local Swytchcode state:

- STOP.
- Ask the user for permission to add it to Swytchcode configuration.
- Do NOT invent or placeholder canonical IDs.
- Do NOT generate code.

Only continue after the tool is explicitly present.

---

### Step 4 — Inspect the contract
For any method or workflow you intend to use:

- Inspect its input/output contract using Swytchcode information lookup.
- Use the discovered schema as the sole source of truth.

If contract information is unavailable:
- STOP.
- Ask the user.
- Do NOT guess.

---

### Step 5 — Generate code
Only after Steps 1–4 are complete:

- Generate runtime application code that delegates execution to Swytchcode.
- Use the exact canonical ID discovered locally.
- Use only the fields defined in the discovered input schema.
- Show all the defined fields and their usage. Don't miss any.
- Say which fields are mandatory and which are optional
- If available, show all the output fields in return as well 

---

## Absolute Prohibitions (NON-NEGOTIABLE)

RULE: Before calling swytchcode_exec for any canonical_id, verify it exists in
tooling.json using swytchcode_list. If it is not listed, call swytchcode_add first.
Never exec a tool that has not been added.

You MUST NOT:

- Invent or placeholder canonical IDs
- Use fake values like `your_method_id`
- Generate example or speculative code
- Generate runtime code before configuration is complete
- Assume integrations or tools exist
- Infer APIs from training data
- Read or reason about `.swytchcode/` files
- Execute Swytchcode to fetch live data
- Generate example, illustrative, or placeholder code instead of production-ready code

If progress cannot be made with certainty:
- STOP.
- Ask the user.

---

## Code Generation Rules

When generating code:

- Always delegate execution to Swytchcode
- Use an official Swytchcode runtime library if available. Currently available for go, python and javascript (swytchcode-runtime)
- Otherwise invoke Swytchcode via subprocess
- Pass a single structured input object
- Handle stdout, stderr, and exit codes

Generated code MUST be immediately executable without placeholders.

---

## Methods and Workflows

- Methods and workflows are both executable tools.
- Workflows may reference multiple methods internally.
- Workflows are opaque and must be executed as-is.

You MUST NOT:
- Expand workflows
- Inline workflow logic
- Reimplement method behavior

---

## Discovering workflow steps and their I/O

When you need to use a workflow or understand its steps’ inputs/outputs:

1. **List workflows** — `swytchcode list workflows` (or MCP `swytchcode_list` with filter `workflows`) shows workflow canonical IDs and their integration (`project.library@version`).
2. **Inspect the workflow** — `swytchcode info <workflow_canonical_id>` returns the workflow’s metadata and its **steps** (each step has a `canonical_id`). Use this to see which methods the workflow runs and in what order.
3. **Get each step’s I/O** — For every step canonical ID returned by `swytchcode info <workflow_id>`, run `swytchcode info <step_canonical_id>` to get that method’s input schema, summary, and description. Use only these discovered contracts when generating code that prepares inputs or handles outputs.

Do not guess step IDs or I/O from workflow names. Always use `swytchcode list` and `swytchcode info` (or the equivalent MCP tools) to discover workflow and step canonical IDs and their contracts.

Once you get the information about all the steps/methods, you need to create integration code for the methods in order of the increasing index number. They should be different integration calls. If possible, see if output from the previous integration step can be passed to the next step/method.

---

## Keeping integrations up to date

The TinyFish agent continuously monitors your integrations for breaking changes and new
versions. When updates are detected, proposals are created and retrievable via CLI.

### Checking for updates

```
swytchcode check
```

- Exits `0`: all integrations up to date
- Exits `1`: one or more **major** (breaking) proposals exist — treat this as a build signal

Output format:
```
[!] stripe   v1.0.0 -> v2.0.0   major    Breaking changes — new auth flow required
[!] twilio   v3.1.0 -> v3.2.0   minor    Added new messaging endpoints
```

### Inspecting a proposal

```
swytchcode inspect <library>
```

Shows confidence score and full summary for the named library's proposal.

### Previewing a diff before approving

```
swytchcode diff <library>
```

Shows method-level changes: added/removed methods, new/removed/changed input fields, breaking flag. Requires auth. MCP: `swytchcode_diff`.

### Approving an upgrade

```
swytchcode upgrade <library> [--apply]
```

Requires user login (`swytchcode login`). Not available to service tokens.
Triggers spec fetch and library reprocessing on the backend.
`--apply`: after approval, automatically re-fetches integration bundle and re-adds all affected methods to `tooling.json`.

### When to use these in agent workflows

- After any `swytchcode exec` that fails unexpectedly: run `swytchcode check` to see if a breaking integration change is the cause.
- In CI/CD: use `swytchcode check` exit code as a gate — exit 1 blocks the pipeline.
- Do NOT auto-approve upgrades without explicit user confirmation.

---

## Mental Model (CRITICAL)

Claude is **not exploring** Swytchcode.

Claude is **compiling against Swytchcode**.

If something does not exist, compilation must fail.

Failing fast is correct behavior.

---


**End of Contract**
