#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"

while IFS= read -r gomod; do
  dir="$(dirname "$gomod")"
  rel="${dir#$ROOT/}"
  echo "==> gosec ${rel}"
  (cd "$dir" && gosec -quiet ./...)
done < <(find "$ROOT" -name go.mod -not -path '*/node_modules/*' | sort)
