#!/usr/bin/env bash
# WO-OPS-001: End-to-end lab pipeline smoke — endpoint signal → summaries → platform APIs → audit lifecycle.
# Observe-only / audit-only: no enforcement. Run from repo root with compose up (and optional lab agents).
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

ACTIONS_URL="${ACTIONS_URL:-http://127.0.0.1:8083}"
INGEST_URL="${INGEST_URL:-http://127.0.0.1:9091}"
DETECTION_URL="${DETECTION_URL:-http://127.0.0.1:8089}"
LINUX_AGENT_UID="${LINUX_AGENT_UID:-linux-dev-agent-01}"
WINDOWS_AGENT_UID="${WINDOWS_AGENT_UID:-windows-dev-agent-01}"
REPLAY_DEVICE="${REPLAY_DEVICE:-lab-replay-device}"

SKIP_HEALTH="${SKIP_HEALTH:-0}"
SKIP_LAB_AGENT_ASSERT="${SKIP_LAB_AGENT_ASSERT:-0}"
RUN_DETECTION_ROLLOUT="${RUN_DETECTION_ROLLOUT:-0}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

fail() {
  printf '[fail] %s\n' "$1" >&2
  exit 1
}

ok() {
  printf '[ok] %s\n' "$1"
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "missing command: $1"
}

require_cmd curl
require_cmd python3

if [[ "$SKIP_HEALTH" != "1" ]] && [[ -x "${ROOT_DIR}/scripts/lab/health-sweep.sh" ]]; then
  "${ROOT_DIR}/scripts/lab/health-sweep.sh"
else
  ok "health sweep skipped (SKIP_HEALTH=${SKIP_HEALTH} or script not executable)"
fi

# --- Actions API readiness (dependency lens) ---
READY_FILE="${TMP_DIR}/readyz.json"
curl -fsS --max-time 10 "${ACTIONS_URL}/readyz" -o "${READY_FILE}" || fail "actions-api /readyz unreachable"
python3 - "$READY_FILE" <<'PY' || fail "actions-api /readyz not json"
import json, sys
with open(sys.argv[1]) as f:
    json.load(f)
PY
ok "actions-api /readyz returns JSON"

# --- Agent registry / heartbeat path (read-only check) ---
AGENTS_FILE="${TMP_DIR}/agents.json"
curl -fsS --max-time 15 "${ACTIONS_URL}/agents" -o "${AGENTS_FILE}" || fail "GET /agents unreachable"
python3 - "$AGENTS_FILE" "$SKIP_LAB_AGENT_ASSERT" "$LINUX_AGENT_UID" "$WINDOWS_AGENT_UID" <<'PY'
import json, sys
path, skip, linux_uid, win_uid = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4]
with open(path) as f:
    d = json.load(f)
agents = d.get("agents")
if not isinstance(agents, list):
    raise SystemExit("agents is not a list")
if skip == "1":
    raise SystemExit(0)
uids = {a.get("agent_uid") for a in agents}
# At least one lab UID present when agents are expected online
if linux_uid not in uids and win_uid not in uids:
    raise SystemExit(f"expected at least one of {linux_uid}, {win_uid} in agent_uid set; got {uids!r}")
PY
ok "GET /agents (lab UIDs present or SKIP_LAB_AGENT_ASSERT=1)"

# --- Workbench summary merge (ingest dependency surfaced) ---
WB_FILE="${TMP_DIR}/workbench.json"
curl -fsS --max-time 25 "${ACTIONS_URL}/console/summary/agents-workbench" -o "${WB_FILE}" || fail "agents-workbench unreachable"
python3 - "$WB_FILE" <<'PY' || fail "agents-workbench shape"
import json, sys
with open(sys.argv[1]) as f:
    d = json.load(f)
if "agents" not in d or "total" not in d:
    raise SystemExit("missing agents/total")
if "dependencies" not in d:
    raise SystemExit("missing dependencies (WO-OPS-003 merge diagnostics)")
for dep in d["dependencies"]:
    if dep.get("name") == "ingest" and dep.get("status") != "ok":
        raise SystemExit(f"ingest dependency not ok: {dep!r}")
PY
ok "GET /console/summary/agents-workbench (ingest dependency ok)"

