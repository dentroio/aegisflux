#!/usr/bin/env bash
# WO-OPS-007: coarse latency sample for ingest summary routes (repeatable lab command).
set -euo pipefail

INGEST_URL="${INGEST_URL:-http://127.0.0.1:9091}"
DEVICE="${DEVICE:-lab-replay-device}"

for path in \
  "/v1/visibility/summary/dashboard" \
  "/v1/visibility/summary/device?device_id=$DEVICE" \
  "/v1/visibility/summary/inventory?device_id=$DEVICE"; do
  url="$INGEST_URL$path"
  printf '%s ' "$url"
  curl -fsS -o /dev/null -w 'http_code=%{http_code} time_total=%{time_total}s\n' --max-time 30 "$url"
done
