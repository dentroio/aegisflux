#!/usr/bin/env bash
# WO-OPS-004: POST sample visibility JSONL into local ingest (lab replay).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
INGEST_URL="${INGEST_URL:-http://127.0.0.1:9091}"
FIXTURE="${1:-$ROOT/fixtures/visibility/lab-replay-sample.ndjson}"

if ! test -f "$FIXTURE"; then
  echo "fixture not found: $FIXTURE" >&2
  exit 1
fi

curl -fsS --max-time 30 \
  -H "Content-Type: application/x-ndjson" \
  --data-binary @"$FIXTURE" \
  "$INGEST_URL/v1/visibility/events"
printf '\nReplay complete: %s -> %s\n' "$FIXTURE" "$INGEST_URL"