# --- Readiness fleet ---
curl -fsS --max-time 25 "${ACTIONS_URL}/console/summary/agent-readiness" -o "${TMP_DIR}/readiness.json" || fail "agent-readiness unreachable"
python3 - "${TMP_DIR}/readiness.json" <<'PY' || fail "agent-readiness shape"
import json, sys
with open(sys.argv[1]) as f:
    d = json.load(f)
if "agents" not in d:
    raise SystemExit("missing agents")
PY
ok "GET /console/summary/agent-readiness"

# --- Visibility ingest replay (canonical fixture) ---
if [[ -x "${ROOT_DIR}/scripts/lab/replay-visibility-fixtures.sh" ]]; then
  "${ROOT_DIR}/scripts/lab/replay-visibility-fixtures.sh"
else
  FIX="${ROOT_DIR}/fixtures/visibility/lab-replay-sample.ndjson"
  curl -fsS --max-time 30 -H "Content-Type: application/x-ndjson" --data-binary @"${FIX}" \
    "${INGEST_URL}/v1/visibility/events" >/dev/null || fail "ingest replay failed"
fi
ok "visibility events replayed to ingest"

# --- Ingest summaries (ABOM / inventory path) ---
for path in \
  "/v1/visibility/summary/dashboard" \
  "/v1/visibility/summary/device?device_id=${REPLAY_DEVICE}" \
  "/v1/visibility/summary/inventory?device_id=${REPLAY_DEVICE}"; do
  out="${TMP_DIR}/sum-$(echo "$path" | tr '/?=' '---').json"
  curl -fsS --max-time 30 "${INGEST_URL}${path}" -o "$out" || fail "ingest summary failed: $path"
  python3 - "$out" <<'PY' || fail "summary response missing ok:true"
import json, sys
with open(sys.argv[1]) as f:
    d = json.load(f)
if d.get("ok") is not True:
    raise SystemExit(f"expected ok true, got {d.get('ok')!r}")
PY
  ok "ingest GET $path"
done

# --- Detection pack status (per-agent, detection-pipeline) ---
PACK_URL="${DETECTION_URL}/v1/agents/${LINUX_AGENT_UID}/detection-pack-status"
curl -fsS --max-time 10 "$PACK_URL" -o "${TMP_DIR}/pack-status.json" || fail "detection-pack-status unreachable"
python3 - "${TMP_DIR}/pack-status.json" <<'PY' || fail "detection-pack-status shape"
import json, sys
with open(sys.argv[1]) as f:
    d = json.load(f)
if "status" not in d:
    raise SystemExit("missing status key (null allowed when no rollout)")
PY
ok "GET detection-pack-status (${LINUX_AGENT_UID})"

# --- Operational events (platform store) ---
curl -fsS --max-time 10 "${ACTIONS_URL}/platform/operational-events?limit=50" -o "${TMP_DIR}/op-events.json" || fail "operational-events unreachable"
python3 - "${TMP_DIR}/op-events.json" <<'PY' || fail "operational-events shape"
import json, sys
with open(sys.argv[1]) as f:
    d = json.load(f)
if "events" not in d or not isinstance(d["events"], list):
    raise SystemExit("missing events list")
PY
ok "GET /platform/operational-events"

# --- Detection opportunities / candidates list ---
curl -fsS --max-time 10 "${ACTIONS_URL}/platform/detection-candidates" -o "${TMP_DIR}/candidates.json" || fail "detection-candidates unreachable"
python3 - "${TMP_DIR}/candidates.json" <<'PY' || fail "detection-candidates shape"
import json, sys
with open(sys.argv[1]) as f:
    d = json.load(f)
if "candidates" not in d:
    raise SystemExit("missing candidates")
PY
ok "GET /platform/detection-candidates"

curl -fsS --max-time 10 "${ACTIONS_URL}/platform/research-feed" -o "${TMP_DIR}/research.json" || fail "research-feed unreachable"
python3 - "${TMP_DIR}/research.json" <<'PY' || fail "research-feed shape"
import json, sys
with open(sys.argv[1]) as f:
    d = json.load(f)
if "items" not in d:
    raise SystemExit("missing items")
PY
ok "GET /platform/research-feed"

