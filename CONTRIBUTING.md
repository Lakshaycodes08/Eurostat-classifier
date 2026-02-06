# Contributing to Swytchcode

This guide helps developers understand how to contribute to the Swytchcode kernel implementation.

**Before you start:** Read [DESIGN.md](./DESIGN.md) to understand the architectural principles and boundaries.

---

## Getting Started

### Prerequisites

- Go 1.21 or later
- Understanding of the Swytchcode architecture (see [DESIGN.md](./DESIGN.md))
- Familiarity with Cobra CLI framework

### Repository Structure

```
swytchcode/
├── cmd/
│   └── swytchcode/
│       └── main.go          # CLI entrypoint
├── internal/
│   ├── cli/                 # Cobra commands
│   │   ├── root.go
│   │   ├── init.go
│   │   ├── get.go
│   │   ├── exec.go
│   │   ├── list.go
│   │   ├── describe.go
│   │   ├── rm.go
│   │   ├── upgrade.go
│   │   └── mode.go
│   ├── kernel/              # Execution authority
│   │   ├── executor.go
│   │   ├── resolver.go
│   │   ├── policy.go
│   │   └── errors.go
│   ├── wreken/              # Wrekenfile parsing & validation
│   │   ├── loader.go
│   │   ├── validator.go
│   │   └── schema.go (to be added)
│   ├── tooling/             # tooling.json contract
│   │   ├── loader.go
│   │   └── schema.go (to be added)
│   ├── editors/             # Init-time only
│   │   ├── cursor.go
│   │   ├── vscode.go
│   │   └── claude.go
│   └── util/                # Shared helpers
│       ├── interactive.go
│       ├── jsonio.go
│       ├── fs.go
│       ├── env.go
│       └── prompt.go
├── go.mod
├── go.sum
├── README.md
├── DESIGN.md
└── CONTRIBUTING.md
```

---

## Development Workflow

### 1. Building the CLI

```bash
go build ./cmd/swytchcode
```

This creates a `swytchcode` binary in the current directory.

### 2. Running Tests

```bash
go test ./...
```

### 3. Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Keep functions focused and small
- Document exported functions

---

## Implementation Roadmap

### Phase 1: CLI Wiring (Current State)

**Status:** ✅ Skeleton complete, interactive prompts implemented

**Completed:**
- ✅ Basic command structure (`init`, `get`, `exec`, `rm`, `upgrade`, `list`, `describe`, `mode`)
- ✅ Interactive prompts for `init` (editor and mode selection)
- ✅ Non-interactive mode support with flags
- ✅ Mode stored in `tooling.json`
- ✅ Editor config writers (stubs for Cursor, VS Code, Claude)

**Remaining:**
- [ ] Complete interactive prompts for `get` (library selection, overwrite confirmation)
- [ ] Integrate mode into kernel execution logic (credential selection, policy enforcement)

### Phase 2: Kernel and Contracts

**Status:** 🔄 In progress

**To implement:**

1. **Tool resolution (`internal/kernel/resolver.go`)**
   - Load `tooling.json`
   - Resolve tool → library → Wrekenfile
   - Handle `raw.*` namespace (bypass `tooling.json`, require `--allow-raw`)

2. **Schema validation**
   - Input validation against `tooling.json` schema
   - Output normalization to declared shape
   - Env-based auth checks (missing → exit code 3)

3. **SDK invocation layer**
   - Execute SDK calls based on Wrekenfile definitions
   - Map SDK errors to exit codes (4 = SDK failure, 5 = internal error)

4. **Wrekenfile and tooling.json loaders**
   - Implement `internal/wreken/loader.go` and `validator.go`
   - Implement `internal/tooling/loader.go`
   - Define JSON schemas (structs) for both

### Phase 3: Policy and Retries

**Status:** 📋 Planned

**To implement:**

1. **Policy layer (`internal/kernel/policy.go`)**
   - Idempotency handling (keys, safe retries)
   - Retry strategies for network/transient failures
   - Guardrails from Wrekenfile/tooling.json contracts

2. **HTTP client wrapper**
   - Single shared `*http.Client` per process
   - Connection pooling via custom `Transport`
   - Timeout configuration
   - No third-party REST client (stdlib only)

### Phase 4: Promotion and Proposals

**Status:** 📋 Planned

**To implement:**

1. **`swytchcode propose` command**
   - Generate proposal files under `.swytchcode/proposals/`
   - Read Wrekenfile method definition
   - Infer/generate I/O schema
   - Write proposal JSON (does not modify `tooling.json`)

2. **`swytchcode apply` command**
   - Validate proposal file
   - Apply proposal to `tooling.json`
   - Only command that mutates `tooling.json`
   - Archive or delete proposal after apply

3. **Verified workflows support**
   - `swytchcode add workflow <name>` for verified workflows
   - Direct application (no proposal step)

---

## Key Implementation Rules

### 1. Exec Never Prompts

**Hard rule:** Any prompt inside `exec` is a hard reject.

```go
// ✅ CORRECT
func Execute(stdin io.Reader, stdout io.Writer, stderr io.Writer, allowRaw bool) int {
    // No TTY detection, no prompts
}

// ❌ WRONG
func Execute(...) {
    if util.IsInteractive() {
        // prompt user  // NEVER DO THIS
    }
}
```

### 2. Env-Only Auth (in exec)

```go
// ✅ CORRECT
apiKey := os.Getenv("STRIPE_API_KEY")
if apiKey == "" {
    return ExitCodeAuthError
}

// ❌ WRONG
apiKey := readFromConfigFile()  // NO
apiKey := promptUser()           // NO
```

### 3. tooling.json is Write-Protected

