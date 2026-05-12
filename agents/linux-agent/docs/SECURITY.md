# Linux Agent Security Baseline

## Design Rules

- The agent must run with the least privilege required for active collectors.
- Linux x86_64 is the primary Phase 1 lab deployment target.
- Code must not assume a specific distribution, init system, package manager, or filesystem layout outside the documented lab install path.
- The agent must not expose an inbound network listener by default.
- Telemetry must use an outbound-only control/telemetry channel.
- Event schemas must be validated before backend ingestion.
- Local policy and future commands must be signed by the backend before the agent trusts them.
- Agent updates must be signed and versioned.
- Secrets must not be written to logs or event payloads.
- Command-line collection is disabled by default because command lines can contain tokens, secrets, file paths, and environment-like material.
- When command-line collection is enabled for lab scenarios, values must be sanitized and truncated before emission.
- The collector layer must be separate from event normalization and transport.
- Enforcement work is excluded from Phase 1.

## Rust Safety Rules

- `unsafe_code` is forbidden in the initial crate.
- New dependencies require review for maintenance health, license, transitive dependency count, platform API surface, and known advisories.
- Panics are not acceptable as normal error handling.
- Parsing and event construction should be fuzzable once schemas stabilize.

## Phase 1 Runtime Posture

- Runs in visibility-only mode.
- Emits process, flow, DNS, and detection evidence.
- Does not block traffic.
- Does not modify firewall, routing, nftables, iptables, eBPF programs, or SELinux/AppArmor policy.
- Does not change endpoint posture or SGT state.

## Self-Protection Baseline

- Persistent Linux deployments should use `scripts/install-systemd.sh`, not the
  lab timer.
- The service runs under the dedicated `aegis` account with a locked-down
  filesystem view and no ambient Linux capabilities.
- systemd restarts the agent after crashes or process termination and monitors
  liveness with watchdog notifications from the agent process.
- Service hardening reduces the damage an exploited agent process can do:
  `NoNewPrivileges`, `ProtectSystem=strict`, `ProtectHome`,
  `ProtectKernelTunables`, `ProtectKernelModules`, `ProtectControlGroups`,
  `RestrictSUIDSGID`, `MemoryDenyWriteExecute`, and native syscall architecture
  filtering are enabled.
- This is resilience and tamper evidence, not an absolute guarantee. A local
  root attacker can still stop systemd units unless the host also deploys
  platform controls such as secure boot, signed updates, audit rules, LSM policy,
  EDR controls, and centralized alerting on service stop or binary modification.

## Future Hardening

- Code signing for release builds
- mTLS or signed telemetry envelope
- Dedicated least-privilege service account
- Tamper detection for service stop, binary modification, policy disablement, and log clearing
- Signature verification for local policy cache
- SBOM and dependency audit in CI
- Reproducible release artifact process where practical
