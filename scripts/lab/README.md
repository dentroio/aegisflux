# Aegis Lab Scripts

## End-to-end pipeline smoke (WO-OPS-001)

Prove the lab chain (agents → ingest → summaries → detection status → platform APIs → audit lifecycle) in one script:

```bash
chmod +x scripts/lab/smoke-e2e-pipeline.sh
./scripts/lab/smoke-e2e-pipeline.sh
```

Optional: `RUN_DETECTION_ROLLOUT=1` appends `smoke-detection-rollout.sh`. See `docs/ops/E2E_PIPELINE_SMOKE.md` for scenarios, manual console checklist, and failure modes.

## Operational health sweep (WO-OPS-003)

After `docker compose up -d`:

```bash
chmod +x scripts/lab/health-sweep.sh
./scripts/lab/health-sweep.sh
```

## Ingest replay (WO-OPS-004)

```bash
chmod +x scripts/lab/replay-visibility-fixtures.sh
./scripts/lab/replay-visibility-fixtures.sh
```

## Summary latency sample (WO-OPS-007)

```bash
chmod +x scripts/lab/load-ingest-summaries.sh
./scripts/lab/load-ingest-summaries.sh
```

See also `docs/ops/README.md` for the full operator doc set.

## Detection Rollout Smoke (WO-DET-006)

Use this smoke to verify the full WO-DET-002 + WO-DET-003 dynamic detection path in one run:
fixture ingest -> research -> candidate -> validate -> approve -> sign -> latest/artifact -> rollout status.

### Prerequisites

- Local compose stack is up and healthy.
- Lab tunnels are active (see sections below).
- Linux and Windows lab agents are installed and can post status to Actions API.

Quick pre-check:

```bash
./scripts/lab/check-agents.sh
```

### Run

```bash
./scripts/lab/smoke-detection-rollout.sh
```

### Default endpoints

| Variable | Default |
|----------|---------|
| `DETECTION_URL` | `http://127.0.0.1:8089` |
| `INGEST_URL` | `http://127.0.0.1:9091` |
| `ACTIONS_URL` | `http://127.0.0.1:8083` |
| `LINUX_AGENT_UID` | `linux-dev-agent-01` |
| `WINDOWS_AGENT_UID` | `windows-dev-agent-01` |
| `AGENT_VERSION` | `0.1.0` |

Example with overrides:

```bash
DETECTION_URL=http://127.0.0.1:8089 \
INGEST_URL=http://127.0.0.1:9091 \
./scripts/lab/smoke-detection-rollout.sh
```

### If rollout-status fails

The smoke prints a `next:` command that reruns both one-shot agents with the
current signer key and controller URL. Use that command directly, then rerun
the smoke.

## Direct Mac Lab Address

The current lab agents post directly to the developer Mac at:

```text
192.168.1.180
```

Default lab agent endpoints are:

- Ingest: `http://192.168.1.180:9091`
- Actions API: `http://192.168.1.180:8083`
- Detection pipeline: `http://192.168.1.180:8089`

If the Mac receives a new LAN address, update the lab agent script defaults and
reinstall/rebuild the Linux and Windows lab agents before running health checks.

## Windows Reverse Tunnel on macOS

The reverse tunnel remains available as a fallback for networks where direct
Mac access is blocked.

The Windows lab scheduled task posts to `http://127.0.0.1:9091` on the Windows
host. In the current lab topology, that address is provided by a reverse SSH
tunnel from the Mac to the Windows host:

```bash
ssh -i ~/.ssh/aegis_windows_lab -N -R 9091:127.0.0.1:9091 aegis@192.168.12.101
```

Install the same tunnel as a macOS user `launchd` job:

```bash
./scripts/lab/install-macos-windows-tunnel-launchd.sh
```

The launch agent label is:

```text
net.aegis.windows-reverse-tunnel
```

Inspect it with:

```bash
launchctl print gui/$(id -u)/net.aegis.windows-reverse-tunnel
```

Logs are written to:

```text
~/Library/Logs/Aegis/windows-reverse-tunnel.out.log
~/Library/Logs/Aegis/windows-reverse-tunnel.err.log
```

Remove it with:

```bash
./scripts/lab/uninstall-macos-windows-tunnel-launchd.sh
```

Optional environment overrides:

| Variable | Default |
|----------|---------|
| `AEGIS_WINDOWS_HOST` | `192.168.12.101` |
| `AEGIS_WINDOWS_USER` | `aegis` |
| `AEGIS_WINDOWS_SSH_KEY` | `$HOME/.ssh/aegis_windows_lab` |
| `AEGIS_TUNNEL_REMOTE_PORT` | `9091` |
| `AEGIS_TUNNEL_LOCAL_HOST` | `127.0.0.1` |
| `AEGIS_TUNNEL_LOCAL_PORT` | `9091` |

## Expected Tunnel Port Map

Both lab tunnels expose the same service map on each remote lab machine:

