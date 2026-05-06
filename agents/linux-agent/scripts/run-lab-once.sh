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
export AEGIS_DETECTION_PACKS_ENABLED="${AEGIS_DETECTION_PACKS_ENABLED:-false}"
export ACTIONS_HEARTBEAT_URL="${ACTIONS_HEARTBEAT_URL:-http://127.0.0.1:8083/agents/heartbeat}"

post_actions_heartbeat() {
  if command -v python3 >/dev/null 2>&1; then
    python3 - <<'PY' >/dev/null 2>&1 || true
import json
import os
import platform
import socket
from datetime import datetime, timezone
from urllib import request

host_id = os.environ.get("AEGIS_DEVICE_ID", "linux-dev-agent-01")
hostname = socket.gethostname() or host_id
primary_ip = os.environ.get("AEGIS_PRIMARY_IP", "")
if not primary_ip:
    probe = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    try:
        probe.connect(("192.0.2.1", 80))
        primary_ip = probe.getsockname()[0]
    except OSError:
        primary_ip = ""
    finally:
        probe.close()
payload = {
    "agent_uid": os.environ.get("AEGIS_AGENT_ID", "linux-dev-agent-01"),
    "org_id": os.environ.get("AEGIS_ORG_ID", "default-org"),
    "host_id": host_id,
    "hostname": hostname,
    "agent_version": os.environ.get("AEGIS_SENSOR_VERSION", "0.1.0"),
    "last_seen": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    "status": "online",
    "labels": ["visibility-lab", "linux"],
    "note": "Registered from Linux lab visibility collector",
    "capabilities": {
        "visibility": True,
        "dynamic_detection_packs": os.environ.get("AEGIS_DETECTION_PACKS_ENABLED", "").lower() == "true",
        "platform": "linux"
    },
    "platform": {
        "hostname": hostname,
        "os": "linux",
        "architecture": platform.machine() or "unknown",
        "kernel_version": platform.release() or "unknown",
        "primary_ip": primary_ip
    },
    "network": {
        "primary_ip": primary_ip
    }
}
body = json.dumps(payload).encode("utf-8")
req = request.Request(
    os.environ.get("ACTIONS_HEARTBEAT_URL", "http://127.0.0.1:8083/agents/heartbeat"),
    data=body,
    headers={"Content-Type": "application/json"},
    method="POST",
)
request.urlopen(req, timeout=5).read()
PY
  fi
}

started_at="$(date --iso-8601=seconds)"
if "${AGENT_PATH}" --once >>"${LOG_PATH}" 2>&1; then
  post_actions_heartbeat
  printf '[%s] completed\n' "${started_at}" >>"${LOG_PATH}"
else
  status=$?
  printf '[%s] failed with status %s\n' "${started_at}" "${status}" >>"${LOG_PATH}"
  exit "${status}"
fi
