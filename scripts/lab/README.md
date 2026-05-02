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
