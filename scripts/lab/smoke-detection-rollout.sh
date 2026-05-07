#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

DETECTION_URL="${DETECTION_URL:-http://127.0.0.1:8089}"
INGEST_URL="${INGEST_URL:-http://127.0.0.1:9091}"
ACTIONS_URL="${ACTIONS_URL:-http://127.0.0.1:8083}"
LAB_DEVICE_ID="${LAB_DEVICE_ID:-lab-mcp-01}"
LAB_TENANT_ID="${LAB_TENANT_ID:-}"
AGENT_VERSION="${AGENT_VERSION:-0.1.0}"
LINUX_AGENT_UID="${LINUX_AGENT_UID:-linux-dev-agent-01}"
WINDOWS_AGENT_UID="${WINDOWS_AGENT_UID:-windows-dev-agent-01}"
MAX_WAIT_SECONDS="${MAX_WAIT_SECONDS:-90}"
POLL_INTERVAL_SECONDS="${POLL_INTERVAL_SECONDS:-3}"

RESEARCH_FIXTURE="${ROOT_DIR}/schemas/detection/fixtures/wo-det-002/research_item.example.json"
CANDIDATE_FIXTURE="${ROOT_DIR}/schemas/detection/fixtures/wo-det-002/candidate_mcp_tool_bridge.example.json"
TELEMETRY_FIXTURE="${ROOT_DIR}/schemas/detection/fixtures/wo-det-002/lab-telemetry.json"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

fail_stage() {
  local stage="$1"
  local message="$2"
  local next_cmd="${3:-}"
  printf '[fail] %s: %s\n' "${stage}" "${message}" >&2
  if [[ -n "${next_cmd}" ]]; then
    printf 'next: %s\n' "${next_cmd}" >&2
  fi
  exit 1
}

ok_stage() {
  printf '[ok] %s\n' "$1"
}

require_file() {
  local path="$1"
  [[ -f "${path}" ]] || fail_stage "setup" "missing required file: ${path}"
}

curl_json() {
  local method="$1"
  local url="$2"
  local body_file="$3"
  local out_file="$4"
  local code
  if [[ -n "${body_file}" ]]; then
    code="$(curl -sS -X "${method}" -H 'Content-Type: application/json' --data-binary "@${body_file}" -o "${out_file}" -w '%{http_code}' "${url}")" || return 99
  else
    code="$(curl -sS -X "${method}" -o "${out_file}" -w '%{http_code}' "${url}")" || return 99
  fi
  printf '%s' "${code}"
}

json_get() {
  local file="$1"
  local expr="$2"
  python3 - "$file" "$expr" <<'PY'
import json
import sys

path, expr = sys.argv[1], sys.argv[2]
with open(path, "r", encoding="utf-8") as fh:
    data = json.load(fh)
value = eval(expr, {"__builtins__": {}}, {"d": data})
if isinstance(value, (dict, list)):
    print(json.dumps(value, separators=(",", ":")))
elif value is None:
    print("")
else:
    print(str(value))
PY
}

assert_json_true() {
  local file="$1"
  local expr="$2"
  local stage="$3"
  local hint="$4"
  python3 - "$file" "$expr" <<'PY'
import json
import sys
with open(sys.argv[1], "r", encoding="utf-8") as fh:
    data = json.load(fh)
safe_builtins = {
    "bool": bool,
    "dict": dict,
    "float": float,
    "int": int,
    "isinstance": isinstance,
    "len": len,
    "list": list,
    "str": str,
}
ok = bool(eval(sys.argv[2], {"__builtins__": safe_builtins}, {"d": data}))
raise SystemExit(0 if ok else 1)
PY
  if [[ $? -ne 0 ]]; then
    fail_stage "${stage}" "assertion failed: ${expr}" "${hint}"
  fi
}

health_check() {
  local name="$1"
  local url="$2"
  local out="${TMP_DIR}/health-${name}.txt"
  if ! curl -fsS "${url}" >"${out}"; then
    fail_stage "health" "${name} is unreachable at ${url}" "docker compose ps && curl -v ${url}"
  fi
  ok_stage "health ${name}"
}

require_file "${RESEARCH_FIXTURE}"
require_file "${CANDIDATE_FIXTURE}"
require_file "${TELEMETRY_FIXTURE}"

