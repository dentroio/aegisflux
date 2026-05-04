# Aegis Agent Performance Architecture

## Goal

Aegis endpoint agents must feel invisible. The agent should collect high-value evidence without creating noticeable CPU, memory, disk, network, or battery impact. If the agent competes with user workloads, customers will not deploy it broadly.

## Operating Principles

- Prefer event-driven operating system sources over tight polling loops.
- Keep snapshot collectors bounded by time, count, and output size.
- Run expensive collectors less often than cheap collectors.
- Deduplicate observations before transport.
- Spool locally with size limits and backpressure.
- Fail quiet and degrade gracefully when a collector is unavailable.
- Make every collector measurable: duration, events emitted, bytes emitted, and errors.

## Initial Resource Budgets

These are early engineering targets, not marketing guarantees:

| Resource | Idle Target | Collection Burst Target |
|----------|-------------|-------------------------|
| CPU | near 0 percent outside scheduled work | short bursts only, generally below 2 percent average over 5 minutes |
| Memory | small resident footprint, target under 50 MB for Phase 1 agents | avoid unbounded buffers and large in-memory histories |
| Disk | bounded JSONL spool and temporary files | delete browser/history snapshots immediately after parsing |
| Network | compressed/batched outbound telemetry later | no inbound listener; back off on transport failure |
| Battery | no high-frequency polling on laptops | adaptive scheduling based on power state later |

## Collector Scheduling

Collectors should not all run at the same cadence.

| Collector Class | Example | Cadence Direction |
|-----------------|---------|-------------------|
| Fast snapshot | process list, active flows | frequent in lab, adaptive in production |
| Medium snapshot | DNS cache, browser recent history | less frequent and capped |
| Inventory | browser extensions, SSE/SASE components, installed products | infrequent, change-triggered where possible |
| Heavy enrichment | file hash, signer, certificate inventory | background, sampled, cached, and budgeted |

## Windows Direction

The lab Windows agent currently uses snapshot commands where they are useful for proving the event contract. Production Windows collection should move toward:

- ETW for process, DNS, network, and browser-related signals where available.
- WFP/IP Helper for stronger flow attribution.
- Registry change notifications for installed products, proxy settings, and policy changes.
- Service Control Manager notifications or low-rate service inventory.
- Cached signer/hash enrichment with strict rate limits.

## Linux Direction

Linux agents should follow the same event contract with Linux-native sources:

- eBPF or procfs snapshots for process and flow visibility depending on privilege.
- Netlink/conntrack where available for network state.
- systemd/dbus inventory for services.
- Browser profile and extension inventory with the same low-rate rules.
- Package manager inventory for installed AI/SSE/SASE tooling.

## Required Agent Telemetry

Each agent should eventually emit health metrics for:

- Collector duration.
- Collector status.
- Events emitted per collector.
- Spool queue depth and spool bytes.
- Transport success/failure counts.
- Dropped or skipped events due to budget limits.
- Current resource usage when inexpensive to collect.

Performance is part of the product contract. Aegis should make resource cost visible in the management UI before customers have to ask.
