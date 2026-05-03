# Aegis Linux Agent

The Aegis Linux Agent is the Phase 1 Linux visibility component for AegisFlux.
It is visibility-only and emits the same event contract used by the Windows
agent.

## Phase 1 Scope

- Process inventory and process lineage
- Process-to-flow attribution from Linux socket snapshots
- Resolver observations from `/etc/resolv.conf`
- Non-blocking AI-agent and automation detection evidence
- Outbound-only lab telemetry

No blocking, quarantine, firewall enforcement, or inbound listener is in Phase 1
scope.

Current Linux lab collection uses:

- `sysinfo` process snapshots for process inventory and lineage
- `ss -tunap` snapshots for TCP/UDP socket to PID attribution
- `/etc/resolv.conf` nameserver observations

## Build

```bash
cargo check
cargo build --release
```

## Run in Lab Mode

```bash
AEGIS_BACKEND_URL=http://127.0.0.1:9091 \
AEGIS_AGENT_ID=linux-dev-agent-01 \
AEGIS_DEVICE_ID=linux-dev-agent-01 \
AEGIS_COLLECT_COMMAND_LINE=true \
cargo run -- --once --stdout
```

By default, local events are written to:

```text
/var/lib/aegis/linux-agent/events.jsonl
```

The built-in lab transport supports plain `http://` only for localhost or a
trusted lab tunnel.

## Linux Lab systemd Timer

Build the release binary, then install a one-shot service plus timer:

```bash
cd /opt/aegis/aegisflux/agents/linux-agent
cargo build --release
sudo ./scripts/install-lab-systemd.sh
```

Defaults:

```text
AEGIS_BACKEND_URL=http://127.0.0.1:9091
AEGIS_AGENT_ID=linux-dev-agent-01
AEGIS_DEVICE_ID=linux-dev-agent-01
AEGIS_EVENT_SPOOL=/var/lib/aegis/linux-agent/events.jsonl
```

Logs are appended to:

```text
/var/log/aegis/linux-agent.log
```

Remove the timer and service with:

```bash
sudo ./scripts/uninstall-lab-systemd.sh
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `AEGIS_AGENT_ID` | `linux-agent-dev` | Stable agent identity for lab runs |
| `AEGIS_DEVICE_ID` | hostname fallback | Device identity reported in event envelopes |
| `AEGIS_SENSOR_VERSION` | crate version | Sensor version in event envelopes |
| `AEGIS_EVENT_SPOOL` | `/var/lib/aegis/linux-agent/events.jsonl` | JSONL event spool path |
| `AEGIS_BACKEND_URL` | empty | Optional Phase 1 ingest base URL for outbound lab telemetry |
| `AEGIS_COLLECT_COMMAND_LINE` | `false` | Opt-in command-line collection for lab scenarios; values are sanitized and truncated |
