#!/usr/bin/env bash
set -euo pipefail

LINUX_HOST="${AEGIS_LINUX_HOST:-192.168.101.31}"
LINUX_USER="${AEGIS_LINUX_USER:-clarion}"
SSH_KEY="${AEGIS_LINUX_SSH_KEY:-$HOME/.ssh/id_ed25519}"
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
    "${LINUX_USER}@${LINUX_HOST}" \
    sh -s -- "$REMOTE_PORT" "$ACTIONS_REMOTE_PORT" "$DETECTION_REMOTE_PORT" <<'REMOTE' || true
for port in "$@"; do
  if command -v sudo >/dev/null 2>&1; then
    listeners="$(sudo -n ss -ltnp 2>/dev/null || ss -ltnp 2>/dev/null || true)"
  else
    listeners="$(ss -ltnp 2>/dev/null || true)"
  fi

  printf '%s\n' "$listeners" |
    awk -v suffix=":$port" '$4 ~ suffix "$" && /sshd-session/ { print }' |
    sed -n 's/.*pid=\([0-9][0-9]*\).*/\1/p' |
    sort -u |
    while read -r pid; do
      [ -n "$pid" ] || continue
      kill "$pid" 2>/dev/null || true
    done
done
REMOTE
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
  -R "${REMOTE_PORT}:${LOCAL_HOST}:${LOCAL_PORT}" \
  -R "${ACTIONS_REMOTE_PORT}:${LOCAL_HOST}:${ACTIONS_LOCAL_PORT}" \
  -R "${DETECTION_REMOTE_PORT}:${LOCAL_HOST}:${DETECTION_LOCAL_PORT}" \
  "${LINUX_USER}@${LINUX_HOST}"
