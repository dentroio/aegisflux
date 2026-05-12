#!/usr/bin/env bash
set -euo pipefail

REPO_AGENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SOURCE_BIN="${SOURCE_BIN:-${REPO_AGENT_DIR}/target/release/aegis-linux-agent}"
INSTALL_DIR="${INSTALL_DIR:-/opt/aegis/linux-agent}"
SERVICE_NAME="${SERVICE_NAME:-aegis-linux-agent}"
UNIT_PATH="/etc/systemd/system/${SERVICE_NAME}.service"
BACKEND_URL="${AEGIS_BACKEND_URL:-}"
ACTIONS_HEARTBEAT_URL="${AEGIS_ACTIONS_HEARTBEAT_URL:-http://127.0.0.1:8083/agents/heartbeat}"
AGENT_ID="${AEGIS_AGENT_ID:-linux-agent-prod}"
DEVICE_ID="${AEGIS_DEVICE_ID:-linux-agent-prod}"
COLLECTION_INTERVAL_SECONDS="${AEGIS_COLLECTION_INTERVAL_SECONDS:-60}"

if [[ "${EUID}" -ne 0 ]]; then
  echo "install-systemd.sh must be run as root" >&2
  exit 1
fi

if [[ ! "${SERVICE_NAME}" =~ ^[A-Za-z0-9_.@-]+$ ]]; then
  echo "SERVICE_NAME contains unsupported characters: ${SERVICE_NAME}" >&2
  exit 1
fi

if [[ ! -x "${SOURCE_BIN}" ]]; then
  echo "agent binary not found or not executable: ${SOURCE_BIN}" >&2
  exit 1
fi

if ! id -u aegis >/dev/null 2>&1; then
  useradd --system --home-dir /var/lib/aegis --shell /usr/sbin/nologin aegis
fi

install -d -m 0755 "${INSTALL_DIR}" /var/lib/aegis/linux-agent /var/log/aegis
chown -R aegis:aegis /var/lib/aegis/linux-agent /var/log/aegis
install -m 0755 -o root -g root "${SOURCE_BIN}" "${INSTALL_DIR}/aegis-linux-agent"

cat >"${UNIT_PATH}" <<SERVICE
[Unit]
Description=Aegis Linux Agent
Documentation=https://github.com/sgerhart/aegisflux
After=network-online.target
Wants=network-online.target
StartLimitIntervalSec=0

[Service]
Type=notify
NotifyAccess=main
User=aegis
Group=aegis
ExecStart=${INSTALL_DIR}/aegis-linux-agent
Restart=always
RestartSec=3s
WatchdogSec=90s
KillMode=process
TimeoutStopSec=15s
Environment=AEGIS_AGENT_ID=${AGENT_ID}
Environment=AEGIS_DEVICE_ID=${DEVICE_ID}
Environment=AEGIS_BACKEND_URL=${BACKEND_URL}
Environment=AEGIS_ACTIONS_HEARTBEAT_URL=${ACTIONS_HEARTBEAT_URL}
Environment=AEGIS_COLLECTION_INTERVAL_SECONDS=${COLLECTION_INTERVAL_SECONDS}
Environment=AEGIS_COLLECT_COMMAND_LINE=false
Environment=AEGIS_EVENT_SPOOL=/var/lib/aegis/linux-agent/events.jsonl
Environment=AEGIS_DETECTION_PACKS_ENABLED=false
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/aegis/linux-agent /var/log/aegis
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictRealtime=true
RestrictSUIDSGID=true
RemoveIPC=true
LockPersonality=true
MemoryDenyWriteExecute=true
SystemCallArchitectures=native
RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6 AF_NETLINK
CapabilityBoundingSet=
AmbientCapabilities=
StandardOutput=journal
StandardError=journal
SyslogIdentifier=aegis-linux-agent

[Install]
WantedBy=multi-user.target
SERVICE

systemctl daemon-reload
systemctl enable --now "${SERVICE_NAME}.service"
systemctl --no-pager status "${SERVICE_NAME}.service"
