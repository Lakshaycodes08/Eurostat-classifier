# Setup Instructions - After Adding GitLab Token

## ✅ Step 1: Add Token to CI/CD Variables (DONE)

You've created the token. Now add it to GitLab CI/CD variables:
- Go to: `https://gitlab.com/swytchcode/cli/-/settings/ci_cd`
- Add variable: `GITLAB_TOKEN` (masked, protected)

## 📝 Step 2: Commit and Push Files

Commit the new release configuration files:

```bash
cd /Users/chilarai/Projects/swytchcode/shell

# Stage new files
git add .gitlab-ci.yml .goreleaser.yml scripts/auto_tag.sh RELEASE_PLAN.md

# Commit
git commit -m "Add automated release pipeline with goreleaser and auto-tagging"

# Push to main
git push origin main
```

## 🧪 Step 3: Test the Setup

### Option A: Test Auto-Tagging (Recommended First)

After pushing to `main`, the `auto_tag` job should run automatically:
1. Go to: `https://gitlab.com/swytchcode/cli/-/pipelines`
2. Find the pipeline for your commit
3. Check the `auto_tag` job:
   - Should create and push tag `v0.1.0` (if no tags exist)
   - Or increment from latest tag

**If it fails:**
- Check job logs for errors
- Verify `GITLAB_TOKEN` has `write_repository` scope
- Verify token is set as CI/CD variable

### Option B: Test Release Manually (Skip Auto-Tag)

If you want to test goreleaser first without auto-tagging:

```bash
# Create tag manually
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

This will trigger the `goreleaser` job, which will:
- Build binaries for all platforms
- Create a GitLab Release
- Upload artifacts

Then the `pages` job will publish them to GitLab Pages.

## 🔍 Step 4: Verify Release

After a tag is created (auto or manual):

1. **Check GitLab Release:**
   - Go to: `https://gitlab.com/swytchcode/cli/-/releases`
   - You should see `v0.1.0` release with assets

2. **Check GitLab Pages:**
   - Go to: `https://swytchcode.gitlab.io/cli/releases/v0.1.0/`
   - Or: `https://swytchcode.gitlab.io/cli/releases/latest/`
   - Should see downloadable binaries

3. **Test Download:**
   ```bash
   # Example: macOS Apple Silicon
   curl -L "https://swytchcode.gitlab.io/cli/releases/latest/swytchcode_darwin_arm64.tar.gz" | tar xz
   ./swytchcode --version
   ```

## ⚙️ Step 5: Enable GitLab Pages (If Not Already Enabled)

For public downloads via Pages:

1. Go to: `https://gitlab.com/swytchcode/cli/-/settings/pages`
2. Enable Pages (if not already enabled)
3. Set visibility to **Public** (if you want unauthenticated downloads)
4. The `pages` job in CI will automatically publish artifacts

## 🐛 Troubleshooting

### Auto-Tag Job Fails

**Error: "Permission denied"**
- Token needs `write_repository` scope
- Create new token with both `api` and `write_repository`

**Error: "Tag already exists"**
- Normal if tag was already created
- Job exits successfully (no-op)

### Goreleaser Job Fails

**Error: "401 Unauthorized"**
- Check `GITLAB_TOKEN` is set correctly
- Verify token has `api` scope
- Check token hasn't expired

**Error: "Release already exists"**
- Goreleaser will update existing release (this is fine)

### Pages Job Fails

**Error: "No artifacts found"**
- Ensure `goreleaser` job completed successfully
- Check `needs: goreleaser` dependency in `.gitlab-ci.yml`

**Pages URL returns 404**
- Enable Pages in project settings
- Wait a few minutes for Pages to deploy
- Check Pages deployment logs

## 📋 Quick Checklist

- [ ] Token added to CI/CD variables (`GITLAB_TOKEN`)
- [ ] Token has `api` and `write_repository` scopes
- [ ] Files committed and pushed to `main`
- [ ] Auto-tag job runs successfully (or create tag manually)
- [ ] Goreleaser job builds binaries
- [ ] GitLab Release created with assets
- [ ] Pages job publishes artifacts
- [ ] Can download binaries from Pages URL

## 🎯 Next Release

After merging commits to `main`:
1. Auto-tag job runs automatically
2. Creates next tag (e.g., `v0.1.1` for patch, `v0.2.0` for minor)
3. Goreleaser builds and publishes
4. Pages updates with new binaries

No manual steps needed! 🚀
