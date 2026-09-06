#!/bin/sh
set -eu

entry=${CSS_ENTRY:-web/src/css/app.css}
output=${CSS_OUTPUT:-web/dist/css/app.css}
source_dir=$(dirname "$entry")
output_dir=$(dirname "$output")

if [ ! -f "$entry" ]; then
  echo "CSS entrypoint not found: $entry" >&2
  exit 1
fi

imports=$(sed -n 's/^@import "\.\/\([^"]*\)";$/\1/p' "$entry")
declared=$(grep -c '^@import ' "$entry" || true)
parsed=$(printf '%s\n' "$imports" | sed '/^$/d' | wc -l | tr -d ' ')

if [ "$declared" -eq 0 ]; then
  echo "CSS entrypoint has no imports: $entry" >&2
  exit 1
fi
if [ "$declared" -ne "$parsed" ]; then
  echo "CSS entrypoint contains an invalid import: $entry" >&2
  exit 1
fi

mkdir -p "$output_dir"
: >"$output"

printf '%s\n' "$imports" | while IFS= read -r relative; do
  [ -n "$relative" ] || continue
  case "$relative" in
  /* | *../* | ../* | *'/..')
    echo "CSS import must stay below $source_dir: $relative" >&2
    exit 1
    ;;
  esac

  source_file="$source_dir/$relative"
  if [ ! -f "$source_file" ]; then
    echo "CSS import not found: $source_file" >&2
    exit 1
  fi
  cat "$source_file" >>"$output"
done

# The source partials are build inputs only. Keep the embedded distribution to
# the single stylesheet referenced by the templates.
printf '%s\n' "$imports" | while IFS= read -r relative; do
  [ -n "$relative" ] || continue
  rm -f "$output_dir/$relative"
done
find "$output_dir" -depth -type d -empty -delete 2>/dev/null || true
