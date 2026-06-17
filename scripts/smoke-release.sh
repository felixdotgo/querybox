#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

red() { printf '\033[31m%s\033[0m\n' "$*" >&2; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
bold() { printf '\033[1m%s\033[0m\n' "$*"; }
die() { red "Error: $*"; exit 1; }

REQUIRE_APP_ARTIFACT=0
if [[ "${1:-}" == "--require-app-artifact" ]]; then
  REQUIRE_APP_ARTIFACT=1
elif [[ -n "${1:-}" ]]; then
  die "Usage: $0 [--require-app-artifact]"
fi

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "Required command not found: $1"
}

read_app_version() {
  python3 - <<'PY'
from pathlib import Path

in_info = False
for line in Path("build/config.yml").read_text(encoding="utf-8").splitlines():
    if line.startswith("info:"):
        in_info = True
        continue
    if in_info and line and not line.startswith((" ", "\t")):
        break
    if in_info and line.lstrip().startswith("version:"):
        value = line.split(":", 1)[1].split("#", 1)[0].strip().strip("'\"")
        print(value)
        break
else:
    raise SystemExit("build/config.yml info.version not found")
PY
}

read_registry_plugins() {
  python3 - <<'PY'
import json
from pathlib import Path

data = json.loads(Path("plugins/registry.json").read_text(encoding="utf-8"))
for name in sorted(data.get("plugins", {})):
    print(name)
PY
}

validate_manifest_taxonomy() {
  python3 - <<'PY'
import json
from pathlib import Path

allowed = {
    "resource.graph",
    "query.execute",
    "stream.read",
    "connection.test",
    "schema.inspect",
    "query.explain",
    "row.mutate",
    "row.mutate.edit",
    "row.mutate.delete",
}

errors = []
for path in sorted(Path("plugins").glob("*/plugin.json")):
    data = json.loads(path.read_text(encoding="utf-8"))
    for cap in data.get("capabilities", []):
        if cap not in allowed:
            errors.append(f"{path}: unsupported capability {cap!r}")

if errors:
    print("\n".join(errors))
    raise SystemExit(1)
PY
}

validate_built_plugin_artifacts() {
  local plugin
  local goos
  local binary
  local manifest

  goos="${GOOS:-$(go env GOOS)}"
  while IFS= read -r plugin; do
    binary="bin/plugins/$plugin"
    if [[ "$goos" == "windows" ]]; then
      binary="$binary.exe"
    fi
    manifest="$binary.manifest.json"

    [[ -f "$binary" ]] || die "Missing bundled plugin binary: $binary"
    [[ -f "$manifest" ]] || die "Missing bundled plugin manifest: $manifest"
    if [[ "$goos" != "windows" && ! -x "$binary" ]]; then
      die "Bundled plugin is not executable: $binary"
    fi

    python3 - "$plugin" "plugins/$plugin/plugin.json" "$manifest" <<'PY'
import json
import sys
from pathlib import Path

plugin, source_path, manifest_path = sys.argv[1:4]
source = json.loads(Path(source_path).read_text(encoding="utf-8"))
sidecar = json.loads(Path(manifest_path).read_text(encoding="utf-8"))
if sidecar.get("id") != plugin:
    raise SystemExit(f"{manifest_path}: id {sidecar.get('id')!r} does not match {plugin!r}")
if source != sidecar:
    raise SystemExit(f"{manifest_path}: sidecar manifest differs from {source_path}")
PY
  done < <(read_registry_plugins)
}

check_app_artifact() {
  local candidates=(
    "bin/querybox"
    "bin/querybox.exe"
    "bin/QueryBox"
    "bin/querybox.app"
  )
  local candidate

  for candidate in "${candidates[@]}"; do
    if [[ -e "$candidate" ]]; then
      echo "Found app artifact: $candidate"
      return 0
    fi
  done

  if [[ "$REQUIRE_APP_ARTIFACT" -eq 1 ]]; then
    die "No app artifact found under bin/. Run wails3 build before the required artifact smoke."
  fi

  echo "No app artifact found under bin/; skipping artifact presence check."
  echo "Run with --require-app-artifact after wails3 build for release-candidate artifact sanity."
}

require_cmd python3
require_cmd go

APP_VERSION="$(read_app_version)"
[[ "$APP_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "build/config.yml info.version must be semver, got '$APP_VERSION'"

bold "Release smoke: QueryBox $APP_VERSION"

echo "Validating plugin metadata..."
validate_manifest_taxonomy
while IFS= read -r plugin; do
  bash scripts/check-plugin-release.sh "$plugin"
done < <(read_registry_plugins)

echo "Building bundled plugins..."
build_output="$(bash scripts/build-plugins.sh 2>&1)"
printf '%s\n' "$build_output"
if grep -q "Failed to build" <<<"$build_output"; then
  die "Bundled plugin build failed."
fi
validate_built_plugin_artifacts

echo "Running targeted smoke tests..."
go test ./services/pluginmgr ./plugins/redis ./pkg/plugin

check_app_artifact

green "Release smoke checks passed."
