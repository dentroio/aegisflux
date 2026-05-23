#!/usr/bin/env bash
set -euo pipefail

LABEL="${AEGIS_TUNNEL_LAUNCHD_LABEL:-net.aegis.windows-reverse-tunnel}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RUNNER="${REPO_ROOT}/scripts/lab/aegis-windows-reverse-tunnel.sh"
PLIST="${HOME}/Library/LaunchAgents/${LABEL}.plist"
LOG_DIR="${HOME}/Library/Logs/Aegis"
UID_VALUE="$(id -u)"
WINDOWS_HOST="${AEGIS_WINDOWS_HOST:-192.168.12.101}"
WINDOWS_USER="${AEGIS_WINDOWS_USER:-aegis}"
WINDOWS_SSH_KEY="${AEGIS_WINDOWS_SSH_KEY:-$HOME/.ssh/id_ed25519}"
LOG_BASENAME="${LABEL##*.}"

if [[ ! -x "${RUNNER}" ]]; then
  chmod +x "${RUNNER}"
fi

mkdir -p "$(dirname "${PLIST}")" "${LOG_DIR}"

cat > "${PLIST}" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>${LABEL}</string>
  <key>ProgramArguments</key>
  <array>
    <string>${RUNNER}</string>
  </array>
  <key>EnvironmentVariables</key>
  <dict>
    <key>AEGIS_WINDOWS_HOST</key>
    <string>${WINDOWS_HOST}</string>
    <key>AEGIS_WINDOWS_USER</key>
    <string>${WINDOWS_USER}</string>
    <key>AEGIS_WINDOWS_SSH_KEY</key>
    <string>${WINDOWS_SSH_KEY}</string>
  </dict>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <dict>
    <key>SuccessfulExit</key>
    <false/>
  </dict>
  <key>StandardOutPath</key>
  <string>${LOG_DIR}/${LOG_BASENAME}.out.log</string>
  <key>StandardErrorPath</key>
  <string>${LOG_DIR}/${LOG_BASENAME}.err.log</string>
  <key>WorkingDirectory</key>
  <string>${REPO_ROOT}</string>
</dict>
</plist>
PLIST

launchctl bootout "gui/${UID_VALUE}" "${PLIST}" >/dev/null 2>&1 || true
launchctl bootstrap "gui/${UID_VALUE}" "${PLIST}"
launchctl enable "gui/${UID_VALUE}/${LABEL}"
launchctl kickstart -k "gui/${UID_VALUE}/${LABEL}"

launchctl print "gui/${UID_VALUE}/${LABEL}" | sed -n '1,80p'
