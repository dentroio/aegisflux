# Audit-mode lab smoke

This walkthrough exercises the audit-mode bundle foundation end to end against a lab
`actions-api`. It demonstrates that bundles can be staged, endpoint status can be reported,
matches can be recorded, and revocation works — all without any blocking behavior.

> The walkthrough is observe-only by construction. The platform rejects any mode other than
> `audit` and the agent contract has no allowed `block`/`deny` action.

## Prerequisites

- `actions-api` reachable at `${ACTIONS_API_URL:-http://localhost:8083}`.
- `console` running and proxying `/api/actions/...` to `actions-api`.
- (Optional) one or more lab agents that can be referenced by `device_id`. The smoke does not
  require a real agent — endpoint status and match calls can be issued from `curl` to mimic
  the agent.

## Steps

1. **Create a draft bundle.**

```sh
BASE=${ACTIONS_API_URL:-http://localhost:8083}
curl -sS -X POST "$BASE/platform/audit-bundles" \
  -H 'Content-Type: application/json' \
  -d '{
    "title": "Audit Ollama listen ports",
    "description": "Observe-only audit for ollama serve on 0.0.0.0",
    "scope": ["device:linux-lab-1"],
    "expected_match_telemetry": ["process.listen_port==11434"],
    "rollback_notes": "Revoke bundle id; agents return to pending."
  }' | tee /tmp/audit-bundle.json
BUNDLE_ID=$(jq -r .id /tmp/audit-bundle.json)
```

2. **Stage the bundle and seed an endpoint id.**

```sh
curl -sS -X POST "$BASE/platform/audit-bundles/$BUNDLE_ID/stage" \
  -H 'Content-Type: application/json' \
  -d '{ "device_ids": ["linux-lab-1"] }' | jq '.status, (.endpoint_statuses[] | {device_id, status})'
```

Expected output: `"staged"` and one endpoint row with `pending`.

3. **Simulate an agent accepting the bundle.**

```sh
curl -sS -X POST "$BASE/platform/audit-bundles/$BUNDLE_ID/status" \
  -H 'Content-Type: application/json' \
  -d '{ "device_id": "linux-lab-1", "status": "accepted", "agent_version": "1.4.0" }' \
  | jq '(.endpoint_statuses[] | {device_id, status, agent_version})'
```

Expected: status `accepted`, agent version recorded.

4. **Simulate the agent reporting a match.**

```sh
curl -sS -X POST "$BASE/platform/audit-bundles/$BUNDLE_ID/match" \
  -H 'Content-Type: application/json' \
  -d '{
    "device_id": "linux-lab-1",
    "process": "ollama",
    "indicator": "listen_port==11434",
    "detail": "ollama serve --host 0.0.0.0"
  }' | jq '.matches | length'
```

Expected: a non-zero match count and the bundle remains in `staged` status. The endpoint row
should now have a recent `last_match_at_ms`.

5. **Confirm operational events are emitted.**

```sh
curl -sS "$BASE/platform/operational-events?subject=$BUNDLE_ID" | jq '.events[] | {event_type, status, description}'
```

Expected events include `audit_bundle.created`, `audit_bundle.staged`, one
`audit_bundle.endpoint_status`, and at least one `audit_bundle.match`.

6. **Revoke the bundle and verify match is now rejected.**

```sh
curl -sS -X POST "$BASE/platform/audit-bundles/$BUNDLE_ID/revoke" \
  -H 'Content-Type: application/json' \
  -d '{ "note": "lab teardown" }' | jq '.status'

curl -sS -o /dev/null -w "%{http_code}\n" \
  -X POST "$BASE/platform/audit-bundles/$BUNDLE_ID/match" \
  -H 'Content-Type: application/json' \
  -d '{ "device_id": "linux-lab-1" }'
```

Expected: status `"revoked"` followed by HTTP `409` for the match call.

## What this proves

- Audit-mode bundles can be staged without blocking behavior.
- Endpoint status (`accepted`, `rejected`, `incompatible`, `stale`) is reported and recorded.
- Match telemetry is the only side effect; revocation prevents new matches.
- Operational events provide an auditable trail for staging and status changes.
- The UI (`/control/audit-bundles`) renders all of the above and labels the workflow as
  audit-only at every step.
