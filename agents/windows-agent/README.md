# Aegis Windows Agent

The Aegis Windows Agent is the planned Windows visibility and local-control component for AegisFlux. It is built in Rust and starts with visibility-only collection for process, network, DNS, and early AI-agent/automation evidence.

## Phase 1 Scope

- Process inventory and process lineage
- Process-to-flow attribution
- DNS/domain observations
- Non-blocking AI-agent and automation detection evidence
- Signed/outbound telemetry path preparation

No blocking, quarantine, WFP enforcement, SGT changes, or inbound listener is in Phase 1 scope.

## Event Contract

The Windows agent emits Phase 1 visibility events that must conform to:

```text
../../schemas/visibility/
```

Start with [../../schemas/visibility/agent-heartbeat.schema.json](../../schemas/visibility/agent-heartbeat.schema.json), [../../schemas/visibility/process-started.schema.json](../../schemas/visibility/process-started.schema.json), [../../schemas/visibility/flow-started.schema.json](../../schemas/visibility/flow-started.schema.json), and [../../schemas/visibility/dns-observed.schema.json](../../schemas/visibility/dns-observed.schema.json).

## Security Baseline

- Rust implementation with `unsafe_code = "forbid"`
- No third-party dependencies in the initial skeleton
- No inbound listener
- Outbound-only event emission
- Local JSONL spool for early lab validation
- Explicit schemas before backend ingestion
- Collectors separated from event normalization

See [docs/SECURITY.md](docs/SECURITY.md).

## Build

```bash
cargo check
cargo build
```

## Target Architectures

x86_64 Windows is the primary deployment target for Phase 1. ARM64 is supported as a development and future deployment consideration, so code should still avoid unnecessary x86_64 assumptions.

Planned Windows release targets:

```text
x86_64-pc-windows-msvc
aarch64-pc-windows-msvc
```

On a development machine, check the active Rust host target with:

```bash
rustc -vV
```

## Run in Lab Mode

```bash
cargo run -- --once --stdout
```

By default, local events are written to:

```text
C:\ProgramData\Aegis\Agent\spool\events.jsonl
```

On non-Windows development machines the default spool path is:

```text
/tmp/aegis-windows-agent/events.jsonl
```

To send the same event batch directly to the Phase 1 ingest API during lab runs:

```bash
AEGIS_BACKEND_URL=http://127.0.0.1:9090 cargo run -- --once
```

The agent appends `/v1/visibility/events` to `AEGIS_BACKEND_URL` unless the URL
already ends with that path. The built-in lab transport supports plain `http://`
only; use it with localhost or a trusted lab tunnel.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `AEGIS_AGENT_ID` | `windows-agent-dev` | Stable agent identity for lab runs |
| `AEGIS_DEVICE_ID` | hostname fallback | Device identity reported in event envelopes |
| `AEGIS_SENSOR_VERSION` | crate version | Sensor version in event envelopes |
| `AEGIS_EVENT_SPOOL` | platform default | JSONL event spool path |
| `AEGIS_BACKEND_URL` | empty | Optional Phase 1 ingest base URL for outbound lab telemetry |
| `AEGIS_COLLECT_COMMAND_LINE` | `false` | Opt-in command-line collection for lab scenarios; values are sanitized and truncated |
