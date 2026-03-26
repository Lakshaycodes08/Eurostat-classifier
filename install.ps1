# Install swytchcode from GitLab Releases (Windows).
# Usage: irm https://cli.swytchcode.com/install.ps1 | iex
# Env: $env:VERSION (default: latest), $env:INSTALL_DIR (override install path), $env:BASE_URL (override release base).

$ErrorActionPreference = "Stop"
$BinaryName = "swytchcode"
# Default download base is GitLab Pages (cli.swytchcode.com), which is updated by GitLab CI
# for every tagged release and includes a /releases/latest directory that always matches
# the website's advertised latest version.
#
# You can override via BASE_URL to point at GitLab Releases (or a mirror) if needed.
$ReleaseBase = if ($env:BASE_URL) { $env:BASE_URL } else { "https://cli.swytchcode.com/releases" }
$Version = if ($env:VERSION) { $env:VERSION } else { "latest" }

$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
$artifactName = "${BinaryName}_windows_${arch}.zip"
$downloadBase = if ($Version -eq "latest") {
    "${ReleaseBase}/latest"
} else {
    "${ReleaseBase}/${Version}"
}

$tmpDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_.FullName }
try {
    Write-Host "Downloading $artifactName and checksums.txt..."
    $artifactPath = Join-Path $tmpDir.FullName "artifact.zip"
    $checksumsPath = Join-Path $tmpDir.FullName "checksums.txt"

    Invoke-WebRequest -Uri "${downloadBase}/${artifactName}" -OutFile $artifactPath -UseBasicParsing
    Invoke-WebRequest -Uri "${downloadBase}/checksums.txt" -OutFile $checksumsPath -UseBasicParsing

    $checksumsContent = Get-Content $checksumsPath -Raw
    $line = ($checksumsContent -split "`n") | Where-Object { $_ -match [regex]::Escape($artifactName) } | Select-Object -First 1
    if (-not $line) {
        Write-Error "checksums.txt does not contain $artifactName"
    }
    $expectedHash = ($line -split "\s+")[0].ToLowerInvariant()
    $actualHash = (Get-FileHash -Path $artifactPath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($expectedHash -ne $actualHash) {
        Write-Error "Checksum mismatch. Expected $expectedHash, got $actualHash"
    }
    Write-Host "Checksum OK."

    Expand-Archive -Path $artifactPath -DestinationPath $tmpDir.FullName -Force
    $binaryPath = Join-Path $tmpDir.FullName "${BinaryName}.exe"
    if (-not (Test-Path $binaryPath)) {
        $binaryPath = Join-Path $tmpDir.FullName $BinaryName
    }
    if (-not (Test-Path $binaryPath)) {
        Write-Error "Archive did not contain ${BinaryName}.exe or $BinaryName"
    }

    $installDir = if ($env:INSTALL_DIR) {
        $env:INSTALL_DIR
    } else {
        Join-Path $env:LOCALAPPDATA "Programs\swytchcode\bin"
    }
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    $destPath = Join-Path $installDir (Split-Path $binaryPath -Leaf)

    Copy-Item -Path $binaryPath -Destination $destPath -Force
    Write-Host "Installed swytchcode to $destPath"

    $pathDirs = [Environment]::GetEnvironmentVariable("Path", "User") -split ";"
    if ($pathDirs -notcontains $installDir) {
        [Environment]::SetEnvironmentVariable("Path", ($pathDirs + $installDir) -join ";", "User")
        $env:Path = "$env:Path;$installDir"
        Write-Host "Added $installDir to user PATH."
    }

    & $destPath --version
} finally {
    Remove-Item -Path $tmpDir.FullName -Recurse -Force -ErrorAction SilentlyContinue
}
