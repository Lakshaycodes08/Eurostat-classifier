#!/usr/bin/env bash
set -euo pipefail

# Auto-tagging script for GitLab CI.
# - Runs on merge to main
# - Creates and pushes the next semver tag
# - First tag defaults to v0.1.0
#
# Version bump rules (based on commit messages since last tag):
# - MAJOR: contains "BREAKING CHANGE" or a conventional commit with "!:"
# - MINOR: contains a conventional commit starting with "feat"
# - PATCH: default

DEFAULT_FIRST_TAG="v0.1.0"

git fetch --tags --force >/dev/null 2>&1 || true

latest_tag="$(git tag -l 'v*' --sort=-v:refname | head -n 1 || true)"

if [[ -z "${latest_tag}" ]]; then
  next_tag="${DEFAULT_FIRST_TAG}"
else
  # Strip leading 'v' and split into parts
  version="${latest_tag#v}"
  IFS='.' read -r major minor patch <<<"${version}"

  # Find commits since last tag
  range="${latest_tag}..HEAD"
  commits="$(git log --pretty=format:%s "${range}" || true)"
  bodies="$(git log --pretty=format:%b "${range}" || true)"

  bump="patch"
  if echo "${commits}" | grep -E -q '!:|^.+\!:' || echo "${bodies}" | grep -q 'BREAKING CHANGE'; then
    bump="major"
  elif echo "${commits}" | grep -E -q '^feat(\(.+\))?:'; then
    bump="minor"
  fi

  case "${bump}" in
    major)
      major=$((major + 1))
      minor=0
      patch=0
      ;;
    minor)
      minor=$((minor + 1))
      patch=0
      ;;
    patch)
      patch=$((patch + 1))
      ;;
  esac

  next_tag="v${major}.${minor}.${patch}"
fi

if git rev-parse "${next_tag}" >/dev/null 2>&1; then
  echo "Tag ${next_tag} already exists; nothing to do."
  exit 0
fi

echo "Creating tag: ${next_tag}"
git tag -a "${next_tag}" -m "Release ${next_tag}"

echo "Pushing tag: ${next_tag}"
git push origin "${next_tag}"

