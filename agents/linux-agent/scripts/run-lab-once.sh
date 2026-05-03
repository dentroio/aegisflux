#!/usr/bin/env bash
set -euo pipefail

AGENT_PATH="${AEGIS_LINUX_AGENT_PATH:-/opt/aegis/linux-agent/aegis-linux-agent}"
LOG_PATH="${AEGIS_LINUX_AGENT_LOG:-/var/log/aegis/linux-agent.log}"

mkdir -p "$(dirname "${LOG_PATH}")" "$(dirname "${AEGIS_EVENT_SPOOL:-/var/lib/aegis/linux-agent/events.jsonl}")"

export AEGIS_AGENT_ID="${AEGIS_AGENT_ID:-linux-dev-agent-01}"
export AEGIS_DEVICE_ID="${AEGIS_DEVICE_ID:-linux-dev-agent-01}"
export AEGIS_BACKEND_URL="${AEGIS_BACKEND_URL:-http://127.0.0.1:9091}"
export AEGIS_COLLECT_COMMAND_LINE="${AEGIS_COLLECT_COMMAND_LINE:-true}"
export AEGIS_EVENT_SPOOL="${AEGIS_EVENT_SPOOL:-/var/lib/aegis/linux-agent/events.jsonl}"

started_at="$(date --iso-8601=seconds)"
if "${AGENT_PATH}" --once >>"${LOG_PATH}" 2>&1; then
  printf '[%s] completed\n' "${started_at}" >>"${LOG_PATH}"
else
  status=$?
  printf '[%s] failed with status %s\n' "${started_at}" "${status}" >>"${LOG_PATH}"
  exit "${status}"
fi
