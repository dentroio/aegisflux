#!/usr/bin/env bash
set -u

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DOCKER="${DOCKER_BIN:-docker}"
if ! command -v "$DOCKER" >/dev/null 2>&1 && [ -x /usr/local/bin/docker ]; then
  DOCKER=/usr/local/bin/docker
fi

ACTIONS_URL="${ACTIONS_URL:-http://localhost:8083}"
INGEST_URL="${INGEST_URL:-http://localhost:9091}"
DETECTION_URL="${DETECTION_URL:-http://localhost:8089}"
LOCAL_AGENT_URL="${LOCAL_AGENT_URL:-http://localhost:18080}"
EXPECTED_AGENTS="${EXPECTED_AGENTS:-windows-dev-agent-01 linux-dev-agent-01}"
AGENT_STALE_SECONDS="${AGENT_STALE_SECONDS:-180}"
WINDOWS_HOST="${AEGIS_WINDOWS_HOST:-}"
WINDOWS_USER="${AEGIS_WINDOWS_USER:-aegis}"
WINDOWS_SSH_KEY="${AEGIS_WINDOWS_SSH_KEY:-$HOME/.ssh/id_ed25519}"
LINUX_HOST="${AEGIS_LINUX_HOST:-}"
LINUX_USER="${AEGIS_LINUX_USER:-clarion}"
LINUX_SSH_KEY="${AEGIS_LINUX_SSH_KEY:-$HOME/.ssh/id_ed25519}"

failures=0

ok() {
  printf '[ok] %s\n' "$1"
}

warn() {
  printf '[warn] %s\n' "$1"
}

fail() {
  printf '[fail] %s\n' "$1"
  failures=$((failures + 1))
}

check_http() {
  name="$1"
  url="$2"
  if body="$(curl -fsS "$url" 2>/dev/null)"; then
    ok "$name $url -> $body"
  else
    fail "$name $url is not reachable"
  fi
}

check_remote_ingest() {
  name="$1"
  user="$2"
  host="$3"
  key="$4"
  if [ -z "$host" ]; then
    warn "$name ingest reachability skipped (host not configured)"
    return
  fi

  if /usr/bin/ssh \
    -i "$key" \
    -T \
    -o BatchMode=yes \
    -o StrictHostKeyChecking=accept-new \
    -o UserKnownHostsFile=/private/tmp/aegisflux_known_hosts \
    -o ConnectTimeout=8 \
    "${user}@${host}" \
    "curl -fsS --max-time 5 http://127.0.0.1:9091/healthz" >/tmp/aegisflux-remote-health.$$ 2>/tmp/aegisflux-remote-health.err.$$; then
    ok "$name ingest from ${user}@${host} -> $(cat /tmp/aegisflux-remote-health.$$)"
  else
    fail "$name ingest is not reachable from ${user}@${host}: $(cat /tmp/aegisflux-remote-health.err.$$)"
  fi
  rm -f /tmp/aegisflux-remote-health.$$ /tmp/aegisflux-remote-health.err.$$
}

check_remote_tunnel_forwards() {
  name="$1"
  user="$2"
  host="$3"
  key="$4"
  if [ -z "$host" ]; then
    warn "$name tunnel-forward health skipped (host not configured)"
    return
  fi

  if /usr/bin/ssh \
    -i "$key" \
    -T \
    -o BatchMode=yes \
    -o StrictHostKeyChecking=accept-new \
    -o UserKnownHostsFile=/private/tmp/aegisflux_known_hosts \
    -o ConnectTimeout=8 \
    "${user}@${host}" \
    "for port in 9091 8083 8089; do ss -ltnp 2>/dev/null | awk -v suffix=\":\$port\" '\$4 ~ suffix \"\$\" && /sshd-session/ { found=1 } END { if (found) print \"ok\"; else print \"missing\" }'; done" >/tmp/aegisflux-remote-tunnel.$$ 2>/tmp/aegisflux-remote-tunnel.err.$$; then
    if awk 'BEGIN { good=1 } $0 != "ok" { good=0 } END { exit good ? 0 : 1 }' /tmp/aegisflux-remote-tunnel.$$; then
      ok "$name tunnel forwards (9091,8083,8089) are present on ${user}@${host}"
    else
      fail "$name tunnel forward listeners are incomplete on ${user}@${host}"
    fi
  else
    fail "$name tunnel-forward check failed on ${user}@${host}: $(cat /tmp/aegisflux-remote-tunnel.err.$$)"
  fi
  rm -f /tmp/aegisflux-remote-tunnel.$$ /tmp/aegisflux-remote-tunnel.err.$$
}

