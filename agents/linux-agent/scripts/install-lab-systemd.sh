#!/usr/bin/env bash
set -euo pipefail

REPO_AGENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SOURCE_BIN="${SOURCE_BIN:-${REPO_AGENT_DIR}/target/release/aegis-linux-agent}"
INSTALL_DIR="${INSTALL_DIR:-/opt/aegis/linux-agent}"
SERVICE_NAME="${SERVICE_NAME:-aegis-linux-agent-lab}"
BACKEND_URL="${AEGIS_BACKEND_URL:-http://127.0.0.1:9091}"
AGENT_ID="${AEGIS_AGENT_ID:-linux-dev-agent-01}"
DEVICE_ID="${AEGIS_DEVICE_ID:-linux-dev-agent-01}"

if [[ ! -x "${SOURCE_BIN}" ]]; then
  echo "agent binary not found or not executable: ${SOURCE_BIN}" >&2
  exit 1
fi

install -d -m 0755 "${INSTALL_DIR}" /var/lib/aegis/linux-agent /var/log/aegis
install -m 0755 "${SOURCE_BIN}" "${INSTALL_DIR}/aegis-linux-agent"
install -m 0755 "${REPO_AGENT_DIR}/scripts/run-lab-once.sh" "${INSTALL_DIR}/run-lab-once.sh"

cat >/etc/systemd/system/${SERVICE_NAME}.service <<SERVICE
[Unit]
Description=Aegis Linux Agent lab collection
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
Environment=AEGIS_LINUX_AGENT_PATH=${INSTALL_DIR}/aegis-linux-agent
Environment=AEGIS_AGENT_ID=${AGENT_ID}
Environment=AEGIS_DEVICE_ID=${DEVICE_ID}
Environment=AEGIS_BACKEND_URL=${BACKEND_URL}
Environment=AEGIS_COLLECT_COMMAND_LINE=true
Environment=AEGIS_EVENT_SPOOL=/var/lib/aegis/linux-agent/events.jsonl
ExecStart=${INSTALL_DIR}/run-lab-once.sh
SERVICE

cat >/etc/systemd/system/${SERVICE_NAME}.timer <<TIMER
[Unit]
Description=Run Aegis Linux Agent lab collection every minute

[Timer]
OnBootSec=30s
OnUnitActiveSec=60s
AccuracySec=5s
Unit=${SERVICE_NAME}.service

[Install]
WantedBy=timers.target
TIMER

systemctl daemon-reload
systemctl enable --now "${SERVICE_NAME}.timer"
systemctl start "${SERVICE_NAME}.service"
systemctl --no-pager status "${SERVICE_NAME}.timer"
