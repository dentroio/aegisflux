#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HTTP_ADDR="${INGEST_SMOKE_HTTP_ADDR:-127.0.0.1:19090}"
GRPC_ADDR="${INGEST_SMOKE_GRPC_ADDR:-127.0.0.1:15051}"
BASE_URL="http://${HTTP_ADDR}"
NATS_URL="${NATS_URL:-nats://localhost:4222}"
STORE_PATH="${AEGIS_VISIBILITY_STORE_PATH:-/tmp/aegisflux-smoke-visibility-events.jsonl}"
FIXTURE="${1:-${ROOT_DIR}/testdata/visibility/investigation_path.jsonl}"
DEVICE_ID="FIXTURE-DEVICE-01"
PROCESS_GUID="fixture-proc-001"

rm -f "${STORE_PATH}"

pushd "${ROOT_DIR}" >/dev/null
AEGIS_VISIBILITY_STORE_PATH="${STORE_PATH}" \
NATS_URL="${NATS_URL}" \
INGEST_HTTP_ADDR="${HTTP_ADDR}" \
INGEST_GRPC_ADDR="${GRPC_ADDR}" \
go run ./cmd/ingest >/tmp/aegisflux-ingest-smoke.log 2>&1 &
PID=$!
popd >/dev/null

cleanup() {
  kill "${PID}" >/dev/null 2>&1 || true
  wait "${PID}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

for _ in $(seq 1 30); do
  if curl -fsS "${BASE_URL}/readyz" >/dev/null 2>&1; then
    break
  fi
  sleep 0.25
done

curl -fsS "${BASE_URL}/readyz" >/dev/null
curl -fsS -X POST --data-binary "@${FIXTURE}" "${BASE_URL}/v1/visibility/events" >/tmp/aegisflux-smoke-post.json
curl -fsS "${BASE_URL}/v1/visibility/investigation?device_id=${DEVICE_ID}&process_guid=${PROCESS_GUID}&limit=20" >/tmp/aegisflux-smoke-investigation.json

python3 - <<'PY'
import json
from pathlib import Path

post = json.loads(Path('/tmp/aegisflux-smoke-post.json').read_text())
if post.get('accepted') != 5:
    raise SystemExit(f"expected 5 accepted events, got {post}")

result = json.loads(Path('/tmp/aegisflux-smoke-investigation.json').read_text())
counts = result.get('counts', {})
expected = {'processes': 1, 'flows': 1, 'dns': 1, 'findings': 2}
if counts != expected:
    raise SystemExit(f"expected counts {expected}, got {counts}")
print(json.dumps({'ok': True, 'counts': counts}, sort_keys=True))
PY
