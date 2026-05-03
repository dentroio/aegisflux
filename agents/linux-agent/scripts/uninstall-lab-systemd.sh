#!/usr/bin/env bash
set -euo pipefail

SERVICE_NAME="${SERVICE_NAME:-aegis-linux-agent-lab}"

systemctl disable --now "${SERVICE_NAME}.timer" >/dev/null 2>&1 || true
systemctl stop "${SERVICE_NAME}.service" >/dev/null 2>&1 || true
rm -f "/etc/systemd/system/${SERVICE_NAME}.service" "/etc/systemd/system/${SERVICE_NAME}.timer"
systemctl daemon-reload
echo "Removed ${SERVICE_NAME} service and timer."
