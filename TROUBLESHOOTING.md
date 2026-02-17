# Troubleshooting: GitLab Token Authentication Failed

## Error: "HTTP Basic: Access denied"

This means the `GITLAB_TOKEN` either:
1. ❌ Doesn't have the right scopes
2. ❌ Isn't set in CI/CD Variables
3. ❌ Is protected but `main` branch isn't protected

---

## Fix 1: Verify Token Scopes

Your token **MUST** have these scopes:
- ✅ `read_repository` (for fetching tags)
- ✅ `write_repository` (for pushing tags)
- ✅ `api` (for goreleaser to create releases)

**To check/fix:**
1. Go to: `https://gitlab.com/-/user_settings/personal_access_tokens`
2. Find your token (or create a new one)
3. Ensure it has: `read_repository`, `write_repository`, `api`
4. If not, create a new token with all three scopes

---

## Fix 2: Verify CI/CD Variable is Set

1. Go to: `https://gitlab.com/swytchcode/cli/-/settings/ci_cd`
2. Scroll to **Variables**
3. Check if `GITLAB_TOKEN` exists
4. If not, add it:
   - Key: `GITLAB_TOKEN`
   - Value: Your token
   - ✅ Protect variable (if you want it only on protected branches)
   - ✅ Mask variable
   - ✅ Expand variable reference

**Important:** If you checked "Protect variable":
- The variable only works on **protected branches**
- Make sure `main` is a protected branch, OR
- Uncheck "Protect variable" to allow on all branches

---

## Fix 3: Check Repository Name

Verify your repo is actually named `cli`:
- Go to: `https://gitlab.com/swytchcode/cli`
- Check the URL matches

If your repo is still named `shell`, update `.gitlab-ci.yml`:
```yaml
- git remote set-url origin "https://oauth2:${GITLAB_TOKEN}@gitlab.com/swytchcode/shell.git"
```

---

## Fix 4: Use Project Access Token (Alternative)

Instead of Personal Access Token, use a **Project Access Token**:

1. Go to: `https://gitlab.com/swytchcode/cli/-/settings/access_tokens`
2. Click "Add new token"
3. Name: `ci-release-token`
4. Role: **Maintainer** (or higher)
5. Scopes: ✅ `read_repository`, ✅ `write_repository`, ✅ `api`
6. Expiration: Set as needed
7. Copy the token
8. Add to CI/CD Variables as `GITLAB_TOKEN`

**Advantage:** Project tokens are scoped to the project only.

---

## Quick Test: Verify Token Works

Test your token locally:

```bash
# Set token
export GITLAB_TOKEN="your-token-here"

# Test fetching (read_repository)
git ls-remote "https://oauth2:${GITLAB_TOKEN}@gitlab.com/swytchcode/cli.git"

# Should list branches/tags without error
```

If this fails, your token doesn't have `read_repository` scope.

---

## After Fixing: Re-run Pipeline

1. Go to: `https://gitlab.com/swytchcode/cli/-/pipelines`
2. Find the failed pipeline
3. Click "Retry" button

Or push a new commit to trigger a new pipeline.
