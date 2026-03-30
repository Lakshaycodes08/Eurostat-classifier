# Windows guide — Swytchcode CLI

## JSON and `swytchcode exec`

### `&` in **cmd.exe**

`cmd.exe` treats `&` as a command separator. Inline JSON that contains `&` (common in URLs or form-style values) is often **truncated or split**, so the CLI receives invalid JSON.

**Fix:** put the payload in a file and use:

```cmd
swytchcode exec api.example.method --body payload.json
```

Or use **PowerShell** with a here-string or `Get-Content` so the shell does not interpret `&`:

```powershell
$json = Get-Content -Raw payload.json
$json | swytchcode exec
```

### Prefer `--body` for complex JSON

On Windows, **PowerShell** and **cmd** quoting for large JSON is error-prone. The kernel appends a hint when JSON parsing fails on Windows or when the input contains `&`.

### Paths

Integration bundles live under `.swytchcode\integrations\...`. The CLI uses Go’s `filepath` APIs; use normal Windows paths. If `swytchcode add` reports a missing bundle, the error includes the **expected directory** — compare it to Explorer.

### MCP daemon

Daemon mode is supported on Windows (see README). Use `swytchcode mcp stop` to stop the background server.

### Environment variables

Set `SWYTCHCODE_TOKEN` via PowerShell:

```powershell
$env:SWYTCHCODE_TOKEN = "your_token"
```

For persistence, use User environment variables or `setx` (open a new terminal afterward). See [cli-reference.md](cli-reference.md).
