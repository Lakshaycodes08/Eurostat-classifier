# Release & Distribution Plan

**Repository:** `gitlab.com/swytchcode/cli` (private)  
**Strategy:** Single private repo with public releases  
**First Release:** `v0.1.0`  
**Distribution:** GitLab Releases (public) + curl downloads

---

## Overview

This plan sets up automated binary releases using:
1. **Goreleaser** - Builds binaries for multiple platforms
2. **GitLab Releases** - Hosts downloadable binaries (made public after creation)
3. **curl Downloads** - Users download directly from GitLab release URLs

**Key Approach:** 
- Source code repository stays **private**
- GitLab releases are created as **private** by default
- Releases are made **public** after creation (manual or automated)
- Users can download binaries without authentication

---

## 1. Goreleaser Setup

### Prerequisites

- GitLab repository: `gitlab.com/swytchcode/cli` (private)
- GitLab CI/CD enabled
- GitLab Personal Access Token (or Project Token) with `api` scope
- Git tags for releases (starting from `v0.1.0`)

### Files to Create

1. **`.goreleaser.yml`** (root directory)
   - Configure builds for: `darwin` (amd64, arm64), `linux` (amd64, arm64), `windows` (amd64, arm64)
   - Binary name: `swytchcode`
   - Upload to GitLab releases
   - Generate checksums

2. **`.gitlab-ci.yml`** (if not exists)
   - Add `goreleaser` job that runs on tags
   - Uses GitLab token for releases
   - Optionally: Make release public via GitLab API

### Setup Steps

1. **Install goreleaser locally** (for testing):
   ```bash
   brew install goreleaser
   # or
   go install github.com/goreleaser/goreleaser@latest
   ```

2. **Test locally** (dry run):
   ```bash
   goreleaser release --snapshot --skip-publish
   ```

3. **Create GitLab Access Token:**
   - Go to: `https://gitlab.com/swytchcode/cli/-/settings/access_tokens`
   - Create token with `api` scope (name: `goreleaser-releases`)
   - Copy token value
   - Go to: `https://gitlab.com/swytchcode/cli/-/settings/ci_cd`
   - Add CI/CD variable: `GITLAB_TOKEN` (paste token, mark as masked and protected)

4. **Create first release tag:**
   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0"
   git push origin v0.1.0
   ```

---

## 2. GitLab Releases & Public Visibility

### How It Works

**Initial State:**
- Goreleaser creates release as **private** (inherits repo visibility)
- Release assets are uploaded but not publicly accessible

**Making Releases Public:**

**Option A: Manual (Simplest)**
1. After CI/CD completes, go to: `https://gitlab.com/swytchcode/cli/-/releases`
2. Click on the release (e.g., `v0.1.0`)
3. Click "Edit" → Toggle "Public" visibility → Save
4. Release assets become publicly downloadable

**Option B: Automated (Recommended)**
Add GitLab API call in CI/CD to make release public automatically:

```yaml
# In .gitlab-ci.yml, after goreleaser job:
make_release_public:
  stage: release
  image: curlimages/curl:latest
  script:
    - |
      curl --request PUT \
        --header "PRIVATE-TOKEN: $GITLAB_TOKEN" \
        --header "Content-Type: application/json" \
        "https://gitlab.com/api/v4/projects/swytchcode%2Fcli/releases/v${CI_COMMIT_TAG#v}" \
        --data '{"released_at": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"}'
  only:
    - tags
  needs:
    - goreleaser
```

**Note:** GitLab API may require additional permissions. Test manually first, then automate.

### Public Download URLs

After release is made public, binaries are accessible at:
```
https://gitlab.com/swytchcode/cli/-/releases/v0.1.0/downloads/swytchcode_darwin_arm64.tar.gz
https://gitlab.com/swytchcode/cli/-/releases/v0.1.0/downloads/swytchcode_darwin_amd64.tar.gz
https://gitlab.com/swytchcode/cli/-/releases/v0.1.0/downloads/swytchcode_linux_amd64.tar.gz
https://gitlab.com/swytchcode/cli/-/releases/v0.1.0/downloads/swytchcode_linux_arm64.tar.gz
https://gitlab.com/swytchcode/cli/-/releases/v0.1.0/downloads/swytchcode_windows_amd64.zip
https://gitlab.com/swytchcode/cli/-/releases/v0.1.0/downloads/swytchcode_windows_arm64.zip
```

---

## 3. User Download Methods

### Method 1: Direct curl Download

**macOS (Apple Silicon):**
```bash
curl -L "https://gitlab.com/swytchcode/cli/-/releases/v0.1.0/downloads/swytchcode_darwin_arm64.tar.gz" | tar xz
sudo mv swytchcode /usr/local/bin/
```

**macOS (Intel):**
```bash
curl -L "https://gitlab.com/swytchcode/cli/-/releases/v0.1.0/downloads/swytchcode_darwin_amd64.tar.gz" | tar xz
sudo mv swytchcode /usr/local/bin/
```

**Linux (amd64):**
```bash
curl -L "https://gitlab.com/swytchcode/cli/-/releases/v0.1.0/downloads/swytchcode_linux_amd64.tar.gz" | tar xz
sudo mv swytchcode /usr/local/bin/
```

