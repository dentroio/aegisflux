param(
    [string]$AgentPath = "C:\AegisLab\aegisflux\agents\windows-agent\target\release\aegis-windows-agent.exe",
    [string]$BackendUrl = "http://127.0.0.1:9091",
    [string]$AgentId = "windows-dev-agent-01",
    [string]$DeviceId = "windows-dev-agent-01",
    [string]$SpoolPath = "C:\ProgramData\Aegis\Agent\spool\events.jsonl",
    [int]$MarkerSeconds = 120
)

$ErrorActionPreference = "Stop"

if (!(Test-Path -LiteralPath $AgentPath)) {
    throw "Windows agent binary was not found at $AgentPath"
}

$markerCommand = "title agent_runner openai tool && timeout /t $MarkerSeconds"
Start-Process cmd.exe -ArgumentList @("/k", $markerCommand)
Start-Sleep -Seconds 5

$env:AEGIS_AGENT_ID = $AgentId
$env:AEGIS_DEVICE_ID = $DeviceId
$env:AEGIS_BACKEND_URL = $BackendUrl
$env:AEGIS_COLLECT_COMMAND_LINE = "true"
$env:AEGIS_EVENT_SPOOL = $SpoolPath

& $AgentPath --once
if ($LASTEXITCODE -ne 0) {
    throw "Windows agent exited with code $LASTEXITCODE"
}

Write-Output "Aegis Windows detection smoke completed."
