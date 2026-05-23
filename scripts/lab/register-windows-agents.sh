#!/usr/bin/env bash
set -euo pipefail

ACTIONS_URL="${ACTIONS_URL:-http://localhost:8083}"
WINDOWS_AGENT_IDS="${WINDOWS_AGENT_IDS:-windows-dev-agent-02 windows-dev-agent-03 windows-dev-agent-04}"
WINDOWS_AGENT_IPS="${WINDOWS_AGENT_IPS:-}"

if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required to build heartbeat payloads" >&2
  exit 127
fi

index=0
for agent_id in $WINDOWS_AGENT_IDS; do
  ip="$(WINDOWS_AGENT_IPS="$WINDOWS_AGENT_IPS" python3 - "$index" <<'PY'
import os
import re
import sys

items = [x for x in re.split(r"[\s,]+", os.environ.get("WINDOWS_AGENT_IPS", "").strip()) if x]
idx = int(sys.argv[1])
print(items[idx] if idx < len(items) else "")
PY
)"

  python3 - "$agent_id" "$ip" <<'PY' | curl -fsS -X POST "$ACTIONS_URL/agents/heartbeat" -H "Content-Type: application/json" -d @-
import datetime
import json
import sys

agent_id = sys.argv[1]
primary_ip = sys.argv[2]
hostname = agent_id.upper().replace("-", "")

payload = {
    "agent_uid": agent_id,
    "org_id": "default-org",
    "host_id": agent_id,
    "hostname": hostname,
    "agent_version": "0.1.0",
    "last_seen": datetime.datetime.now(datetime.timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z"),
    "status": "online",
    "labels": ["visibility-lab", "windows"],
    "note": "Seeded from macOS lab bootstrap; replace with live Windows heartbeat after host provisioning",
    "capabilities": {
        "visibility": True,
        "dynamic_detection_packs": False,
        "platform": "windows",
    },
    "platform": {
        "hostname": hostname,
        "os": "windows",
        "architecture": "AMD64",
    },
    "network": {},
}

if primary_ip:
    payload["platform"]["primary_ip"] = primary_ip
    payload["network"]["primary_ip"] = primary_ip

json.dump(payload, sys.stdout, separators=(",", ":"))
PY
  echo
  index=$((index + 1))
done
