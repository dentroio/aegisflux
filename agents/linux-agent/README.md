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

The crate pins a **Rust 1.82+** toolchain via `rust-toolchain.toml` (required by current `sysinfo` and lockfile resolution). Use a recent stable Rust if you override the toolchain.

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
| `AEGIS_CONTROLLER_URL` | empty | Lab detection-pipeline base URL (`http://host:port`) for WO-DET-003 pack APIs |
| `AEGIS_DETECTION_PACKS_ENABLED` | `false` | When `true`, fetch/verify/cache/evaluate signed `detection_pack.v1` documents (observe-only) |
| `AEGIS_DETECTION_PACK_PUBLIC_KEY` | empty | Standard Base64 of the raw 32-byte Ed25519 **verifying** key (required when packs are enabled) |
| `AEGIS_DETECTION_PACK_CACHE` | empty | Override directory for verified pack cache (default: `detection-pack/` next to the event spool parent) |

See [docs/DYNAMIC_PACKS.md](docs/DYNAMIC_PACKS.md) for the controller contract, cache layout, and status fields.
