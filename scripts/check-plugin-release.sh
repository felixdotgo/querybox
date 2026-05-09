#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

red()   { printf '\033[31m%s\033[0m\n' "$*" >&2; }
green() { printf '\033[32m%s\033[0m\n' "$*" >&2; }
die()   { red "Error: $*"; exit 1; }

PLUGIN="${1:-}"
EXPECTED_VERSION="${2:-}"

[[ -n "$PLUGIN" ]] || die "Usage: $0 <plugin-name> [expected-version]"

PLUGIN_DIR="plugins/$PLUGIN"
MANIFEST_PATH="$PLUGIN_DIR/plugin.json"
MAIN_GO="$PLUGIN_DIR/main.go"

[[ -d "$PLUGIN_DIR" ]] || die "Plugin directory '$PLUGIN_DIR' not found."
[[ -f "$MAIN_GO" ]] || die "No main.go found in '$PLUGIN_DIR'."
[[ -f "$MANIFEST_PATH" ]] || die "Missing manifest '$MANIFEST_PATH'."

REGISTRY_VERSION="$(python3 - "$PLUGIN" <<'PY'
import json
import sys

plugin = sys.argv[1]
with open("plugins/registry.json", "r", encoding="utf-8") as fh:
    data = json.load(fh)

try:
    print(data["plugins"][plugin]["version"])
except KeyError:
    raise SystemExit(1)
PY
)" || die "Plugin '$PLUGIN' not found in plugins/registry.json. Add an entry before releasing."

readarray -t manifest_fields < <(python3 - "$MANIFEST_PATH" <<'PY'
import json
import sys

path = sys.argv[1]
with open(path, "r", encoding="utf-8") as fh:
    data = json.load(fh)

print(data.get("id", ""))
print(data.get("version", ""))
PY
) || die "Could not read manifest fields from '$MANIFEST_PATH'."

MANIFEST_ID="${manifest_fields[0]:-}"
MANIFEST_VERSION="${manifest_fields[1]:-}"

[[ -n "$MANIFEST_ID" ]] || die "Manifest '$MANIFEST_PATH' is missing 'id'."
[[ -n "$MANIFEST_VERSION" ]] || die "Manifest '$MANIFEST_PATH' is missing 'version'."
[[ "$MANIFEST_ID" == "$PLUGIN" ]] || die "Manifest id '$MANIFEST_ID' does not match plugin directory '$PLUGIN'."

INFO_VERSION="$(
  grep -oE 'Version:[[:space:]]*"[0-9]+\.[0-9]+\.[0-9]+"' "$MAIN_GO" \
    | grep -oE '[0-9]+\.[0-9]+\.[0-9]+'
)" || die "Could not find Version field in $MAIN_GO."

[[ "$REGISTRY_VERSION" == "$MANIFEST_VERSION" ]] || die "Version mismatch: registry.json=$REGISTRY_VERSION, plugin.json=$MANIFEST_VERSION"
[[ "$INFO_VERSION" == "$MANIFEST_VERSION" ]] || die "Version mismatch: main.go=$INFO_VERSION, plugin.json=$MANIFEST_VERSION"

if [[ -n "$EXPECTED_VERSION" && "$MANIFEST_VERSION" != "$EXPECTED_VERSION" ]]; then
  die "Version mismatch: expected=$EXPECTED_VERSION, plugin.json=$MANIFEST_VERSION"
fi

green "Validated plugin release metadata for '$PLUGIN' (version $MANIFEST_VERSION)."
