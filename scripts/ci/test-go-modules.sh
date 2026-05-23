#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"

failed=0
while IFS= read -r gomod; do
  dir="$(dirname "$gomod")"
  rel="${dir#$ROOT/}"
  echo "==> testing ${rel}"
  if ! (cd "$dir" && go test ./...); then
    failed=1
  fi
done < <(find "$ROOT" -name go.mod -not -path '*/node_modules/*' | sort)

if [[ "$failed" -ne 0 ]]; then
  echo "One or more Go modules failed tests."
  exit 1
fi

echo "All Go modules passed."
