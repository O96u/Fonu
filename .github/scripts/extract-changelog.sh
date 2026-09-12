#!/usr/bin/env bash
set -euo pipefail

version="${1:?usage: extract-changelog.sh <version> [output-file]}"
output="${2:-release-body.md}"
changelog="CHANGELOG.md"

if [ ! -f "$changelog" ]; then
  echo "CHANGELOG.md not found" >&2
  exit 1
fi

awk -v ver="$version" '
  $0 ~ "^## \\[" ver "\\]" { found = 1; next }
  found && /^## \[v/ { exit }
  found { print }
' "$changelog" > "$output"

if [ ! -s "$output" ]; then
  echo "No changelog section for ${version}" >&2
  rm -f "$output"
  exit 1
fi

echo "Extracted release notes for ${version} ($(wc -l < "$output") lines)"
