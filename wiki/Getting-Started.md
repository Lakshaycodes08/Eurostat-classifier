# Getting Started

1. **Install** – [Install page](https://swytchcode.gitlab.io/cli/install/) or:
   ```bash
   curl -fsSL https://swytchcode.gitlab.io/cli/install.sh | sh
   ```

2. **Initialize your project**
   ```bash
   swytchcode init
   ```
   Choose editor (Cursor / Claude) and mode (production / sandbox). This creates `.swytchcode/`, `tooling.json`, and editor rules.

3. **Fetch an integration**
   ```bash
   swytchcode get <project>
   ```
   Example: `swytchcode get weaviate`. This downloads the Wrekenfile and methods/workflows; it does *not* add anything to `tooling.json`.

4. **Add a tool to the allow-list**
   ```bash
   swytchcode add <canonical_id>
   ```
   Example: `swytchcode add api.cluster.create`. This adds the tool to `tooling.json`.

5. **Run a tool**
   ```bash
   swytchcode exec <canonical_id> [args...]
   ```
   Or via JSON stdin: `echo '{"tool":"...","args":{...}}' | swytchcode exec`.

**Next:** [Core Concepts](Execution-Model) and [CLI Reference](CLI-Reference).
