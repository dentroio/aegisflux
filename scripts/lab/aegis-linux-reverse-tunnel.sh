#!/usr/bin/env bash
set -euo pipefail

LINUX_HOST="${AEGIS_LINUX_HOST:-192.168.101.31}"
LINUX_USER="${AEGIS_LINUX_USER:-clarion}"
SSH_KEY="${AEGIS_LINUX_SSH_KEY:-$HOME/.ssh/aegis_windows_lab}"
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
  "${LINUX_USER}@${LINUX_HOST}"
