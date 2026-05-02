#!/usr/bin/env bash
set -euo pipefail

WINDOWS_HOST="${AEGIS_WINDOWS_HOST:-192.168.12.101}"
WINDOWS_USER="${AEGIS_WINDOWS_USER:-aegis}"
SSH_KEY="${AEGIS_WINDOWS_SSH_KEY:-$HOME/.ssh/aegis_windows_lab}"
REMOTE_PORT="${AEGIS_TUNNEL_REMOTE_PORT:-9091}"
LOCAL_HOST="${AEGIS_TUNNEL_LOCAL_HOST:-127.0.0.1}"
LOCAL_PORT="${AEGIS_TUNNEL_LOCAL_PORT:-9091}"

exec /usr/bin/ssh \
  -i "$SSH_KEY" \
  -N \
  -T \
  -o ExitOnForwardFailure=yes \
  -o ServerAliveInterval=30 \
  -o ServerAliveCountMax=3 \
  -R "${REMOTE_PORT}:${LOCAL_HOST}:${LOCAL_PORT}" \
  "${WINDOWS_USER}@${WINDOWS_HOST}"
