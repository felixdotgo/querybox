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

  # Only build packages with 'package main'
  pkg_name=$(go list -f '{{.Name}}' "./plugins/$name" 2>/dev/null || true)
  if [ -z "$pkg_name" ] || [ "$pkg_name" != "main" ]; then
    echo "- Skipping $name (package: ${pkg_name:-unknown})"
    continue
  fi

  out_path="$OUT_DIR/$name"
  echo "- Building $name -> $out_path"
  if GOOS=${GOOS:-} GOARCH=${GOARCH:-} go build -o "$out_path" "./plugins/$name"; then
    chmod +x "$out_path" || true
    built=$((built+1))
  else
    echo "  Failed to build $name" >&2
  fi
done

echo "Built $built plugin(s)."
