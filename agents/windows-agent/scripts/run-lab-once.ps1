param(
    [string]$AgentPath = "C:\AegisLab\aegisflux\agents\windows-agent\target\release\aegis-windows-agent.exe",
    [string]$BackendUrl = "http://127.0.0.1:9091",
    [string]$AgentId = "windows-dev-agent-01",
    [string]$DeviceId = "windows-dev-agent-01",
    [string]$SpoolPath = "C:\ProgramData\Aegis\Agent\spool\events.jsonl",
    [string]$LogPath = "C:\ProgramData\Aegis\Agent\logs\scheduled-task.log",
    [string]$ActionsHeartbeatUrl = "http://127.0.0.1:8083/agents/heartbeat",
    [string]$ControllerUrl = "",
    [string]$DetectionPackCache = "",
    [string]$DetectionPackPublicKey = "",
    [switch]$DetectionPacksEnabled,
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
$env:AEGIS_DETECTION_PACKS_ENABLED = $(if ($DetectionPacksEnabled) { "true" } else { "false" })

if (![string]::IsNullOrWhiteSpace($ControllerUrl)) {
    $env:AEGIS_CONTROLLER_URL = $ControllerUrl
}
if (![string]::IsNullOrWhiteSpace($DetectionPackCache)) {
    $env:AEGIS_DETECTION_PACK_CACHE = $DetectionPackCache
}
if (![string]::IsNullOrWhiteSpace($DetectionPackPublicKey)) {
    $env:AEGIS_DETECTION_PACK_PUBLIC_KEY = $DetectionPackPublicKey
}

function Send-ActionsHeartbeat {
    $hostname = $env:COMPUTERNAME
    if ([string]::IsNullOrWhiteSpace($hostname)) {
        $hostname = $DeviceId
    }

    $primaryIp = ""
    try {
        $primaryIp = (Get-NetIPAddress -AddressFamily IPv4 -ErrorAction Stop |
            Where-Object { $_.IPAddress -notlike "169.254.*" -and $_.IPAddress -ne "127.0.0.1" } |
            Select-Object -First 1 -ExpandProperty IPAddress)
    } catch {
        $primaryIp = ""
    }

    $body = @{
        agent_uid = $AgentId
        org_id = "default-org"
        host_id = $DeviceId
        hostname = $hostname
        agent_version = "0.1.0"
        last_seen = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
        status = "online"
        labels = @("visibility-lab", "windows")
        note = "Registered from Windows lab visibility collector"
        capabilities = @{
            visibility = $true
            dynamic_detection_packs = [bool]$DetectionPacksEnabled
            platform = "windows"
        }
        platform = @{
            hostname = $hostname
            os = "windows"
            architecture = $env:PROCESSOR_ARCHITECTURE
            kernel_version = [System.Environment]::OSVersion.VersionString
            primary_ip = $primaryIp
        }
        network = @{
            primary_ip = $primaryIp
        }
    } | ConvertTo-Json -Depth 6 -Compress

    try {
        Invoke-RestMethod -Method Post -Uri $ActionsHeartbeatUrl -ContentType "application/json" -Body $body | Out-Null
    } catch {
        "[$((Get-Date).ToString("o"))] heartbeat failed: $($_.Exception.Message)" | Out-File -FilePath $LogPath -Append -Encoding utf8
    }
}

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

    Send-ActionsHeartbeat
    "[$startedAt] completed" | Out-File -FilePath $LogPath -Append -Encoding utf8
} catch {
    "[$startedAt] failed: $($_.Exception.Message)" | Out-File -FilePath $LogPath -Append -Encoding utf8
    throw
}
