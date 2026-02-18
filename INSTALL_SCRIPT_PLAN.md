# Install Script Plan: OS/Arch-Aware Distribution

## Goal

A single install command that:
1. Detects the user's **OS** (macOS, Linux, Windows) and **architecture** (amd64, arm64)
2. Downloads the matching binary from **GitLab Releases**
3. Verifies with checksums, extracts, and installs to a chosen path

---

## Your Setup: Private Repo, Public Pages + Releases

- **Repo:** private  
- **Public:** Pages, Wiki, Releases (per your GitLab config)

So:
- **Binaries:** Served from **GitLab Releases** (public). Download URLs work for everyone.
- **Script:** If the script lives only in the **repo**, the raw file URL (`gitlab.com/.../raw/main/install.sh`) requires auth for a private repo, so **the script would not be publicly visible or usable**.
- **Conclusion:** For a public one-liner, the script **must** be served from somewhere public = **Pages**. Then:
  - Users run: `curl -fsSL https://<your-pages-url>/install.sh | sh`
  - The script (running on the user’s machine) downloads **binaries from GitLab Releases** (public) and installs them.
  - So: **script lives on Pages, binaries from Releases.**

**Will it execute?** Yes. The browser/curl fetches the script as plain text from Pages; the user’s shell then runs it with `sh`. No auth needed. Only the script source and the Release assets need to be public; your repo can stay private.

**Single command, no setup:** One command does everything: `curl -fsSL https://<pages>/install.sh | sh`. No login, no API token, no saving a script file first. Curl streams the script to `sh`; the script downloads the binary from Releases (public), verifies checksums, and installs. No manual script creation on the user machine.

---

## Decided Choices

| Topic | Choice |
|--------|--------|
| **Binary source** | GitLab Releases first (e.g. `.../releases/latest/downloads/...`) |
| **Script hosting** | **Pages** (so it’s public). Script in repo is only for editing; CI copies it into the Pages artifact so it’s served at e.g. `https://<pages>/install.sh`. |
| **Install path** | If `/usr/local/bin` is writable → use it. Else → `~/.local/bin`. Always allow override via **`INSTALL_DIR`**. |
| **Version** | Latest by default; support pinned version via **`VERSION`** (e.g. `VERSION=v0.1.5`). |
| **Windows** | Support both: shell script (Unix/macOS/Linux + WSL) and PowerShell script. Both scripts must be on **Pages** (e.g. `install.sh`, `install.ps1`). |
| **Checksums** | Yes. Script downloads `checksums.txt` from the same release and verifies SHA256 before installing. |

---

## Where the script “lives” (summary)

- **Repo (private):**  
  - `install.sh` and `install.ps1` live in the repo for versioning and edits.  
  - They are **not** publicly accessible via raw GitLab URL.

- **Pages (public):**  
  - CI (e.g. the same job that publishes release artifacts to Pages) **copies** `install.sh` and `install.ps1` from the repo into the Pages output (e.g. into `public/`).  
  - Pages then serves them at e.g.:
    - `https://<your-pages>/install.sh`
    - `https://<your-pages>/install.ps1`
  - So the script “lives” on Pages for **execution**; the repo is the **source of truth** and stays private.

- **Releases:**  
  - Only binaries (+ checksums) are needed from Releases. The install scripts are not required to be release assets unless you want a copy there too.

---

## Current Artifact Layout (from Goreleaser)

| OS     | Arch  | Archive              | Extension |
|--------|--------|----------------------|-----------|
| darwin | amd64, arm64 | swytchcode_darwin_*.tar.gz  | .tar.gz |
| linux  | amd64, arm64 | swytchcode_linux_*.tar.gz   | .tar.gz |
| windows| amd64, arm64 | swytchcode_windows_*.zip    | .zip     |

**Binary URLs (GitLab Releases):**
- Latest: `https://gitlab.com/swytchcode/cli/-/releases/latest/downloads/swytchcode_<os>_<arch>.<ext>`
- Pinned:  `https://gitlab.com/swytchcode/cli/-/releases/v0.1.5/downloads/swytchcode_<os>_<arch>.<ext>`

**Script URLs (Pages, after CI publishes them):**
- Shell:   `https://<your-pages-url>/install.sh`
- Windows: `https://<your-pages-url>/install.ps1`

**CI change:** The existing Pages job (runs on tag) should also copy `install.sh` and `install.ps1` from the repo into `public/` so they are deployed with the same pipeline.

---

## Behavior and safety (summary)

- **Idempotent:** Overwrites existing binary if present.
- **No auto-exec:** Script only downloads, verifies, extracts, and moves (no running the binary unless you add opt-in).
- **Compatibility:** Shell = portable sh/bash; PowerShell for Windows.
- **curl -fsSL** for downloads.

---

### Windows

- Goreleaser produces **.zip** for Windows. Provide both **install.sh** (Unix/macOS/Linux + WSL) and **install.ps1** (Windows); both served from Pages.

## Proposed Script Flow (high level)

1. **Detect OS:** `uname -s` → darwin, linux, (windows → suggest manual or PowerShell).
2. **Detect arch:** `uname -m` → x86_64 → amd64; arm64/aarch64 → arm64.
3. **Choose artifact:** e.g. `swytchcode_${OS}_${ARCH}.tar.gz` or `.zip` for Windows.
4. **Base URL:** GitLab Releases (e.g. `.../releases/${VERSION}/downloads/`); optional `BASE_URL` override.
5. **Version:** From env `VERSION` or default `latest`.
6. **Download:** `curl -fsSL` artifact and `checksums.txt` from the same release.
7. **Verify:** SHA256 of artifact vs `checksums.txt`; exit on mismatch.
8. **Extract:** `tar -xzf` or `unzip` into a temp dir.
9. **Install:** Move binary to `$INSTALL_DIR` (if set), else `/usr/local/bin` if writable, else `~/.local/bin`; use sudo only when needed.
10. **Print:** e.g. "Installed swytchcode to /usr/local/bin/swytchcode" and run `swytchcode --version`.

---

## Implementation checklist

- [x] Add `install.sh` (and `install.ps1`) in repo: detect OS/arch, download from GitLab Releases, verify checksums, install to `INSTALL_DIR` or default path.
- [x] In Pages job: copy `install.sh` and `install.ps1` from repo into `public/` so they are served at e.g. `https://<pages>/install.sh`.
- [x] Document one-liner: `curl -fsSL https://<pages-url>/install.sh | sh` (and Windows/PowerShell equivalent).
