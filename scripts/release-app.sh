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

# ── args ──────────────────────────────────────────────────────────────────────
VERSION="${1:-}"
[[ -n "$VERSION" ]] || die "Usage: $0 <version>  (e.g. v1.2.3)"
[[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "Version must be in format v1.2.3 (with leading 'v')"

# ── git checks ────────────────────────────────────────────────────────────────
BRANCH="$(git rev-parse --abbrev-ref HEAD)"
[[ "$BRANCH" == "main" ]] || die "Must be on main branch (currently on '$BRANCH')"

[[ -z "$(git status --porcelain)" ]] || die "Working tree is not clean. Commit or stash changes first."

git fetch --tags --quiet

[[ -z "$(git tag -l "$VERSION")" ]] || die "Tag '$VERSION' already exists."

# ── show changes since last tag ───────────────────────────────────────────────
LAST_TAG="$(git describe --tags --abbrev=0 --match='v[0-9]*' 2>/dev/null || echo '')"

echo ""
bold "App release: $VERSION"
echo ""
if [[ -n "$LAST_TAG" ]]; then
  echo "Changes since $LAST_TAG:"
  git log --oneline "$LAST_TAG..HEAD"
else
  echo "No previous app tag found — this will be the first release."
  git log --oneline | head -20
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
