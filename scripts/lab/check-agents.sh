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

if agents_json="$(curl -fsS "$ACTIONS_URL/agents" 2>/dev/null)"; then
  printf '\nAgents:\n%s\n' "$agents_json"
  for agent in $EXPECTED_AGENTS; do
    if printf '%s' "$agents_json" | python3 -c 'import json,sys; data=json.load(sys.stdin); uid=sys.argv[1]; sys.exit(0 if any(a.get("agent_uid")==uid and a.get("status")=="online" for a in data.get("agents") or []) else 1)' "$agent"; then
      ok "$agent is online"
    else
      fail "$agent is missing or not online"
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
