param(
    [string]$AgentPath = "C:\AegisLab\aegisflux\agents\windows-agent\target\release\aegis-windows-agent.exe",
    [string]$BackendUrl = "http://127.0.0.1:9091",
    [string]$AgentId = "windows-dev-agent-01",
    [string]$DeviceId = "windows-dev-agent-01",
    [string]$SpoolPath = "C:\ProgramData\Aegis\Agent\spool\events.jsonl",
    [string]$LogPath = "C:\ProgramData\Aegis\Agent\logs\scheduled-task.log",
    [switch]$Stdout
)

$ErrorActionPreference = "Stop"

if (!(Test-Path -LiteralPath $AgentPath)) {
    throw "Windows agent binary was not found at $AgentPath"
}

$spoolDir = Split-Path -Parent $SpoolPath
$logDir = Split-Path -Parent $LogPath
New-Item -ItemType Directory -Force -Path $spoolDir | Out-Null
New-Item -ItemType Directory -Force -Path $logDir | Out-Null

$env:AEGIS_AGENT_ID = $AgentId
$env:AEGIS_DEVICE_ID = $DeviceId
$env:AEGIS_BACKEND_URL = $BackendUrl
$env:AEGIS_COLLECT_COMMAND_LINE = "true"
$env:AEGIS_EVENT_SPOOL = $SpoolPath

$startedAt = (Get-Date).ToString("o")
try {
    if ($Stdout) {
        & $AgentPath --once --stdout 2>&1 | Tee-Object -FilePath $LogPath -Append
    } else {
        & $AgentPath --once 2>&1 | Out-File -FilePath $LogPath -Append -Encoding utf8
    }

    if ($LASTEXITCODE -ne 0) {
        throw "Windows agent exited with code $LASTEXITCODE"
    }

    "[$startedAt] completed" | Out-File -FilePath $LogPath -Append -Encoding utf8
} catch {
    "[$startedAt] failed: $($_.Exception.Message)" | Out-File -FilePath $LogPath -Append -Encoding utf8
    throw
}
