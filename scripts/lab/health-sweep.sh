#!/usr/bin/env bash
# WO-OPS-003: curl-based health sweep for local compose ports (host view).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

ACTIONS_URL="${ACTIONS_URL:-http://127.0.0.1:8083}"
INGEST_URL="${INGEST_URL:-http://127.0.0.1:9091}"
DETECTION_URL="${DETECTION_URL:-http://127.0.0.1:8089}"
ETL_URL="${ETL_URL:-http://127.0.0.1:8088}"
CONFIG_URL="${CONFIG_URL:-http://127.0.0.1:8085}"
CORR_URL="${CORR_URL:-http://127.0.0.1:8082}"
DECISION_URL="${DECISION_URL:-http://127.0.0.1:8087}"

fail=0
check() {
  local name="$1" url="$2"
  if out="$(curl -fsS --max-time 5 "$url" 2>/dev/null)"; then
    printf '[ok] %s %s\n' "$name" "$url"
    if test "${VERBOSE:-0}" = "1"; then
      printf '    %s\n' "$out" | head -c 300
      printf '\n'
    fi
  else
    printf '[fail] %s %s\n' "$name" "$url"
    fail=$((fail + 1))
  fi
}

printf 'Health sweep (WO-OPS-003)\n\n'

check "actions-api /healthz" "$ACTIONS_URL/healthz"
check "actions-api /readyz" "$ACTIONS_URL/readyz"
check "actions-api /ops/metrics" "$ACTIONS_URL/ops/metrics"
check "ingest /healthz" "$INGEST_URL/healthz"
check "ingest /readyz" "$INGEST_URL/readyz"
check "detection-pipeline /healthz" "$DETECTION_URL/healthz"
check "etl-enrich /healthz" "$ETL_URL/healthz"
check "etl-enrich /readyz" "$ETL_URL/readyz"
check "config-api /healthz" "$CONFIG_URL/healthz"
check "correlator /healthz" "$CORR_URL/healthz"
check "decision /healthz" "$DECISION_URL/healthz"

if test "$fail" -ne 0; then
  printf '\n%d check(s) failed\n' "$fail"
  exit 1
fi
printf '\nAll checks passed\n'
