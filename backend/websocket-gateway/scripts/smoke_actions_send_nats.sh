#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

NATS_CLIENT_PORT="${NATS_CLIENT_PORT:-14222}"
NATS_MONITOR_PORT="${NATS_MONITOR_PORT:-18222}"
NEO4J_HTTP_PORT="${NEO4J_HTTP_PORT:-17474}"
NEO4J_BOLT_PORT="${NEO4J_BOLT_PORT:-17687}"
GATEWAY_BASE_URL="${GATEWAY_BASE_URL:-http://localhost:8080}"
ACTIONS_API_BASE_URL="${ACTIONS_API_BASE_URL:-http://localhost:8083}"
ACTIONS_API_URL="${ACTIONS_API_URL:-http://actions-api:8083}"

compose() {
  NATS_CLIENT_PORT="${NATS_CLIENT_PORT}" \
  NATS_MONITOR_PORT="${NATS_MONITOR_PORT}" \
  NEO4J_HTTP_PORT="${NEO4J_HTTP_PORT}" \
  NEO4J_BOLT_PORT="${NEO4J_BOLT_PORT}" \
  ACTIONS_API_URL="${ACTIONS_API_URL}" \
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
    compose stop websocket-gateway actions-api timescale nats >/dev/null
  fi
}
trap cleanup EXIT

cd "${ROOT_DIR}"

echo "Starting Actions API, WebSocket Gateway, and NATS smoke stack..."
compose up -d --build nats actions-api websocket-gateway >/dev/null

echo "Waiting for Actions API and WebSocket Gateway health..."
wait_for_http "${ACTIONS_API_BASE_URL}/healthz" "actions-api"
wait_for_http "${GATEWAY_BASE_URL}/health" "websocket-gateway"

echo "Posting Actions API send request and validating websocket.messages..."
(
  cd "${ROOT_DIR}/backend/actions-api"
  ACTIONS_API_BASE_URL="${ACTIONS_API_BASE_URL}" \
  NATS_URL="nats://localhost:${NATS_CLIENT_PORT}" \
  go run ./cmd/smoke-actions-nats
)
