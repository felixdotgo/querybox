#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

# Usage: scripts/release-plugin.sh <plugin-name>
# Example: scripts/release-plugin.sh mysql
#
# Version is read from plugins/registry.json.
# Both registry.json and the plugin's Info() must have matching versions before running this script.
#
# Creates the release tag locally. Push manually after review:
#   git push origin plugin-<name>-v<version>

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# ── helpers ───────────────────────────────────────────────────────────────────
red()   { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
bold()  { printf '\033[1m%s\033[0m\n' "$*"; }
die()   { red "Error: $*"; exit 1; }

# ── args ──────────────────────────────────────────────────────────────────────
PLUGIN="${1:-}"
[[ -n "$PLUGIN" ]] || die "Usage: $0 <plugin-name>  (e.g. $0 mysql)"

PLUGIN_DIR="plugins/$PLUGIN"
[[ -d "$PLUGIN_DIR" ]]       || die "Plugin directory '$PLUGIN_DIR' not found."
[[ -f "$PLUGIN_DIR/main.go" ]] || die "No main.go found in '$PLUGIN_DIR'."

# ── read versions ─────────────────────────────────────────────────────────────
REGISTRY_VERSION="$(python3 -c "
import json, sys
try:
    d = json.load(open('plugins/registry.json'))
    print(d['plugins']['$PLUGIN']['version'])
except KeyError:
    sys.exit(1)
" 2>/dev/null)" || die "Plugin '$PLUGIN' not found in plugins/registry.json. Add an entry before releasing."

INFO_VERSION="$(grep -oE 'Version:[[:space:]]*"[0-9]+\.[0-9]+\.[0-9]+"' "$PLUGIN_DIR/main.go" \
  | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')" \
  || die "Could not find Version field in $PLUGIN_DIR/main.go."

# ── validate consistency ──────────────────────────────────────────────────────
if [[ "$REGISTRY_VERSION" != "$INFO_VERSION" ]]; then
  die "Version mismatch — both must match before releasing:
  plugins/registry.json  →  $REGISTRY_VERSION
  $PLUGIN_DIR/main.go   →  $INFO_VERSION"
fi

VERSION="$REGISTRY_VERSION"
TAG="plugin-$PLUGIN-v$VERSION"

# ── git checks ────────────────────────────────────────────────────────────────
BRANCH="$(git rev-parse --abbrev-ref HEAD)"
[[ "$BRANCH" == "main" ]] || die "Must be on main branch (currently on '$BRANCH')"

[[ -z "$(git status --porcelain)" ]] || die "Working tree is not clean. Commit or stash changes first."

git fetch --tags --quiet

[[ -z "$(git tag -l "$TAG")" ]] || die "Tag '$TAG' already exists."

# ── show changes since last tag ───────────────────────────────────────────────
LAST_TAG="$(git describe --tags --abbrev=0 --match="plugin-$PLUGIN-v*" 2>/dev/null || echo '')"

echo ""
bold "Plugin release: $TAG"
echo ""
printf "  %-24s %s\n" "Plugin:" "$PLUGIN"
printf "  %-24s %s\n" "Version:" "$VERSION"
printf "  %-24s %s ✓\n" "registry.json:" "$REGISTRY_VERSION"
printf "  %-24s %s ✓\n" "Info() in main.go:" "$INFO_VERSION"
echo ""

if [[ -n "$LAST_TAG" ]]; then
  echo "Changes since $LAST_TAG:"
  git log --oneline "$LAST_TAG..HEAD" -- "$PLUGIN_DIR" plugins/registry.json
else
  echo "No previous tag found for plugin '$PLUGIN' — this will be the first release."
  git log --oneline -- "$PLUGIN_DIR" plugins/registry.json | head -20
fi

# ── confirm ───────────────────────────────────────────────────────────────────
echo ""
read -r -p "Create local tag $TAG? [y/N] " confirm
[[ "$confirm" =~ ^[Yy]$ ]] || { echo "Aborted."; exit 1; }

# ── tag ───────────────────────────────────────────────────────────────────────
git tag "$TAG"

echo ""
green "Tag $TAG created locally."
echo ""
echo "Review the tag, then push to trigger the release:"
echo "  git push origin $TAG"
