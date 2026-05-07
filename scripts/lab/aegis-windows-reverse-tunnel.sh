#!/usr/bin/env bash
set -euo pipefail

WINDOWS_HOST="${AEGIS_WINDOWS_HOST:-192.168.12.101}"
WINDOWS_USER="${AEGIS_WINDOWS_USER:-aegis}"
SSH_KEY="${AEGIS_WINDOWS_SSH_KEY:-$HOME/.ssh/id_ed25519}"
REMOTE_PORT="${AEGIS_TUNNEL_REMOTE_PORT:-9091}"
LOCAL_HOST="${AEGIS_TUNNEL_LOCAL_HOST:-127.0.0.1}"
LOCAL_PORT="${AEGIS_TUNNEL_LOCAL_PORT:-9091}"
ACTIONS_REMOTE_PORT="${AEGIS_ACTIONS_TUNNEL_REMOTE_PORT:-8083}"
ACTIONS_LOCAL_PORT="${AEGIS_ACTIONS_TUNNEL_LOCAL_PORT:-8083}"
DETECTION_REMOTE_PORT="${AEGIS_DETECTION_TUNNEL_REMOTE_PORT:-8089}"
DETECTION_LOCAL_PORT="${AEGIS_DETECTION_TUNNEL_LOCAL_PORT:-8089}"
CLEAN_STALE="${AEGIS_TUNNEL_CLEAN_STALE:-true}"

cleanup_stale_remote_forwards() {
  if [[ "${CLEAN_STALE}" != "true" ]]; then
    return 0
  fi

  /usr/bin/ssh \
    -i "$SSH_KEY" \
    -T \
    -o StrictHostKeyChecking=accept-new \
    -o UserKnownHostsFile=/private/tmp/aegisflux_known_hosts \
    -o ConnectTimeout=8 \
    "${WINDOWS_USER}@${WINDOWS_HOST}" \
    "powershell.exe -NoProfile -ExecutionPolicy Bypass -Command \"\$ports=@(${REMOTE_PORT},${ACTIONS_REMOTE_PORT},${DETECTION_REMOTE_PORT}); Get-NetTCPConnection -State Listen -ErrorAction SilentlyContinue | Where-Object { \$ports -contains \$_.LocalPort } | Select-Object -ExpandProperty OwningProcess -Unique | ForEach-Object { Stop-Process -Id \$_ -Force -ErrorAction SilentlyContinue }\"" || true
}

cleanup_stale_remote_forwards

exec /usr/bin/ssh \
  -i "$SSH_KEY" \
  -N \
  -T \
  -o StrictHostKeyChecking=accept-new \
  -o UserKnownHostsFile=/private/tmp/aegisflux_known_hosts \
  -o ExitOnForwardFailure=yes \
  -o ServerAliveInterval=30 \
  -o ServerAliveCountMax=3 \
  -R "127.0.0.1:${REMOTE_PORT}:${LOCAL_HOST}:${LOCAL_PORT}" \
  -R "127.0.0.1:${ACTIONS_REMOTE_PORT}:${LOCAL_HOST}:${ACTIONS_LOCAL_PORT}" \
  -R "127.0.0.1:${DETECTION_REMOTE_PORT}:${LOCAL_HOST}:${DETECTION_LOCAL_PORT}" \
  "${WINDOWS_USER}@${WINDOWS_HOST}"