printf 'Detection rollout smoke\n'
printf 'repo=%s\n' "${ROOT_DIR}"
printf 'detection=%s ingest=%s actions=%s\n' "${DETECTION_URL}" "${INGEST_URL}" "${ACTIONS_URL}"

health_check "detection-pipeline" "${DETECTION_URL}/healthz"
health_check "ingest" "${INGEST_URL}/healthz"
health_check "actions-api" "${ACTIONS_URL}/healthz"

POST_RESP="${TMP_DIR}/ingest-post.json"
POST_CODE="$(curl_json "POST" "${INGEST_URL}/v1/visibility/events" "${TELEMETRY_FIXTURE}" "${POST_RESP}")" || fail_stage "ingest" "failed to post fixture telemetry" "curl -v -X POST --data-binary @${TELEMETRY_FIXTURE} ${INGEST_URL}/v1/visibility/events"
if [[ "${POST_CODE}" != "200" && "${POST_CODE}" != "202" ]]; then
  fail_stage "ingest" "unexpected status ${POST_CODE}" "cat ${POST_RESP}"
fi
ok_stage "fixture telemetry posted"

RESEARCH_RESP="${TMP_DIR}/research.json"
RESEARCH_CODE="$(curl_json "POST" "${DETECTION_URL}/v1/detection/research-items" "${RESEARCH_FIXTURE}" "${RESEARCH_RESP}")" || fail_stage "research" "request failed" "curl -v -X POST -H 'Content-Type: application/json' --data-binary @${RESEARCH_FIXTURE} ${DETECTION_URL}/v1/detection/research-items"
[[ "${RESEARCH_CODE}" == "201" ]] || fail_stage "research" "unexpected status ${RESEARCH_CODE}" "cat ${RESEARCH_RESP}"
RESEARCH_ID="$(json_get "${RESEARCH_RESP}" "d.get('id')")"
[[ -n "${RESEARCH_ID}" ]] || fail_stage "research" "missing research item id" "cat ${RESEARCH_RESP}"
ok_stage "research created id=${RESEARCH_ID}"

SMOKE_PACK_VERSION="${SMOKE_PACK_VERSION:-0.1.$(date +%s)}"
CANDIDATE_REQ="${TMP_DIR}/candidate-request.json"
python3 - "${CANDIDATE_FIXTURE}" "${RESEARCH_ID}" "${SMOKE_PACK_VERSION}" >"${CANDIDATE_REQ}" <<'PY'
import json
import sys
path, rid, version = sys.argv[1], sys.argv[2], sys.argv[3]
with open(path, "r", encoding="utf-8") as fh:
    doc = json.load(fh)
doc["research_item_id"] = rid
doc["pack_version"] = version
doc["title"] = f"{doc.get('title', 'lab candidate')} smoke {version}"
print(json.dumps(doc))
PY

CANDIDATE_RESP="${TMP_DIR}/candidate.json"
CANDIDATE_CODE="$(curl_json "POST" "${DETECTION_URL}/v1/detection/candidates" "${CANDIDATE_REQ}" "${CANDIDATE_RESP}")" || fail_stage "candidate" "request failed" "curl -v -X POST -H 'Content-Type: application/json' --data-binary @${CANDIDATE_REQ} ${DETECTION_URL}/v1/detection/candidates"
[[ "${CANDIDATE_CODE}" == "201" ]] || fail_stage "candidate" "unexpected status ${CANDIDATE_CODE}" "cat ${CANDIDATE_RESP}"
CANDIDATE_ID="$(json_get "${CANDIDATE_RESP}" "d.get('id')")"
PACK_ID="$(json_get "${CANDIDATE_RESP}" "d.get('pack_id')")"
PACK_VERSION="$(json_get "${CANDIDATE_RESP}" "d.get('pack_version')")"
[[ -n "${CANDIDATE_ID}" && -n "${PACK_ID}" && -n "${PACK_VERSION}" ]] || fail_stage "candidate" "candidate response missing id/pack_id/pack_version" "cat ${CANDIDATE_RESP}"
ok_stage "candidate created id=${CANDIDATE_ID} pack=${PACK_ID}@${PACK_VERSION}"

