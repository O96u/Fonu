#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
# Transitional import: preserve the existing npm resolutions until the pnpm lock is committed.
if [[ ! -f pnpm-lock.yaml ]]; then
  test -f package-lock.json
  pnpm import
fi
pnpm install --frozen-lockfile
