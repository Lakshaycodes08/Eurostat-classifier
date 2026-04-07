# FAQ

**Q: Where do I install?**  
A: [Pages → Install](https://swytchcode.gitlab.io/cli/install/). One command: `curl -fsSL https://swytchcode.gitlab.io/cli/install.sh | sh`

**Q: Can I pin a version?**  
A: Yes. `VERSION=v0.1.5 curl -fsSL https://swytchcode.gitlab.io/cli/install.sh | sh`

**Q: How do I add a new tool?**  
A: `swytchcode get <project>` (if needed), then `swytchcode add <canonical_id>`. The tool must be in `tooling.json` to be executable.

**Q: Why "tool not found" when I see it in list?**  
A: `list` shows what is *available* from fetched integrations. `exec` only runs tools that are also in `tooling.json`. Run `swytchcode add <canonical_id>`.

**Q: Does Swytchcode call the registry at runtime?**  
A: No. `search` hits the registry; `get`/`bootstrap`/`list`/`exec` use only local config and `.swytchcode/integrations/`.

**Q: What editors are supported?**  
A: Cursor and Claude (via `swytchcode init --editor=cursor|claude`). Any MCP client can use `swytchcode mcp serve`.

**Q: Can I use a plain `http://` API URL for tools?**  
A: Only if the host is loopback: `localhost`, `127.0.0.1`, or `::1`. Otherwise the base URL must be `https://`. This is enforced for every exec (including in GitHub Actions, GitLab CI, and Docker). See the manifest section in [docs/config-spec.md](https://gitlab.com/swytchcode/cli/-/blob/main/docs/config-spec.md).

**Q: Does `SWYTCHCODE_INSECURE=1` let me use HTTP to internal hosts?**  
A: No — it only skips TLS certificate verification. It does not bypass the HTTPS-or-loopback-HTTP rule. In CI, registry calls fail if `SWYTCHCODE_INSECURE=1` is set.