**Linux (arm64):**
```bash
curl -L "https://gitlab.com/swytchcode/cli/-/releases/v0.1.0/downloads/swytchcode_linux_arm64.tar.gz" | tar xz
sudo mv swytchcode /usr/local/bin/
```

### Method 2: Install Script (Future)

Create `install.sh` that auto-detects OS/arch and downloads appropriate binary.

---

## 4. Implementation Checklist

### Phase 1: Setup (One-time)

- [ ] Create `.goreleaser.yml` with correct repo name (`cli`)
- [ ] Create/update `.gitlab-ci.yml` with goreleaser job
- [ ] Create GitLab access token with `api` scope
- [ ] Add `GITLAB_TOKEN` as CI/CD variable (masked, protected)
- [ ] Test goreleaser locally: `goreleaser release --snapshot --skip-publish`

### Phase 2: First Release

- [ ] Update version in `internal/constants/constants.go` to `0.1.0`
- [ ] Commit and push changes
- [ ] Create tag: `git tag -a v0.1.0 -m "Release v0.1.0" && git push origin v0.1.0`
- [ ] Verify CI/CD pipeline runs goreleaser job
- [ ] Verify release is created in GitLab (will be private)
- [ ] Manually make release public in GitLab UI
- [ ] Test download URLs work without authentication
- [ ] Verify checksums file is present

### Phase 3: Automation (Optional)

- [ ] Test GitLab API to make release public programmatically
- [ ] Add `make_release_public` job to `.gitlab-ci.yml`
- [ ] Test automated flow with a test tag (e.g., `v0.1.1-test`)
- [ ] Remove test tag after verification

### Phase 4: Documentation

- [ ] Update README with installation instructions
- [ ] Document release process for maintainers
- [ ] Create install script (optional)

---

## 5. Configuration Files

### `.goreleaser.yml` (Template)

```yaml
project_name: swytchcode
before:
  hooks:
    - go mod download
    - go mod verify
builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    main: ./cmd/swytchcode
    binary: swytchcode
    ldflags:
      - -s -w
      - -X gitlab.com/swytchcode/shell/internal/constants.Version={{.Version}}
    flags:
      - -trimpath
archives:
  - format: tar.gz
    name_template: '{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}'
    files:
      - README.md
  - format: zip
    name_template: '{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}'
    files:
      - README.md
checksum:
  name_template: 'checksums.txt'
  algorithm: sha256
release:
  gitlab:
    owner: swytchcode
    name: cli
snapshot:
  name_template: "{{ .Tag }}-next"
```

### `.gitlab-ci.yml` (Goreleaser Job)

```yaml
stages:
  - release

goreleaser:
  stage: release
  image: goreleaser/goreleaser:latest
  script:
    - goreleaser release --clean
  only:
    - tags
  variables:
    GITLAB_TOKEN: $GITLAB_TOKEN
```

---

## 6. Release Workflow

### Standard Release Process

1. **Update version:**
   ```bash
   # Edit internal/constants/constants.go
   Version = "0.1.0"
   ```

2. **Commit and push:**
   ```bash
   git add internal/constants/constants.go
   git commit -m "Bump version to 0.1.0"
   git push origin main
   ```

3. **Create release tag:**
   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0"
   git push origin v0.1.0
   ```

4. **CI/CD automatically:**
   - Detects tag push
   - Runs goreleaser job
   - Builds binaries for all platforms
   - Uploads to GitLab Releases (as private)

5. **Make release public:**
   - Go to GitLab releases page
   - Edit release → Toggle "Public" → Save
   - (Or wait for automated job if implemented)

6. **Verify:**
   - Test download URLs work without authentication
   - Verify checksums file is correct
   - Test binary on different platforms

---

## 7. Troubleshooting

### Release Created but Not Public

**Symptom:** Release exists but download URLs return 404 or require auth.

**Solution:**
- Go to GitLab releases page
- Edit the release
- Toggle "Public" visibility
- Save

### Goreleaser Job Fails

**Common Issues:**
- Missing `GITLAB_TOKEN` → Check CI/CD variables
- Token lacks `api` scope → Regenerate token with correct scope
- Invalid repo name → Check `.goreleaser.yml` has `name: cli`

### Binaries Not Building

**Check:**
- Go version compatibility
- Build flags in `.goreleaser.yml`
- Local test: `goreleaser release --snapshot --skip-publish`

---

## 8. Future Enhancements

- **Install Script:** Auto-detect OS/arch and download appropriate binary
- **Latest Release URL:** Use GitLab API to get latest release tag
- **GPG Signing:** Sign binaries for additional security
- **Release Notes:** Auto-generate from git commits
- **Changelog:** Maintain CHANGELOG.md and include in releases

---

## Summary

**What This Plan Achieves:**
- ✅ Automated binary builds for multiple platforms
- ✅ Public downloads from private GitLab repo
- ✅ Simple curl-based installation for users
- ✅ Versioned releases starting from v0.1.0
- ✅ Checksums for verification

**What's NOT Included:**
- ❌ Homebrew distribution (deferred)
- ❌ Windows installer (.msi/.exe)
- ❌ Auto-update mechanism

**Next Steps:**
1. Review this plan
2. Create `.goreleaser.yml` and `.gitlab-ci.yml`
3. Set up GitLab token
4. Test with first release (v0.1.0)
