# Aegis and Clarion Integration Contract

**Status:** Draft  
**Owner:** Aegis / Clarion architecture  
**Last updated:** April 26, 2026

## Purpose

Aegis and Clarion should remain independent products with a deliberate integration boundary. Aegis provides host and workload visibility, local telemetry, and future endpoint/workload enforcement. Clarion provides enterprise context, policy intelligence, risk decisions, and orchestration across network, endpoint, gateway, identity, and security systems.

The integration model is event-driven and API-driven. Aegis must not become a Clarion subdirectory or share Clarion internals. Clarion must not depend on Aegis internal storage or agent implementation details.

## Product Boundary

### Aegis Owns

- Host and workload agent lifecycle.
- Agent identity, registration, health, version, and capabilities.
- Process inventory and process lineage.
- Process-to-flow attribution from the host perspective.
- DNS and destination observations from the host perspective.
- Local application, automation, and AI-agent behavior signals.
- Signed host telemetry and local evidence capture.
- Local policy cache and future host/workload enforcement.
- Enforcement audit events, rollback evidence, and local control status.

### Clarion Owns

- Enterprise policy intent and policy authoring.
- Context graph across users, devices, sessions, SGTs, destinations, applications, identities, and risk.
- Network and infrastructure telemetry from ISE, pxGrid, NetFlow/IPFIX, Zeek, firewalls, DNS, DHCP, AD, MDM, EDR, proxies, and gateways.
- Risk scoring and policy decisions across network, endpoint, gateway, and application layers.
- Orchestration into ISE, TrustSec/SGT, SGACL, firewalls, proxies, model gateways, API gateways, SIEM/SOAR, MDM, and EDR.
- Operator workflows for investigation, policy, exception handling, and audit.

## Integration Principles

- Aegis emits evidence; Clarion decides enterprise action.
- Clarion sends decisions; Aegis applies only the endpoint/workload actions it is responsible for.
- Integration uses explicit versioned events and APIs.
- No shared implicit database writes.
- No direct coupling to internal service tables, local queues, or implementation-specific IDs.
- Both products can run independently in observe-only mode.
- Enforcement must be explainable, reversible, audited, and scoped to the process/workload where possible.

## Phase 1: Observe-Only Contract

Phase 1 keeps Aegis non-blocking. The goal is to make Clarion smarter with endpoint and workload evidence before any local enforcement is enabled.

### Aegis to Clarion Events

Aegis should publish these event families:

- `aegis.agent.registered`
- `aegis.agent.heartbeat`
- `aegis.process.started`
- `aegis.process.exited`
- `aegis.flow.started`
- `aegis.flow.ended`
- `aegis.dns.observed`
- `aegis.application.classified`
- `aegis.agent.detected`
- `aegis.risk_finding.created`

### Required Event Envelope

Every Aegis event sent to Clarion should include:

```json
{
  "schema_version": "visibility.v1",
  "event_id": "string",
  "event_type": "aegis.process.started",
  "timestamp_ms": 1777075005616,
  "source": "aegis",
  "tenant_id": "string",
  "device_id": "string",
  "agent_id": "string",
  "sensor_version": "string",
  "sequence": 1,
  "payload": {}
}
```

### Clarion Context Objects Created from Aegis

Clarion should map Aegis evidence into these context objects:

- Device
- Agent
- User
- Session
- Process
- Process lineage edge
- Application
- Flow
- Destination
- DNS observation
- AI-agent or automation finding
- Evidence bundle

## Identity Mapping

Aegis must preserve both host identity and agent identity. Clarion should not assume they are the same object.

| Concept | Aegis Field | Clarion Mapping | Notes |
|---|---|---|---|
| Tenant | `tenant_id` or `org_id` | Tenant/customer | Required for multi-tenant deployments. |
| Device | `device_id`, `host_id`, machine hash | Device node | Stable across agent reinstall when possible. |
| Agent | `agent_id` / `agent_uid` | Agent instance node | Stable for the registered keypair. |
| User | `user`, SID/UID where available | Identity/session node | Clarion correlates with AD/IdP/ISE. |
| Process | `process_guid`, pid, start time | Process node | PID alone is never stable enough. |
| Flow | `flow_id`, five-tuple, process guid | Flow/session edge | Clarion can enrich with network telemetry. |
| Destination | hostname, IP, port, SNI, DNS answer | Destination/app node | Clarion owns enterprise destination context. |
| Finding | `finding_id` | Risk/finding node | Clarion scores and decides action. |

## Phase 2: Decision Handoff

After observe-only validation, Clarion can send endpoint/workload decisions to Aegis.

Clarion to Aegis decision types:

- `observe_only`
- `increase_monitoring`
- `request_evidence_bundle`
- `redirect_process_destination`
- `block_process_destination`
- `apply_local_policy`
- `rollback_local_policy`

Decision payloads must include:

- Decision ID
- Tenant ID
- Target scope: device, agent, process, workload, destination, or policy group
- Reason and source policy
- Mode: observe, warn, enforce, rollback
- TTL
- Expected audit events
- Fail-open or fail-closed behavior

## Enforcement Boundary

Aegis may enforce only host/workload-local controls assigned to it. Clarion remains the system of record for why the action was selected.

Aegis enforcement examples:

- Block one process from one destination.
- Redirect model API calls from one process to an approved model gateway.
- Apply workload/container network policy.
- Cache a local policy while offline.
- Roll back or expire local policy by TTL.

Clarion orchestration examples:

- Change SGT or TrustSec posture.
- Trigger firewall, proxy, gateway, SIEM, SOAR, MDM, or EDR workflows.
- Escalate an investigation.
- Decide when endpoint enforcement should be combined with network or identity action.

## Non-Goals

- Do not merge Clarion and Aegis repositories.
- Do not make Aegis depend on Clarion database tables.
- Do not make Clarion depend on Aegis local storage internals.
- Do not place Clarion product documents permanently inside Aegis without an explicit docs decision.
- Do not build blocking enforcement before observe-only telemetry is validated.

## Near-Term Implementation Plan

1. Keep Aegis and Clarion as separate repos/products.
2. Keep Aegis visibility event schemas in Aegis.
3. Use the Aegis ingest lab export endpoint, `GET /v1/clarion/events`, to validate the first Clarion-facing event contract.
4. Add a Clarion-side importer later that consumes Aegis events into Clarion context objects.
5. Build macOS and Windows agent telemetry in observe-only mode first.
6. Validate investigation paths before adding enforcement decisions.

## Open Decisions

- Production transport for Aegis-to-Clarion events after the lab HTTP pull contract: HTTP push, NATS, Kafka, or file export.
- Whether Clarion should store raw Aegis events, normalized context objects, or both.
- Canonical tenant/device identity source when Aegis and Clarion see the same host through different systems.
- Minimum evidence required before Clarion may send an endpoint enforcement decision.
- Where the current Clarion architecture PDF should live long term.
