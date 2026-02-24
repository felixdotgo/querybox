#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

# Builds all plugin folders under ./plugins/ into ./bin/plugins/
# Usage: bash ./scripts/build-plugins.sh
# Export GOOS/GOARCH to cross-compile (optional)

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLUGINS_DIR="$ROOT_DIR/plugins"
OUT_DIR="$ROOT_DIR/bin/plugins"

if [ ! -d "$PLUGINS_DIR" ]; then
  echo "No plugins directory found at $PLUGINS_DIR"
  exit 0
fi

mkdir -p "$OUT_DIR"

echo "Building plugins from $PLUGINS_DIR -> $OUT_DIR"

built=0
# report target platform for clarity
echo "Target GOOS=${GOOS:-$(go env GOOS)} GOARCH=${GOARCH:-$(go env GOARCH)}"
for d in "$PLUGINS_DIR"/*; do
  [ -d "$d" ] || continue
  name="$(basename "$d")"

  # Skip example/template plugin (not intended for production build)
  if [ "$name" = "template" ]; then
    echo "- Skipping template plugin (example)"
    continue
  fi

  # Build only if plugin contains a `main.go` file — use that file directly
  if [ -f "./plugins/$name/main.go" ]; then
    build_target="./plugins/$name/main.go"
  else
    echo "- Skipping $name (no main.go)"
    continue
  fi

  # determine final output path; on Windows add .exe if not already
  out_path="$OUT_DIR/$name"

  # decide GOOS for extension detection; respect explicit env var first,
  # otherwise try `go env` then fallback to shell platform hints
  goos=${GOOS:-}
  if [ -z "$goos" ]; then
    goos=$(go env GOOS)
  fi
  # shells like Git Bash/MSYS/WSL may misreport; check uname for Windows indicators
  case "$(uname -s)" in
    *MINGW*|*MSYS*|*CYGWIN*|*Microsoft*)
      goos=windows
      ;;
  esac

  if [ "$goos" = "windows" ] && [[ "$out_path" != *.exe ]]; then
    out_path="${out_path}.exe"
  fi

  echo "- Building $name -> $out_path"
  if GOOS=${GOOS:-} GOARCH=${GOARCH:-} go build -o "$out_path" "${build_target:-./plugins/$name}"; then
    # make executable and preserve extension
    chmod +x "$out_path" || true
    built=$((built+1))
  else
    echo "  Failed to build $name" >&2
  fi
done

echo "Built $built plugin(s)."
