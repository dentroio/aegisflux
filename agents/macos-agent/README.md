# Aegis macOS Agent

The Aegis macOS Agent is the planned macOS visibility component for AegisFlux. It is built in Rust and starts with visibility-only collection for process, network, DNS, and early automation/AI-agent evidence.

## Phase 1 Scope

- Agent heartbeat and local event spool
- Process visibility design for Endpoint Security Framework integration
- Network and DNS visibility design
- Non-blocking automation and AI-agent evidence

No blocking, Network Extension enforcement, PF rules, quarantine, or MDM action is in Phase 1 scope.

## Event Contract

The macOS agent emits Phase 1 visibility events that must conform to:

```text
../../schemas/visibility/
```

The scaffold currently emits [../../schemas/visibility/agent-heartbeat.schema.json](../../schemas/visibility/agent-heartbeat.schema.json) and collector status events. Process, network, and DNS events should use the same shared visibility contracts as Windows once implemented.

## Security Baseline

- Rust implementation with `unsafe_code = "forbid"`
- No third-party dependencies in the initial skeleton
- No inbound listener
- Outbound-only event emission
- Local JSONL spool for early lab validation
- Explicit schemas before backend ingestion
- Collector code separated from event normalization

See [docs/SECURITY.md](docs/SECURITY.md).

## macOS Collection Strategy

Preferred future collection layers:

- Endpoint Security Framework for process and file-system events
- Network Extension or system APIs for network visibility where approved
- DNS observation through approved system mechanisms
- Unified log collection only for lab diagnostics, not as the primary production sensor

Endpoint Security requires Apple-granted entitlements for production distribution. The first agent scaffold does not request those entitlements and does not install a system extension.

## Build

```bash
cargo check
cargo build
```

## Run in Lab Mode

```bash
cargo run -- --once --stdout
```

By default, local events are written to:

```text
~/Library/Application Support/Aegis/Agent/spool/events.jsonl
```

Production service packaging may use a system-owned path after code signing, launchd packaging, and permissions are defined. On non-macOS development machines the default spool path is:

```text
/tmp/aegis-macos-agent/events.jsonl
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `AEGIS_AGENT_ID` | `macos-agent-dev` | Stable agent identity for lab runs |
| `AEGIS_DEVICE_ID` | hostname fallback | Device identity reported in event envelopes |
| `AEGIS_SENSOR_VERSION` | crate version | Sensor version in event envelopes |
| `AEGIS_EVENT_SPOOL` | platform default | JSONL event spool path |
| `AEGIS_BACKEND_URL` | empty | Reserved for future outbound telemetry |
