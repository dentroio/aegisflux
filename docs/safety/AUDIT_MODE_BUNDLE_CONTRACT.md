# Audit-mode bundle: wire contract

This document defines the API and agent receipt contract for audit-mode bundles. It complements
[`AUDIT_MODE.md`](AUDIT_MODE.md), which explains why the contract is observe-only.

All routes are served by `actions-api`. The Next.js console proxies them under
`/api/actions/platform/audit-bundles…`.

## Lifecycle

```
draft -> staged -> [revoked | expired]
                \
                 -> endpoint statuses: pending | accepted | rejected | incompatible | stale
```

## Bundle shape (server)

```jsonc
{
  "id": "uuid",
  "version": "v1",
  "mode": "audit",
  "title": "Audit Ollama listen ports",
  "description": "Observe-only audit for ollama serve on 0.0.0.0",
  "scope": ["device:linux-lab-1"],
  "expected_match_telemetry": ["process.listen_port==11434"],
  "approval_refs": ["ticket:OPS-1234"],
  "rollback_notes": "Revoke bundle id; agents return to pending.",
  "source_candidate_id": "uuid?",
  "source_draft_id": "uuid?",
  "status": "draft|staged|expired|revoked",
  "staged_at_ms": 0,
  "expires_at_ms": 0,
  "endpoint_statuses": [
    { "device_id": "linux-lab-1", "status": "accepted", "agent_version": "1.4.0", "reported_at_ms": 0, "last_match_at_ms": 0 }
  ],
  "matches": [
    { "id": "uuid", "device_id": "linux-lab-1", "process": "ollama", "indicator": "listen_port==11434", "detail": "ollama serve --host 0.0.0.0", "at_ms": 0 }
  ],
  "history": [
    { "id": "uuid", "action": "audit_bundle.staged", "from_status": "draft", "to_status": "staged", "at_ms": 0 }
  ]
}
```

`mode` MUST be `audit`. The platform rejects any other mode with `400`.

## Routes

| Method | Path                                                  | Description                                         |
| ------ | ----------------------------------------------------- | --------------------------------------------------- |
| GET    | `/platform/audit-bundles`                             | List bundles with `status_counts`.                  |
| POST   | `/platform/audit-bundles`                             | Create a draft bundle.                              |
| GET    | `/platform/audit-bundles/{id}`                        | Fetch one bundle.                                   |
| PATCH  | `/platform/audit-bundles/{id}`                        | Edit a draft bundle (only while `status == draft`). |
| POST   | `/platform/audit-bundles/{id}/stage`                  | Move from draft → staged; optionally seed device ids.|
| POST   | `/platform/audit-bundles/{id}/status`                 | Endpoint reports its status for the bundle.         |
| POST   | `/platform/audit-bundles/{id}/match`                  | Endpoint reports an observe-only match.             |
| POST   | `/platform/audit-bundles/{id}/revoke`                 | Operator revokes a bundle.                          |

## Endpoint receipt contract

When the agent fetches a staged bundle (or receives a push), it MUST report status using:

```http
POST /platform/audit-bundles/{id}/status
Content-Type: application/json

{
  "device_id": "linux-lab-1",
  "status": "accepted | rejected | incompatible | stale",
  "reason": "optional human-readable reason",
  "agent_version": "1.4.0"
}
```

Recommended status mapping:

- `accepted` — bundle parsed; scope evaluator is active; agent will emit matches.
- `rejected` — bundle parsed but operator policy refuses it (e.g., approval missing).
- `incompatible` — bundle uses scope or telemetry that this agent cannot evaluate.
- `stale` — agent has the bundle but considers it expired or replaced.

When scope matches, the agent MUST emit a match record:

```http
POST /platform/audit-bundles/{id}/match
Content-Type: application/json

{
  "device_id": "linux-lab-1",
  "process": "ollama",
  "indicator": "listen_port==11434",
  "detail": "ollama serve --host 0.0.0.0"
}
```

The match record is the **only** side effect. There is no allowed `block` or `deny` action in
this contract.

## Operational events

Each lifecycle change emits an entry on `/platform/operational-events`:

- `audit_bundle.created`
- `audit_bundle.updated`
- `audit_bundle.staged`
- `audit_bundle.endpoint_status` (with `device_id`)
- `audit_bundle.match` (with `device_id`)
- `audit_bundle.revoked`

These entries are also recorded in the bundle's `history` array.

## Lab smoke

See [`docs/demos/AUDIT_MODE_LAB_SMOKE.md`](../demos/AUDIT_MODE_LAB_SMOKE.md) for a repeatable
walkthrough that creates a bundle, stages it, simulates an endpoint accept/reject, and records
a match.
