#!/usr/bin/env bash
set -euo pipefail

LABEL="${AEGIS_TUNNEL_LAUNCHD_LABEL:-net.aegis.linux-reverse-tunnel}"
PLIST="${HOME}/Library/LaunchAgents/${LABEL}.plist"
UID_VALUE="$(id -u)"

launchctl bootout "gui/${UID_VALUE}" "${PLIST}" >/dev/null 2>&1 || true
rm -f "${PLIST}"
echo "Removed ${LABEL}."
