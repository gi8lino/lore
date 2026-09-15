#!/bin/sh
set -eu

# Mermaid's standalone UMD build is self-contained, so only the package archive
# is needed when updating the plugin. Keep the version pinned for reproducible assets.
MERMAID_VERSION="${MERMAID_VERSION:-12.0.0}"
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
MERMAID_OUTPUT_DIR="${MERMAID_OUTPUT_DIR:-$SCRIPT_DIR/assets}"
NPM="${NPM:-npm}"

workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT INT TERM

archive=$(
  "$NPM" pack \
    --silent \
    --ignore-scripts \
    --pack-destination "$workdir" \
    "mermaid@$MERMAID_VERSION"
)

mkdir -p "$MERMAID_OUTPUT_DIR"
tar -xOf "$workdir/$archive" package/dist/mermaid.min.js >"$MERMAID_OUTPUT_DIR/mermaid.min.js"
tar -xOf "$workdir/$archive" package/LICENSE >"$MERMAID_OUTPUT_DIR/LICENSE"
