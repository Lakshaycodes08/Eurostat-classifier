#!/bin/sh
# Install swytchcode from GitLab Releases.
# Usage: curl -fsSL https://<pages-url>/install.sh | sh
# Env: VERSION (default: latest), INSTALL_DIR (override install path), BASE_URL (override release base URL).

set -e

BINARY_NAME="swytchcode"
RELEASE_BASE="${BASE_URL:-https://gitlab.com/swytchcode/cli/-/releases}"
VERSION="${VERSION:-latest}"

# Detect OS and arch (darwin/linux + amd64/arm64)
detect_platform() {
  os="$(uname -s)"
  arch="$(uname -m)"
  case "$os" in
    Darwin)  os="darwin" ;;
    Linux)   os="linux"  ;;
    *)
      echo "Unsupported OS: $os. Use Windows? Try install.ps1 or download from Releases."
      exit 1
      ;;
  esac
  case "$arch" in
    x86_64)  arch="amd64" ;;
    aarch64) arch="arm64" ;;
    arm64)   arch="arm64" ;;
    *)
      echo "Unsupported arch: $arch"
      exit 1
      ;;
  esac
}

# Choose install directory: INSTALL_DIR, or /usr/local/bin if writable, else ~/.local/bin
choose_install_dir() {
  if [ -n "$INSTALL_DIR" ]; then
    echo "$INSTALL_DIR"
    return
  fi
  if [ -w /usr/local/bin ] 2>/dev/null; then
    echo "/usr/local/bin"
    return
  fi
  if [ -z "$HOME" ]; then
    HOME="$(eval echo ~)"
  fi
  echo "$HOME/.local/bin"
}

# Compute SHA256 of file; print hash only (portable: macOS shasum, Linux sha256sum)
sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

main() {
  detect_platform
  artifact_name="${BINARY_NAME}_${os}_${arch}.tar.gz"
  if [ "$VERSION" = "latest" ]; then
    download_base="${RELEASE_BASE}/permalink/latest/downloads"
  else
    download_base="${RELEASE_BASE}/${VERSION}/downloads"
  fi

  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT

  echo "Downloading ${artifact_name} and checksums.txt..."
  if ! curl -fsSL -o "${tmpdir}/artifact.tar.gz" "${download_base}/${artifact_name}"; then
    echo "Download failed. Check VERSION (e.g. v0.1.0) and network."
    exit 1
  fi
  if ! curl -fsSL -o "${tmpdir}/checksums.txt" "${download_base}/checksums.txt"; then
    echo "Failed to download checksums.txt."
    exit 1
  fi

  expected_hash="$(grep " ${artifact_name}$" "${tmpdir}/checksums.txt" | awk '{print $1}')"
  if [ -z "$expected_hash" ]; then
    echo "Checksums.txt does not contain ${artifact_name}."
    exit 1
  fi
  actual_hash="$(sha256_of "${tmpdir}/artifact.tar.gz")"
  if [ "$expected_hash" != "$actual_hash" ]; then
    echo "Checksum mismatch. Expected ${expected_hash}, got ${actual_hash}."
    exit 1
  fi
  echo "Checksum OK."

  (cd "$tmpdir" && tar -xzf artifact.tar.gz)
  if [ ! -f "${tmpdir}/${BINARY_NAME}" ]; then
    echo "Archive did not contain ${BINARY_NAME}."
    exit 1
  fi

  install_dir="$(choose_install_dir)"
  mkdir -p "$install_dir"
  dest="${install_dir}/${BINARY_NAME}"

  if [ -w "$install_dir" ]; then
    cp "${tmpdir}/${BINARY_NAME}" "$dest"
  else
    echo "Installing to ${install_dir} (may prompt for password)."
    sudo cp "${tmpdir}/${BINARY_NAME}" "$dest"
  fi
  chmod 755 "$dest"

  echo "Installed swytchcode to ${dest}"
  "$dest" --version
}

main "$@"