check_launchd() {
  label="$1"
  if ! command -v launchctl >/dev/null 2>&1; then
    warn "launchctl not available; skipped $label"
    return
  fi

  if output="$(launchctl print "gui/$(id -u)/$label" 2>/dev/null)"; then
    state="$(printf '%s\n' "$output" | awk -F'= ' '/state =/ {print $2; exit}')"
    if [ "$state" = "running" ]; then
      ok "$label is running"
    else
      fail "$label state is ${state:-unknown}"
    fi
  else
    fail "$label is not loaded"
  fi
}

cd "$ROOT_DIR" || exit 1

printf 'AegisFlux lab check\n'
printf 'Repo: %s\n\n' "$ROOT_DIR"

if "$DOCKER" compose ps >/tmp/aegisflux-compose-ps.$$ 2>/tmp/aegisflux-compose-ps.err.$$; then
  if awk 'NR > 1 && $0 !~ / Up / { bad=1 } END { exit bad ? 1 : 0 }' /tmp/aegisflux-compose-ps.$$; then
    ok "all compose services report Up"
  else
    fail "one or more compose services are not Up"
    cat /tmp/aegisflux-compose-ps.$$
  fi
else
  fail "docker compose ps failed: $(cat /tmp/aegisflux-compose-ps.err.$$)"
fi
rm -f /tmp/aegisflux-compose-ps.$$ /tmp/aegisflux-compose-ps.err.$$

check_http "actions-api" "$ACTIONS_URL/healthz"
check_http "ingest" "$INGEST_URL/healthz"
check_http "detection-pipeline" "$DETECTION_URL/healthz"
check_http "local-agent" "$LOCAL_AGENT_URL/healthz"

if "$DOCKER" compose exec -T websocket-gateway wget -qO- http://127.0.0.1:8080/health >/tmp/aegisflux-ws-health.$$ 2>/dev/null; then
  ok "websocket-gateway container health -> $(cat /tmp/aegisflux-ws-health.$$)"
else
  fail "websocket-gateway health is not reachable inside the container"
fi
rm -f /tmp/aegisflux-ws-health.$$

check_launchd net.aegis.windows-reverse-tunnel
check_launchd net.aegis.linux-reverse-tunnel

check_remote_tunnel_forwards "windows-lab" "$WINDOWS_USER" "$WINDOWS_HOST" "$WINDOWS_SSH_KEY"
check_remote_tunnel_forwards "linux-lab" "$LINUX_USER" "$LINUX_HOST" "$LINUX_SSH_KEY"
check_remote_ingest "windows-lab" "$WINDOWS_USER" "$WINDOWS_HOST" "$WINDOWS_SSH_KEY"
check_remote_ingest "linux-lab" "$LINUX_USER" "$LINUX_HOST" "$LINUX_SSH_KEY"

if agents_json="$(curl -fsS "$ACTIONS_URL/agents" 2>/dev/null)"; then
  printf '\nAgents:\n%s\n' "$agents_json"
  for agent in $EXPECTED_AGENTS; do
    if printf '%s' "$agents_json" | python3 -c 'import json,sys,time,datetime; data=json.load(sys.stdin); uid=sys.argv[1]; stale=int(sys.argv[2]); now=time.time(); agents=data.get("agents") or []; agent=next((a for a in agents if a.get("agent_uid")==uid), None); 
if not agent or agent.get("status")!="online": sys.exit(1)
last_seen_raw=agent.get("last_seen")
if not last_seen_raw: sys.exit(2)
try:
    ts=last_seen_raw.replace("Z","+00:00")
    seen=datetime.datetime.fromisoformat(ts).timestamp()
except Exception:
    sys.exit(3)
sys.exit(0 if now-seen <= stale else 4)' "$agent" "$AGENT_STALE_SECONDS"; then
      ok "$agent is online and fresh"
    else
      fail "$agent is missing, stale, or not online"
    fi
  done
else
  fail "could not fetch agents from $ACTIONS_URL/agents"
fi

printf '\n'
if [ "$failures" -eq 0 ]; then
  ok "AegisFlux lab looks healthy"
else
  fail "$failures check(s) failed"
fi

exit "$failures"