VALIDATE_REQ="${TMP_DIR}/validate-request.json"
python3 - "${LAB_DEVICE_ID}" "${LAB_TENANT_ID}" >"${VALIDATE_REQ}" <<'PY'
import json
import sys
device_id, tenant_id = sys.argv[1], sys.argv[2]
payload = {"device_id": device_id, "limit": 500}
if tenant_id:
    payload["tenant_id"] = tenant_id
print(json.dumps(payload))
PY

VALIDATE_RESP="${TMP_DIR}/validate.json"
VALIDATE_CODE="$(curl_json "POST" "${DETECTION_URL}/v1/detection/candidates/${CANDIDATE_ID}/validate" "${VALIDATE_REQ}" "${VALIDATE_RESP}")" || fail_stage "validate" "request failed" "curl -v -X POST -H 'Content-Type: application/json' --data-binary @${VALIDATE_REQ} ${DETECTION_URL}/v1/detection/candidates/${CANDIDATE_ID}/validate"
[[ "${VALIDATE_CODE}" == "200" ]] || fail_stage "validate" "unexpected status ${VALIDATE_CODE}" "cat ${VALIDATE_RESP}"
assert_json_true "${VALIDATE_RESP}" "d.get('success') is True" "validate" "cat ${VALIDATE_RESP}"
assert_json_true "${VALIDATE_RESP}" "int(d.get('events_fetched', 0)) >= 1" "validate" "curl -sS '${INGEST_URL}/v1/visibility/events?device_id=${LAB_DEVICE_ID}&limit=20'"
ok_stage "candidate validated"

APPROVE_RESP="${TMP_DIR}/approve.json"
APPROVE_CODE="$(curl_json "POST" "${DETECTION_URL}/v1/detection/candidates/${CANDIDATE_ID}/approve" "" "${APPROVE_RESP}")" || fail_stage "approve" "request failed" "curl -v -X POST ${DETECTION_URL}/v1/detection/candidates/${CANDIDATE_ID}/approve"
[[ "${APPROVE_CODE}" == "200" ]] || fail_stage "approve" "unexpected status ${APPROVE_CODE}" "cat ${APPROVE_RESP}"
ok_stage "candidate approved"

SIGN_RESP="${TMP_DIR}/sign.json"
SIGN_CODE="$(curl_json "POST" "${DETECTION_URL}/v1/detection/candidates/${CANDIDATE_ID}/sign" "" "${SIGN_RESP}")" || fail_stage "sign" "request failed" "curl -v -X POST ${DETECTION_URL}/v1/detection/candidates/${CANDIDATE_ID}/sign"
[[ "${SIGN_CODE}" == "200" ]] || fail_stage "sign" "unexpected status ${SIGN_CODE}" "cat ${SIGN_RESP}"
SIGNED_PACK_ID="$(json_get "${SIGN_RESP}" "d.get('signed_pack', {}).get('id')")"
[[ -n "${SIGNED_PACK_ID}" ]] || fail_stage "sign" "missing signed_pack.id" "cat ${SIGN_RESP}"
ok_stage "candidate signed artifact=${SIGNED_PACK_ID}"

SIGNER_RESP="${TMP_DIR}/signer-info.json"
SIGNER_CODE="$(curl_json "GET" "${DETECTION_URL}/v1/detection/signer-info" "" "${SIGNER_RESP}")" || fail_stage "signer-info" "request failed" "curl -v ${DETECTION_URL}/v1/detection/signer-info"
[[ "${SIGNER_CODE}" == "200" ]] || fail_stage "signer-info" "unexpected status ${SIGNER_CODE}" "cat ${SIGNER_RESP}"
PACK_PUBLIC_KEY="$(json_get "${SIGNER_RESP}" "d.get('public_key_b64')")"
[[ -n "${PACK_PUBLIC_KEY}" ]] || fail_stage "signer-info" "missing public_key_b64" "cat ${SIGNER_RESP}"
ok_stage "signer info available"

