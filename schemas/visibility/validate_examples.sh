#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required to validate example JSON syntax" >&2
  exit 1
fi
shopt -s nullglob
for f in "$ROOT"/examples/*.json; do
  jq -e . "$f" >/dev/null
  echo "OK $f"
done