```go
// ✅ CORRECT - Only via apply command
func ApplyProposal(proposalPath string) error {
    // Validate, then write to tooling.json
}

// ❌ WRONG - Never modify directly from get/init/etc
func GetLibrary(library string) error {
    // tooling.json[library] = ...  // NEVER DO THIS
}
```

### 4. Raw Method Execution

```go
// ✅ CORRECT
if strings.HasPrefix(tool, "raw.") {
    if !allowRaw {
        return ExitCodeInvalidInput  // Fail fast
    }
    // Resolve via Wrekenfile, bypass tooling.json
}

// ❌ WRONG
if strings.HasPrefix(tool, "raw.") {
    // Silent fallback to verified  // NEVER DO THIS
}
```

### 5. Single Shared HTTP Client

```go
// ✅ CORRECT - Package-level shared client
var httpClient = &http.Client{
    Transport: &http.Transport{
        MaxIdleConns: 100,
        MaxIdleConnsPerHost: 10,
    },
    Timeout: 30 * time.Second,
}

// ❌ WRONG - New client per request
func fetchWrekenfile(url string) {
    client := &http.Client{}  // NO - creates new connections
}
```

---

## Testing Requirements

### Quality Gates (Must Pass)

Before any PR is merged, ensure:

1. **Docker test:**
   ```bash
   docker run --rm -v $(pwd):/work -w /work golang:1.21 go build ./cmd/swytchcode
   echo '{"tool":"x.y","args":{}}' | ./swytchcode exec
   ```

2. **CI simulation:**
   ```bash
   # No TTY, no HOME
   unset HOME
   echo '{"tool":"x.y","args":{}}' | ./swytchcode exec
   ```

3. **Exit code verification:**
   - Invalid JSON → exit code 1
   - Tool not found → exit code 2
   - Missing env var → exit code 3
   - SDK failure → exit code 4

4. **Non-interactive init:**
   ```bash
   swytchcode init --editor=none --mode=production --non-interactive
   # Must not prompt, must succeed
   ```

---

## Code Review Checklist

When reviewing PRs, verify:

- [ ] No prompts in `exec` command
- [ ] No SDK logic in thin clients (if adding client code)
- [ ] No editor logic at runtime
- [ ] Env-only auth (no config files for secrets)
- [ ] JSON in / JSON out (no prose in exec)
- [ ] Stable exit codes (0-5 only)
- [ ] Works without TTY
- [ ] `get` never modifies `tooling.json`
- [ ] Raw methods require explicit opt-in
- [ ] Single shared HTTP client (if adding HTTP code)
- [ ] No third-party REST client in kernel

---

## Common Pitfalls to Avoid

### ❌ "Helpful" Fallbacks

```go
// DON'T: Silent fallback from verified to raw
if !foundInTooling {
    // Try raw method instead  // NO
}
```

### ❌ Auto-Promotion

```go
// DON'T: Auto-add to tooling.json
func GetLibrary(library string) {
    tooling.Tools[library] = ...  // NO
}
```

### ❌ Environment Guessing

```go
// DON'T: Guess CI vs dev
if os.Getenv("CI") == "true" {
    // Different behavior  // NO - use TTY + flags only
}
```

### ❌ Interactive Exec

```go
// DON'T: Prompt during exec
if util.IsInteractive() {
    confirm := prompt("Continue?")  // NO - exec is never interactive
}
```

---

## Adding New Commands

If you need to add a new command:

1. **Determine interaction mode:**
   - Setup/setup → may be interactive (e.g. `init`, `get`)
   - Diagnostic → may be interactive (e.g. `list`, `describe`)
   - Execution → **never interactive** (only `exec`)

2. **Create command file:**
   ```go
   // internal/cli/newcommand.go
   package cli
   
   var newCommandCmd = &cobra.Command{
       Use:   "newcommand",
       Short: "Description",
       RunE: func(cmd *cobra.Command, args []string) error {
           // Implementation
       },
   }
   ```

3. **Register in `root.go`:**
   ```go
   rootCmd.AddCommand(newCommandCmd)
   ```

4. **Update README.md** with command documentation.

---

## Schema Definitions (To Be Implemented)

### tooling.json Schema

```go
// internal/tooling/schema.go (to be created)
type Tooling struct {
    Version string             `json:"version"`
    Mode    string             `json:"mode"` // "production" | "sandbox"
    Tools   map[string]ToolDef `json:"tools"`
}

type ToolDef struct {
    Library   string                 `json:"library"`
    Operation string                 `json:"operation"`
    Input     map[string]interface{} `json:"input,omitempty"`
    Output    map[string]interface{} `json:"output,omitempty"`
    // ... other fields
}
```

### Wrekenfile Schema

```go
// internal/wreken/schema.go (to be created)
type Wrekenfile struct {
    Library string              `json:"library"`
    Methods map[string]Method   `json:"methods"`
    Auth    AuthConfig          `json:"auth"`
}

type Method struct {
    Title   string                 `json:"title,omitempty"`
    SDKCall string                 `json:"sdk_call"`
    // ... other fields
}
```

---

## Questions?

- **Architecture questions:** See [DESIGN.md](./DESIGN.md)
- **Implementation questions:** Check existing code patterns in `internal/kernel/` and `internal/cli/`
- **Unclear requirements:** The README and DESIGN.md are the source of truth

---

## Definition of "Done" for v1

Before shipping v1, ensure:

- ✅ `swytchcode exec` fully deterministic
- ✅ Works in Docker scratch image
- ✅ Works in GitHub Actions
- ✅ Thin client can be written in <50 LOC
- ✅ No interactive code path in kernel
- ✅ Editor configs affect IDEs only
- ✅ One kernel binary only

---

**Remember:** You are building a kernel, not a helper CLI.  
**init and get → human-friendly**  
**exec → machine-only, deterministic**