LATEST_LINUX_RESP="${TMP_DIR}/latest-linux.json"
LATEST_LINUX_CODE="$(curl_json "GET" "${DETECTION_URL}/v1/detection-packs/latest?os=linux&agent_version=${AGENT_VERSION}&pack_id=${PACK_ID}" "" "${LATEST_LINUX_RESP}")" || fail_stage "latest-linux" "request failed" "curl -v '${DETECTION_URL}/v1/detection-packs/latest?os=linux&agent_version=${AGENT_VERSION}&pack_id=${PACK_ID}'"
[[ "${LATEST_LINUX_CODE}" == "200" ]] || fail_stage "latest-linux" "unexpected status ${LATEST_LINUX_CODE}" "cat ${LATEST_LINUX_RESP}"
assert_json_true "${LATEST_LINUX_RESP}" "d.get('pack_id') == '${PACK_ID}' and d.get('pack_version') == '${PACK_VERSION}'" "latest-linux" "cat ${LATEST_LINUX_RESP}"
ok_stage "latest pack linux matches"

LATEST_WINDOWS_RESP="${TMP_DIR}/latest-windows.json"
LATEST_WINDOWS_CODE="$(curl_json "GET" "${DETECTION_URL}/v1/detection-packs/latest?os=windows&agent_version=${AGENT_VERSION}&pack_id=${PACK_ID}" "" "${LATEST_WINDOWS_RESP}")" || fail_stage "latest-windows" "request failed" "curl -v '${DETECTION_URL}/v1/detection-packs/latest?os=windows&agent_version=${AGENT_VERSION}&pack_id=${PACK_ID}'"
[[ "${LATEST_WINDOWS_CODE}" == "200" ]] || fail_stage "latest-windows" "unexpected status ${LATEST_WINDOWS_CODE}" "cat ${LATEST_WINDOWS_RESP}"
assert_json_true "${LATEST_WINDOWS_RESP}" "d.get('pack_id') == '${PACK_ID}' and d.get('pack_version') == '${PACK_VERSION}'" "latest-windows" "cat ${LATEST_WINDOWS_RESP}"
ok_stage "latest pack windows matches"

ARTIFACT_HEADERS="${TMP_DIR}/artifact-headers.txt"
ARTIFACT_BODY="${TMP_DIR}/artifact.json"
ARTIFACT_CODE="$(curl -sS -D "${ARTIFACT_HEADERS}" -o "${ARTIFACT_BODY}" -w '%{http_code}' "${DETECTION_URL}/v1/detection-packs/${PACK_ID}/artifact?os=linux&agent_version=${AGENT_VERSION}&version=${PACK_VERSION}")" || fail_stage "artifact" "request failed" "curl -v '${DETECTION_URL}/v1/detection-packs/${PACK_ID}/artifact?os=linux&agent_version=${AGENT_VERSION}&version=${PACK_VERSION}'"
[[ "${ARTIFACT_CODE}" == "200" ]] || fail_stage "artifact" "unexpected status ${ARTIFACT_CODE}" "cat ${ARTIFACT_BODY}"
if ! rg -qi '^X-Content-SHA256:\s' "${ARTIFACT_HEADERS}"; then
  fail_stage "artifact" "missing X-Content-SHA256 header" "cat ${ARTIFACT_HEADERS}"
fi
if ! rg -qi '^X-Signature-Key-Id:\s' "${ARTIFACT_HEADERS}"; then
  fail_stage "artifact" "missing X-Signature-Key-Id header" "cat ${ARTIFACT_HEADERS}"
fi
if ! rg -qi '^X-Signature-Algorithm:\s' "${ARTIFACT_HEADERS}"; then
  fail_stage "artifact" "missing X-Signature-Algorithm header" "cat ${ARTIFACT_HEADERS}"
fi
assert_json_true "${ARTIFACT_BODY}" "d.get('pack_id') == '${PACK_ID}' and d.get('pack_version') == '${PACK_VERSION}' and isinstance(d.get('rules'), list) and len(d.get('rules')) >= 1" "artifact" "cat ${ARTIFACT_BODY}"
ok_stage "artifact headers and body verified"

STATUS_REQ="${TMP_DIR}/pack-status.json"
python3 - "${PACK_ID}" "${PACK_VERSION}" "${AGENT_VERSION}" >"${STATUS_REQ}" <<'PY'
import json
import sys

pack_id, pack_version, agent_version = sys.argv[1:]
print(json.dumps({
    "reported_agent_version": agent_version,
    "rollout_state": "applied",
    "active_pack_id": pack_id,
    "active_pack_version": pack_version,
    "signature_status": "valid",
    "hash_status": "valid",
    "schema_status": "valid",
    "compatibility_status": "compatible",
    "emit_visibility": True,
}))
PY