| Remote Port (lab host) | Local Port (developer Mac) | Service |
|------------------------|----------------------------|---------|
| `9091` | `9091` | Ingest (`/v1/visibility/events`, `/healthz`) |
| `8083` | `8083` | Actions API (`/agents`, `/agents/heartbeat`) |
| `8089` | `8089` | Detection pipeline (`/healthz`, pack endpoints) |

When reverse tunnels are enabled, each remote host uses local loopback URLs:

- Ingest: `http://127.0.0.1:9091`
- Actions API: `http://127.0.0.1:8083`
- Detection pipeline: `http://127.0.0.1:8089`

## Heartbeat Freshness Thresholds

The Actions API classifies each agent's heartbeat into one of three states based
on the age of the last `POST /agents/heartbeat` call:

| Status  | Last-seen age         | Meaning |
|---------|-----------------------|---------|
| online  | < 3 minutes           | Heartbeat is current; agent is alive. |
| stale   | 3 – 15 minutes        | Agent missed several expected cycles. Check tunnel and process. |
| offline | > 15 minutes          | Agent is disconnected. Do not trust new evidence from it. |

The default lab collection interval is 60 seconds (`AEGIS_COLLECTION_INTERVAL_SECONDS`).
Three missed cycles (3 minutes) triggers the `stale` transition. Fifteen minutes
without contact (≈ 15 missed cycles) triggers `offline`.

The `check-agents.sh` script reports each expected agent as:

- `[ok]` — API status is `online` (heartbeat < 3 min old).
- `[warn]` — API status is `stale` (heartbeat 3–15 min old).
- `[fail]` — API status is `offline` or agent is missing.

Detection-pack status is tracked independently and may show a valid last-applied
pack even when the agent heartbeat is stale. Evaluate them separately.

## Failure Mode Diagnosis

Use the following checks to distinguish the most common lab failure patterns.
Run `./scripts/lab/check-agents.sh` first — each section below maps to a
possible failure output from that script.

### Agent process stopped

**Symptom:** API returns `stale` or `offline`; remote tunnel listeners are
present on the lab host.

**Check:**
```bash
# Linux lab
ssh -i ~/.ssh/id_ed25519 clarion@<LINUX_HOST> 'systemctl status aegis-linux-agent-lab.timer'
# or for the persistent service:
ssh -i ~/.ssh/id_ed25519 clarion@<LINUX_HOST> 'systemctl status aegis-linux-agent.service'

# Windows lab (from PowerShell on the Windows host)
schtasks /Query /TN "AegisLabWindowsAgent" /FO LIST
```

**Fix:**
```bash
# Linux timer restart
ssh -i ~/.ssh/id_ed25519 clarion@<LINUX_HOST> \
  'sudo systemctl restart aegis-linux-agent-lab.timer && sudo systemctl start aegis-linux-agent-lab.service'

# Windows one-shot
# (run on Windows host)
powershell -NoProfile -ExecutionPolicy Bypass -File C:\AegisLab\aegisflux\agents\windows-agent\scripts\run-lab-once.ps1
```

### Tunnel down

**Symptom:** Remote tunnel-forward check fails (`[fail] linux-lab tunnel-forward listeners
are incomplete`). API may show `stale` or `offline`.

**Check:**
```bash
# Inspect launchd tunnel state on the developer Mac
launchctl print gui/$(id -u)/net.aegis.linux-reverse-tunnel
launchctl print gui/$(id -u)/net.aegis.windows-reverse-tunnel

# Check tunnel logs
tail -40 ~/Library/Logs/Aegis/linux-reverse-tunnel.err.log
tail -40 ~/Library/Logs/Aegis/windows-reverse-tunnel.err.log
```

**Fix:**
```bash
./scripts/lab/install-macos-linux-tunnel-launchd.sh
./scripts/lab/install-macos-windows-tunnel-launchd.sh
```

### Backend down

**Symptom:** `check_http` lines report `[fail]` for `actions-api`, `ingest`, or
`detection-pipeline`. Tunnel listeners may be present but agents cannot deliver events.

**Check:**
```bash
docker compose ps
curl -fsS http://localhost:8083/healthz
curl -fsS http://localhost:9091/healthz
curl -fsS http://localhost:8089/healthz
```

**Fix:**
```bash
docker compose down
docker compose up -d
# Wait for services to pass health checks, then re-run check-agents.sh
```

### Clock or freshness issue

**Symptom:** Tunnel and backend are healthy, but the API shows `stale` or
`offline` even though the agent ran recently. The `last_seen` timestamp in
`GET /agents` may differ significantly from the developer Mac clock.

**Check:**
```bash
# Compare clocks
date -u
curl -fsS http://localhost:8083/agents | python3 -c \
  'import json,sys; [print(a["agent_uid"], a.get("last_seen"), a.get("status")) for a in json.load(sys.stdin).get("agents", [])]'

# Check NTP sync on the lab host
ssh clarion@<LINUX_HOST> 'timedatectl status'
```

