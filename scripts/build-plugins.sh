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

  out_path="$OUT_DIR/$name"
  echo "- Building $name -> $out_path"
  if GOOS=${GOOS:-} GOARCH=${GOARCH:-} go build -o "$out_path" "${build_target:-./plugins/$name}"; then
    chmod +x "$out_path" || true
    built=$((built+1))
  else
    echo "  Failed to build $name" >&2
  fi
done

echo "Built $built plugin(s)."
