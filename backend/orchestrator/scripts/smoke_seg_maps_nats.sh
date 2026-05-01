#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

NATS_CLIENT_PORT="${NATS_CLIENT_PORT:-14222}"
NATS_MONITOR_PORT="${NATS_MONITOR_PORT:-18222}"
NEO4J_HTTP_PORT="${NEO4J_HTTP_PORT:-17474}"
NEO4J_BOLT_PORT="${NEO4J_BOLT_PORT:-17687}"
ORCHESTRATOR_HTTP_PORT="${ORCHESTRATOR_HTTP_PORT:-18084}"
ORCHESTRATOR_BASE_URL="${ORCHESTRATOR_BASE_URL:-http://localhost:${ORCHESTRATOR_HTTP_PORT}}"

compose() {
  NATS_CLIENT_PORT="${NATS_CLIENT_PORT}" \
  NATS_MONITOR_PORT="${NATS_MONITOR_PORT}" \
  NEO4J_HTTP_PORT="${NEO4J_HTTP_PORT}" \
  NEO4J_BOLT_PORT="${NEO4J_BOLT_PORT}" \
  ORCHESTRATOR_HTTP_PORT="${ORCHESTRATOR_HTTP_PORT}" \
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

cleanup() {
  if [[ "${AEGIS_SMOKE_STOP_AFTER:-false}" == "true" ]]; then
    compose stop orchestrator nats >/dev/null
  fi
}
trap cleanup EXIT

cd "${ROOT_DIR}"

echo "Starting orchestrator MapSnapshot smoke stack..."
compose up -d --build --no-deps nats orchestrator >/dev/null

echo "Waiting for orchestrator health..."
wait_for_http "${ORCHESTRATOR_BASE_URL}/healthz" "orchestrator"

echo "Posting MapSnapshot and validating actions.seg.maps..."
(
  cd "${ROOT_DIR}/backend/orchestrator"
  ORCHESTRATOR_BASE_URL="${ORCHESTRATOR_BASE_URL}" \
  NATS_URL="nats://localhost:${NATS_CLIENT_PORT}" \
  go run ./cmd/smoke-seg-maps-nats
)
