# Aegis Lab Scripts

## Windows Reverse Tunnel on macOS

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

This means each remote host uses local loopback URLs:

- Ingest: `http://127.0.0.1:9091`
- Actions API: `http://127.0.0.1:8083`
- Detection pipeline: `http://127.0.0.1:8089`

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

The Linux lab systemd timer also posts to `http://127.0.0.1:9091` on the Linux
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