for agent_uid in "${LINUX_AGENT_UID}" "${WINDOWS_AGENT_UID}"; do
  STATUS_AGENT_REQ="${TMP_DIR}/pack-status-${agent_uid}.json"
  python3 - "${STATUS_REQ}" "${agent_uid}" >"${STATUS_AGENT_REQ}" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    doc = json.load(fh)
doc["device_id"] = sys.argv[2]
print(json.dumps(doc))
PY
  STATUS_RESP="${TMP_DIR}/pack-status-${agent_uid}-resp.json"
  STATUS_CODE="$(curl_json "POST" "${DETECTION_URL}/v1/agents/${agent_uid}/detection-pack-status" "${STATUS_AGENT_REQ}" "${STATUS_RESP}")" || fail_stage "pack-status" "failed to post status for ${agent_uid}" "cat ${STATUS_AGENT_REQ}"
  [[ "${STATUS_CODE}" == "200" ]] || fail_stage "pack-status" "unexpected status ${STATUS_CODE} for ${agent_uid}" "cat ${STATUS_RESP}"
done
ok_stage "lab agent pack status posted"

deadline=$((SECONDS + MAX_WAIT_SECONDS))
rollout_ok=0
while (( SECONDS < deadline )); do
  ROLLOUT_RESP="${TMP_DIR}/rollout.json"
  ROLLOUT_CODE="$(curl_json "GET" "${DETECTION_URL}/v1/detection-packs/${PACK_ID}/rollout-status" "" "${ROLLOUT_RESP}")" || true
  if [[ "${ROLLOUT_CODE:-}" == "200" ]]; then
    if python3 - "${ROLLOUT_RESP}" "${LINUX_AGENT_UID}" "${WINDOWS_AGENT_UID}" "${PACK_ID}" "${PACK_VERSION}" <<'PY'
import json
import sys

path, linux_uid, windows_uid, pack_id, pack_version = sys.argv[1:]
with open(path, "r", encoding="utf-8") as fh:
    data = json.load(fh)
agents = data.get("agents") or []
required = {linux_uid, windows_uid}
seen = set()
for row in agents:
    uid = row.get("agent_uid")
    if uid not in required:
        continue
    if row.get("computed_stale"):
        continue
    if row.get("rollout_state") != "applied":
        continue
    if row.get("active_pack_id") != pack_id:
        continue
    if row.get("active_pack_version") != pack_version:
        continue
    seen.add(uid)
raise SystemExit(0 if seen == required else 1)
PY
    then
      rollout_ok=1
      break
    fi
  fi
  sleep "${POLL_INTERVAL_SECONDS}"
done

if [[ "${rollout_ok}" -ne 1 ]]; then
  fail_stage "rollout-status" "linux/windows agents did not both report non-stale applied status for ${PACK_ID}@${PACK_VERSION}" "AEGIS_DETECTION_PACKS_ENABLED=true AEGIS_CONTROLLER_URL=${DETECTION_URL} AEGIS_DETECTION_PACK_PUBLIC_KEY=${PACK_PUBLIC_KEY} ${ROOT_DIR}/agents/linux-agent/scripts/run-lab-once.sh && pwsh -File ${ROOT_DIR}/agents/windows-agent/scripts/run-lab-once.ps1 -DetectionPacksEnabled -ControllerUrl ${DETECTION_URL} -DetectionPackPublicKey ${PACK_PUBLIC_KEY} && curl -sS ${DETECTION_URL}/v1/detection-packs/${PACK_ID}/rollout-status"
fi
ok_stage "rollout status includes both agents as applied and non-stale"

printf '\nSmoke result\n'
printf -- '- research_id: %s\n' "${RESEARCH_ID}"
printf -- '- candidate_id: %s\n' "${CANDIDATE_ID}"
printf -- '- signed_pack_id: %s\n' "${SIGNED_PACK_ID}"
printf -- '- pack: %s@%s\n' "${PACK_ID}" "${PACK_VERSION}"
printf -- '- rollout_status: %s/v1/detection-packs/%s/rollout-status\n' "${DETECTION_URL}" "${PACK_ID}"
printf -- '- signer_public_key_b64: %s\n' "${PACK_PUBLIC_KEY}"
