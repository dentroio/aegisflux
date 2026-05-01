#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

NATS_CLIENT_PORT="${NATS_CLIENT_PORT:-14222}"
NATS_MONITOR_PORT="${NATS_MONITOR_PORT:-18222}"
NEO4J_HTTP_PORT="${NEO4J_HTTP_PORT:-17474}"
NEO4J_BOLT_PORT="${NEO4J_BOLT_PORT:-17687}"
INGEST_BASE_URL="${INGEST_BASE_URL:-http://localhost:9091}"
ETL_BASE_URL="${ETL_BASE_URL:-http://localhost:8088}"

EVENT_SUFFIX="$(date +%Y%m%d%H%M%S)"
EVENT_ID="evt-smoke-connect-${EVENT_SUFFIX}"
HOST_ID="SMOKE-HOST-${EVENT_SUFFIX}"
DST_IP="203.0.113.10"
DST_PORT="443"
EVENT_TS_MS="$(($(date +%s) * 1000))"
PAYLOAD_FILE="$(mktemp -t aegisflux-etl-smoke.XXXXXX.json)"

cleanup() {
  rm -f "${PAYLOAD_FILE}"
  if [[ "${AEGIS_SMOKE_STOP_AFTER:-false}" == "true" ]]; then
    compose stop etl-enrich ingest neo4j timescale nats >/dev/null
  fi
}
trap cleanup EXIT

compose() {
  NATS_CLIENT_PORT="${NATS_CLIENT_PORT}" \
  NATS_MONITOR_PORT="${NATS_MONITOR_PORT}" \
  NEO4J_HTTP_PORT="${NEO4J_HTTP_PORT}" \
  NEO4J_BOLT_PORT="${NEO4J_BOLT_PORT}" \
  docker compose "$@"
}

wait_for_http() {
  local url="$1"
  local label="$2"
  local attempts="${3:-60}"
  local delay="${4:-2}"

  for _ in $(seq 1 "${attempts}"); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      return 0
    fi
    sleep "${delay}"
  done

  echo "Timed out waiting for ${label}: ${url}" >&2
  return 1
}

metric_value() {
  local metric="$1"
  curl -fsS "${ETL_BASE_URL}/metrics" | awk -v metric="${metric}" '$1 == metric {print $2; found=1} END {if (!found) print "0"}'
}

query_timescale_count() {
  compose exec -T timescale psql -U postgres -d aegisflux -At -c \
    "SELECT COUNT(*) FROM events_raw WHERE host_id = '${HOST_ID}' AND event_type = 'connect';"
}

query_neo4j_edge_count() {
  compose exec -T neo4j cypher-shell -u neo4j -p password \
    "MATCH (h:Host {host_id:'${HOST_ID}'})-[r:COMMUNICATES]->(n:NetworkEndpoint {endpoint_id:'ip:${DST_IP}:${DST_PORT}'}) RETURN count(r);" \
    | tail -n 1 | tr -d '[:space:]'
}

cat >"${PAYLOAD_FILE}" <<JSON
{"schema_version":"visibility.v1","event_id":"${EVENT_ID}","event_type":"connect","timestamp_ms":${EVENT_TS_MS},"source":"aegis-smoke","device_id":"${HOST_ID}","agent_id":"smoke-agent","sensor_version":"0.1.0","sequence":1,"payload":{"dst_ip":"${DST_IP}","dst_port":${DST_PORT}}}
JSON

cd "${ROOT_DIR}"

echo "Starting Aegis ingest/ETL smoke stack..."
compose up -d nats timescale neo4j ingest etl-enrich >/dev/null

echo "Waiting for ingest and ETL readiness..."
wait_for_http "${INGEST_BASE_URL}/readyz" "ingest"
wait_for_http "${ETL_BASE_URL}/readyz" "etl-enrich"

before_processed="$(metric_value etl_enrich_processed_messages_total)"

echo "Posting smoke visibility event ${EVENT_ID}..."
post_response="$(curl -fsS -X POST \
  -H "Content-Type: application/x-ndjson" \
  --data-binary "@${PAYLOAD_FILE}" \
  "${INGEST_BASE_URL}/v1/visibility/events")"

python3 - "${post_response}" <<'PY'
import json
import sys

response = json.loads(sys.argv[1])
if response.get("accepted") != 1:
    raise SystemExit(f"expected one accepted event, got {response}")
PY

echo "Waiting for ETL to consume the event..."
for _ in $(seq 1 60); do
  after_processed="$(metric_value etl_enrich_processed_messages_total)"
  if [[ "${after_processed}" =~ ^[0-9]+$ ]] && (( after_processed > before_processed )); then
    break
  fi
  sleep 1
done

after_processed="$(metric_value etl_enrich_processed_messages_total)"
if ! [[ "${after_processed}" =~ ^[0-9]+$ ]] || (( after_processed <= before_processed )); then
  echo "ETL processed counter did not increment: before=${before_processed}, after=${after_processed}" >&2
  exit 1
fi

timescale_count="$(query_timescale_count)"
if [[ "${timescale_count}" != "1" ]]; then
  echo "Expected one Timescale events_raw row, got ${timescale_count}" >&2
  exit 1
fi

neo4j_count="$(query_neo4j_edge_count)"
if [[ "${neo4j_count}" != "1" ]]; then
  echo "Expected one Neo4j COMMUNICATES edge, got ${neo4j_count}" >&2
  exit 1
fi

python3 - <<PY
import json

print(json.dumps({
    "ok": True,
    "event_id": "${EVENT_ID}",
    "host_id": "${HOST_ID}",
    "etl_processed_before": int("${before_processed}"),
    "etl_processed_after": int("${after_processed}"),
    "timescale_rows": int("${timescale_count}"),
    "neo4j_edges": int("${neo4j_count}"),
}, sort_keys=True))
PY