# --- Audit-mode bundle lifecycle (observe-only contract) ---
create_body='{"title":"WO-OPS-001 lab smoke bundle","description":"E2E smoke audit bundle","scope":["device:linux-lab-1"],"expected_match_telemetry":["process.listen_port==11434"],"rollback_notes":"Revoke after smoke"}'
curl -fsS --max-time 15 -X POST -H "Content-Type: application/json" \
  --data-binary "$create_body" "${ACTIONS_URL}/platform/audit-bundles" -o "${TMP_DIR}/audit-create.json" || fail "audit bundle create"
python3 - "${TMP_DIR}/audit-create.json" <<'PY' || fail "audit create response"
import json, sys
with open(sys.argv[1]) as f:
    d = json.load(f)
if d.get("status") != "draft":
    raise SystemExit(f"expected draft, got {d.get('status')!r}")
if d.get("mode") != "audit":
    raise SystemExit(f"expected audit mode, got {d.get('mode')!r}")
PY
BUNDLE_ID="$(python3 -c "import json; print(json.load(open('${TMP_DIR}/audit-create.json'))['id'])")"
ok "POST /platform/audit-bundles (draft audit)"

curl -fsS --max-time 15 -X POST -H "Content-Type: application/json" \
  --data-binary '{"device_ids":["linux-lab-1"]}' \
  "${ACTIONS_URL}/platform/audit-bundles/${BUNDLE_ID}/stage" -o "${TMP_DIR}/audit-stage.json" || fail "audit stage"
python3 - "${TMP_DIR}/audit-stage.json" <<'PY' || fail "audit staged"
import json, sys
with open(sys.argv[1]) as f:
    d = json.load(f)
if d.get("status") != "staged":
    raise SystemExit(f"expected staged, got {d.get('status')!r}")
PY
ok "POST .../audit-bundles/{id}/stage"

curl -fsS --max-time 15 -X POST -H "Content-Type: application/json" \
  --data-binary '{"device_id":"linux-lab-1","status":"accepted","agent_version":"0.1.0"}' \
  "${ACTIONS_URL}/platform/audit-bundles/${BUNDLE_ID}/status" -o "${TMP_DIR}/audit-status.json" || fail "audit status"
ok "POST .../audit-bundles/{id}/status (accepted)"

curl -fsS --max-time 15 -X POST -H "Content-Type: application/json" \
  --data-binary '{"device_id":"linux-lab-1","process":"lab-smoke","indicator":"listen_port==11434","detail":"observe-only match"}' \
  "${ACTIONS_URL}/platform/audit-bundles/${BUNDLE_ID}/match" -o "${TMP_DIR}/audit-match.json" || fail "audit match"
ok "POST .../audit-bundles/{id}/match"

curl -fsS --max-time 15 -X POST -H "Content-Type: application/json" \
  --data-binary '{"note":"WO-OPS-001 smoke revoke"}' \
  "${ACTIONS_URL}/platform/audit-bundles/${BUNDLE_ID}/revoke" -o "${TMP_DIR}/audit-revoke.json" || fail "audit revoke"
python3 - "${TMP_DIR}/audit-revoke.json" <<'PY' || fail "audit revoked"
import json, sys
with open(sys.argv[1]) as f:
    d = json.load(f)
if d.get("status") != "revoked":
    raise SystemExit(f"expected revoked, got {d.get('status')!r}")
PY
ok "POST .../audit-bundles/{id}/revoke (no enforcement)"

# --- Optional: full detection rollout smoke (WO-DET-006) ---
if [[ "$RUN_DETECTION_ROLLOUT" == "1" ]]; then
  if [[ -x "${ROOT_DIR}/scripts/lab/smoke-detection-rollout.sh" ]]; then
    "${ROOT_DIR}/scripts/lab/smoke-detection-rollout.sh"
    ok "smoke-detection-rollout.sh complete"
  else
    fail "RUN_DETECTION_ROLLOUT=1 but smoke-detection-rollout.sh missing or not executable"
  fi
else
  ok "detection rollout smoke skipped (set RUN_DETECTION_ROLLOUT=1 to enable WO-DET-006 path)"
fi

printf '\nWO-OPS-001 e2e smoke: all stages passed\n'
printf 'Manual: follow docs/ops/E2E_PIPELINE_SMOKE.md console checklist when UI validation is required.\n'
