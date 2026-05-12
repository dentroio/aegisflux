#!/usr/bin/env bash
set -euo pipefail

SERVICE_NAME="${SERVICE_NAME:-aegis-linux-agent}"
UNIT_PATH="/etc/systemd/system/${SERVICE_NAME}.service"

if [[ "${EUID}" -ne 0 ]]; then
  echo "uninstall-systemd.sh must be run as root" >&2
  exit 1
fi

if [[ ! "${SERVICE_NAME}" =~ ^[A-Za-z0-9_.@-]+$ ]]; then
  echo "SERVICE_NAME contains unsupported characters: ${SERVICE_NAME}" >&2
  exit 1
fi

systemctl disable --now "${SERVICE_NAME}.service" >/dev/null 2>&1 || true
rm -f "${UNIT_PATH}"
systemctl daemon-reload
echo "Removed ${SERVICE_NAME} service."
