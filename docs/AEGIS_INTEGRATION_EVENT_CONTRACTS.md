# AegisFlux → Clarion integration events (WO-INT-001 baseline)

Minimal JSON envelopes for webhook/future-bus publication. Consumers map `device_id` to Clarion endpoint keys and attach evidence handles without sharing raw secrets.

## aegis.device.observed

Payload fields: `device_id`, `agent_id`, `source`, `freshness_ms`, `sensor_version`.

## aegis.ai_activity.summarized

Payload fields: `device_id`, `signal_counts` (DNS/process/finding tallies), `redacted_hints` booleans aligned with Actions API privacy toggles.

## aegis.inventory.item_observed

Payload fields: `device_id`, `item_type` (`browser_extension` \| `sase_component` \| …), `name`, `vendor`, `confidence`.

## aegis.finding.created

Payload fields: `finding_id`, `device_id`, `severity`, `title`, `relative_link` inside Aegis console.

## Clarion mapping notes

- Endpoint identity: join on `device_id` / external agent hash.
- User/session: optional when present in flow/DNS correlation metadata.
- Flow/destination: prefer normalized five-tuple summaries; never ship raw credentials.
- Inventory rows link to enterprise control catalog entries in Clarion.

See `GET /api/actions/platform/integration/devices/{device_id}` for the evidence summary contract returned by AegisFlux without Clarion online.
