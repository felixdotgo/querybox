#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

# Usage: scripts/release-app.sh <version>
# Example: scripts/release-app.sh v1.2.3
#
# Creates the release tag locally. Push manually after review:
#   git push origin <version>

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# ── helpers ───────────────────────────────────────────────────────────────────
red()   { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
bold()  { printf '\033[1m%s\033[0m\n' "$*"; }
die()   { red "Error: $*"; exit 1; }
fetch_tags() {
  local output
  local conflicting_tag

  echo "Fetching tags from remote..."
  if ! output="$(git fetch --tags 2>&1)"; then
    [[ -z "$output" ]] || red "$output"
    if [[ "$output" =~ \-\>\ ([^[:space:]]+)[[:space:]]+\(would\ clobber\ existing\ tag\) ]]; then
      conflicting_tag="${BASH_REMATCH[1]}"
      red "Local tag '$conflicting_tag' differs from the remote tag."
      red "Inspect it with: git show-ref --tags $conflicting_tag && git ls-remote --tags origin $conflicting_tag"
      red "If the remote tag is correct, delete the local tag with: git tag -d $conflicting_tag"
    fi
    die "Could not fetch tags. Resolve the Git tag issue above, then rerun this script."
  fi
}

# ── args ──────────────────────────────────────────────────────────────────────
VERSION="${1:-}"
[[ -n "$VERSION" ]] || die "Usage: $0 <version>  (e.g. v1.2.3)"
[[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "Version must be in format v1.2.3 (with leading 'v')"

# ── git checks ────────────────────────────────────────────────────────────────
BRANCH="$(git rev-parse --abbrev-ref HEAD)"
[[ "$BRANCH" == "main" ]] || die "Must be on main branch (currently on '$BRANCH')"

[[ -z "$(git status --porcelain)" ]] || die "Working tree is not clean. Commit or stash changes first."

fetch_tags

[[ -z "$(git tag -l "$VERSION")" ]] || die "Tag '$VERSION' already exists."

# ── show changes since last tag ───────────────────────────────────────────────
LAST_TAG="$(git tag --sort=-version:refname -l 'v[0-9]*' | head -1)"

echo ""
bold "App release: $VERSION"
echo ""
if [[ -n "$LAST_TAG" ]]; then
  echo "Changes since $LAST_TAG:"
  git log --oneline "$LAST_TAG..HEAD" || true
else
  echo "No previous app tag found — this will be the first release."
  git log --oneline | head -20 || true
fi

# ── confirm ───────────────────────────────────────────────────────────────────
echo ""
read -r -p "Create local tag $VERSION? [y/N] " confirm
[[ "$confirm" =~ ^[Yy]$ ]] || { echo "Aborted."; exit 1; }

# ── tag ───────────────────────────────────────────────────────────────────────
git tag "$VERSION"

echo ""
green "Tag $VERSION created locally."
echo ""
echo "Review the tag, then push to trigger the release:"
echo "  git push origin $VERSION"
