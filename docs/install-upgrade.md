# Swytchcode CLI – Install & Upgrade

This document describes how to install Swytchcode on macOS, Linux, and Windows, and how the install scripts behave. It also covers basic upgrade behavior.

## Install (macOS / Linux / WSL)

### One-liner (latest)

```bash
curl -fsSL https://cli.swytchcode.com/install.sh | sh
```

Behavior (`install.sh`):

- Detects OS (`Darwin` / `Linux`) and architecture (`amd64` / `arm64`).
- Determines the correct artifact name:
  - `swytchcode_darwin_amd64.tar.gz`
  - `swytchcode_darwin_arm64.tar.gz`
  - `swytchcode_linux_amd64.tar.gz`
  - `swytchcode_linux_arm64.tar.gz`
- Resolves the release download base (GitLab Pages, published by GitLab CI):
  - Latest: `https://cli.swytchcode.com/releases/latest`.
  - Pinned: `https://cli.swytchcode.com/releases/$VERSION`.
- Downloads:
  - The tarball for your OS/arch.
  - `checksums.txt` from the same release.
- Verifies SHA256 checksum:
  - Computes SHA256 of the tarball.
  - Compares it to the entry in `checksums.txt`.
- Extracts the tarball into a temporary directory.
- Installs the `swytchcode` binary to:
  - `$INSTALL_DIR` if set, otherwise:
    - `/usr/local/bin` if writable, else
    - `$HOME/.local/bin`.
- Uses `sudo` only when needed (non-writable system directory).
- Prints the installed path and runs `swytchcode --version`.

### Pinned version

```bash
VERSION=v1.0.5 curl -fsSL https://cli.swytchcode.com/install.sh | sh
```

The script uses `VERSION` to construct the download base:

- `https://cli.swytchcode.com/releases/v0.1.5`.

### Custom install directory

```bash
INSTALL_DIR="$HOME/bin" curl -fsSL https://cli.swytchcode.com/install.sh | sh
```

The script installs `swytchcode` into `$INSTALL_DIR` instead of system defaults.

### Environment variables

- `VERSION` – release tag (e.g. `v0.1.5`). Defaults to `latest`.
- `INSTALL_DIR` – target directory for the binary.
- `BASE_URL` – override for the release base URL (advanced use; defaults to `https://cli.swytchcode.com/releases`).

## Install (Windows, PowerShell)

### One-liner (latest)

```powershell
irm https://cli.swytchcode.com/install.ps1 | iex
```

Behavior (`install.ps1`):

- Determines architecture (currently treats 64-bit Windows as `amd64`).
- Constructs the artifact name:
  - `swytchcode_windows_amd64.zip`
- Resolves the release download base (GitLab Pages, published by GitLab CI):
  - Latest: `https://cli.swytchcode.com/releases/latest`.
  - Pinned: `https://cli.swytchcode.com/releases/$env:VERSION`.
- Downloads:
  - Zip file: `swytchcode_windows_amd64.zip`.
  - `checksums.txt` from the same release.
- Verifies SHA256 checksum using `Get-FileHash`.
- Extracts the zip into a temporary directory via `Expand-Archive`.
- Finds the binary (prefers `swytchcode.exe`).
- Installs to:
  - `$env:INSTALL_DIR` if set, else
  - `$env:LOCALAPPDATA\Programs\swytchcode\bin`.
- Ensures the install directory exists and copies the binary there.
- Adds the install directory to the user `PATH` if not already present.
- Prints the installed path and runs `swytchcode --version`.

### Pinned version

```powershell
$env:VERSION = "v1.0.5"
irm https://cli.swytchcode.com/install.ps1 | iex
```

### Custom install directory

```powershell
$env:INSTALL_DIR = "$env:USERPROFILE\bin"
irm https://cli.swytchcode.com/install.ps1 | iex
```

### Non-piped usage (for strict execution policy)

If your execution policy prevents running scripts from the internet, you can:

1. Download the installer:

   ```powershell
   Invoke-WebRequest -Uri https://cli.swytchcode.com/install.ps1 -OutFile install.ps1
   ```

2. Run it explicitly (adjust `-ExecutionPolicy` according to your org’s policy):

   ```powershell
   powershell -NoProfile -ExecutionPolicy Bypass -File .\install.ps1
   ```

If this still fails due to policy restrictions, fall back to **manual install** from Releases.

### Manual install from Releases

1. Go to: `https://gitlab.com/swytchcode/cli/-/releases`.
2. Download the appropriate `swytchcode_windows_amd64.zip` (or matching architecture).
3. Extract it to a directory of your choice (e.g. `C:\Tools\swytchcode`).
4. Add that directory to the user `PATH` (via System Properties → Environment Variables or `setx`).

## Upgrade

Swytchcode is versioned via Git tags (e.g. `v0.1.20`):

- New releases are built by Goreleaser and published to:
  - GitLab Releases (`https://gitlab.com/swytchcode/cli/-/releases`).
  - GitLab Pages (artifacts under `/releases/<tag>/`).

Upgrade options:

- **Re-run the installer**:
  - macOS/Linux:

    ```bash
    curl -fsSL https://cli.swytchcode.com/install.sh | sh
    ```

  - Windows (PowerShell):

    ```powershell
    irm https://cli.swytchcode.com/install.ps1 | iex
    ```

  The scripts always target the latest (or pinned) release, so re-running them upgrades your binary in-place.

- **Manual download**:
  - Download the new binary/zip from the Releases page and replace your existing binary in the install directory.

## Troubleshooting (Windows highlights)

Common issues:

- **Execution policy error**:
  - Use the non-piped approach with `-ExecutionPolicy Bypass`, or install manually from Releases.

- **`Invoke-WebRequest` or TLS errors**:
  - Check corporate proxies or TLS interception.
  - As a fallback, download the zip directly in a browser and install manually.

- **Command not found after install**:
  - Ensure the install directory (default `LOCALAPPDATA\Programs\swytchcode\bin` or your custom `INSTALL_DIR`) is on your user `PATH`.
  - Open a new terminal session after install so environment changes take effect.

For macOS/Linux issues, check:

- That the install directory is on PATH (`echo $PATH`).
- That `curl` can reach `cli.swytchcode.com` and `gitlab.com`.
- That `tar` and `sha256sum` / `shasum` are available.