**Fix:** Sync NTP on the affected host (`sudo timedatectl set-ntp true` on
systemd Linux). Re-run the lab agent once after clock correction.

### Detection-pack status stale, but last applied pack is valid

**Symptom:** Agent shows `online` heartbeat but readiness score shows
`stale` or `needs_attention` in the detection-pack dimension. The agent
collected and reported events, but has not fetched a new pack recently.

**Check:**
```bash
# Query detection-pack rollout status for both lab agents
curl -fsS http://localhost:8089/v1/agents/linux-dev-agent-01/detection-pack-status
curl -fsS http://localhost:8089/v1/agents/windows-dev-agent-01/detection-pack-status
```

**Fix:** If `AEGIS_DETECTION_PACKS_ENABLED=true` is set but the agent cannot
reach the detection pipeline, confirm the tunnel port 8089 listener is present
on the lab host and that `AEGIS_CONTROLLER_URL` is set correctly.

If detection packs are intentionally disabled (`AEGIS_DETECTION_PACKS_ENABLED=false`),
the `detection_pack` readiness dimension will read `no rollout record` — this is
expected and does not indicate a connectivity fault.

## Reliability Notes

- Both tunnel scripts now clean stale remote forward listeners before creating new
  `-R` forwards. Set `AEGIS_TUNNEL_CLEAN_STALE=false` to disable cleanup.
- To avoid silent disconnects, both scripts use `ServerAliveInterval=30`,
  `ServerAliveCountMax=3`, and `ExitOnForwardFailure=yes`.

## Restart Runbook

Use this order after local stack or network changes:

1. Restart Docker Compose on the developer machine.
2. Restart macOS launchd tunnels.
3. Restart Linux systemd timer/service.
4. Restart Windows scheduled task (or one-shot run script).
5. Run health checks.

### 1) Restart Docker Compose

```bash
docker compose down
docker compose up -d
```

### 2) Restart macOS launchd tunnels

```bash
./scripts/lab/install-macos-windows-tunnel-launchd.sh
./scripts/lab/install-macos-linux-tunnel-launchd.sh
```

### 3) Restart Linux lab agent timer/service

```bash
ssh -i ~/.ssh/aegis_windows_lab clarion@192.168.101.31 'sudo systemctl restart aegis-linux-agent-lab.timer && sudo systemctl start aegis-linux-agent-lab.service'
```

### 4) Restart Windows scheduled task or one-shot run

```powershell
schtasks /Run /TN "AegisLabWindowsAgent"
```

or:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File C:\AegisLab\aegisflux\agents\windows-agent\scripts\run-lab-once.ps1
```

### 5) Run health checks

```bash
AEGIS_WINDOWS_HOST=192.168.12.101 \
AEGIS_LINUX_HOST=192.168.101.31 \
./scripts/lab/check-agents.sh
```

## Troubleshooting

- **Stale heartbeat:** verify `/agents` shows recent `last_seen`; if stale, run the
  one-shot agent scripts and inspect local logs.
- **Tunnel collision (`remote port forwarding failed`):** rerun install scripts.
  Stale `sshd-session` listeners are cleaned automatically.
- **Local port drift:** ensure Docker maps services on `8083`, `8089`, and `9091`
  on the developer machine before restarting tunnels.
- **Tunnel process drift on lab hosts:** run `./scripts/lab/check-agents.sh` with
  `AEGIS_WINDOWS_HOST` and `AEGIS_LINUX_HOST`; it now validates that remote
  `sshd-session` listeners exist for ports `9091`, `8083`, and `8089`.

## Linux Reverse Tunnel on macOS

The Linux reverse-tunnel fallback exposes `http://127.0.0.1:9091` on the Linux
host. Install the Linux tunnel as a macOS user `launchd` job:

```bash
./scripts/lab/install-macos-linux-tunnel-launchd.sh
```

The launch agent label is:

```text
net.aegis.linux-reverse-tunnel
```

Logs are written to:

```text
~/Library/Logs/Aegis/linux-reverse-tunnel.out.log
~/Library/Logs/Aegis/linux-reverse-tunnel.err.log
```

Remove it with:

```bash
./scripts/lab/uninstall-macos-linux-tunnel-launchd.sh
```

Optional environment overrides:

| Variable | Default |
|----------|---------|
| `AEGIS_LINUX_HOST` | `192.168.101.31` |
| `AEGIS_LINUX_USER` | `clarion` |
| `AEGIS_LINUX_SSH_KEY` | `$HOME/.ssh/aegis_windows_lab` |
| `AEGIS_TUNNEL_REMOTE_PORT` | `9091` |
| `AEGIS_TUNNEL_LOCAL_HOST` | `127.0.0.1` |
| `AEGIS_TUNNEL_LOCAL_PORT` | `9091` |
