//! Visibility collectors.

use std::time::SystemTime;

use crate::config::AgentConfig;
use crate::event::{AegisEvent, EventPayload};
use sysinfo::{ProcessRefreshKind, ProcessesToUpdate, System, UpdateKind};

/// A visibility collector.
pub trait Collector {
    /// Collector display name.
    fn name(&self) -> &'static str;

    /// Collect one batch of visibility events.
    fn collect_once(&self, config: &AgentConfig) -> Result<Vec<AegisEvent>, String>;
}

/// Phase 1 collector set.
#[derive(Debug, Default)]
pub struct CollectorSet {
    process: ProcessCollector,
    network: NetworkCollector,
    dns: DnsCollector,
    browser_history: BrowserHistoryCollector,
    browser_extensions: BrowserExtensionCollector,
    sase_inventory: SaseInventoryCollector,
}

impl CollectorSet {
    /// Return collectors in deterministic order.
    pub fn collectors(&self) -> Vec<&dyn Collector> {
        vec![
            &self.process,
            &self.network,
            &self.dns,
            &self.browser_history,
            &self.browser_extensions,
            &self.sase_inventory,
        ]
    }
}

#[derive(Debug, Default)]
struct ProcessCollector;

#[derive(Debug, Default)]
struct NetworkCollector;

#[derive(Debug, Default)]
struct DnsCollector;

#[derive(Debug, Default)]
struct BrowserHistoryCollector;

#[derive(Debug, Default)]
struct BrowserExtensionCollector;

#[derive(Debug, Default)]
struct SaseInventoryCollector;

impl Collector for ProcessCollector {
    fn name(&self) -> &'static str {
        "windows.process"
    }

    fn collect_once(&self, config: &AgentConfig) -> Result<Vec<AegisEvent>, String> {
        let mut system = System::new();
        system.refresh_processes_specifics(
            ProcessesToUpdate::All,
            true,
            ProcessRefreshKind::nothing()
                .with_cmd(UpdateKind::Always)
                .with_exe(UpdateKind::Always),
        );

        let mut events = Vec::new();
        events.push(collector_status(
            config,
            self.name(),
            "healthy",
            snapshot_message(system.processes().len()),
        ));

        let mut emitted = 0usize;
        for process in system.processes().values() {
            if emitted >= config.process_snapshot_limit {
                break;
            }
            let pid = process.pid().as_u32();
            let ppid = process.parent().map(|parent| parent.as_u32());
            let instance_id = process_guid(config, process.start_time(), pid);
            let parent_process_guid = process
                .parent()
                .and_then(|parent_pid| {
                    system
                        .process(parent_pid)
                        .map(|parent| (parent_pid, parent))
                })
                .map(|(parent_pid, parent)| {
                    process_guid(config, parent.start_time(), parent_pid.as_u32())
                });
            let command_line = command_line(process.cmd(), config.collect_command_line);

            events.push(AegisEvent::new(
                config,
                "aegis.process.started",
                SystemTime::now(),
                EventPayload::ProcessStarted {
                    process_guid: instance_id,
                    parent_process_guid,
                    pid,
                    ppid,
                    name: process.name().to_string_lossy().to_string(),
                    path: process.exe().map(|path| path.display().to_string()),
                    command_line,
                    user: None,
                    logon_session_id: None,
                    integrity_level: None,
                    sha256: None,
                    publisher: None,
                    collection_method: "sysinfo.snapshot".to_string(),
                },
            ));
            emitted += 1;
        }

        Ok(events)
    }
}

impl Collector for NetworkCollector {
    fn name(&self) -> &'static str {
        "windows.network"
    }

    fn collect_once(&self, config: &AgentConfig) -> Result<Vec<AegisEvent>, String> {
        #[cfg(windows)]
        {
            return collect_windows_network(config, self.name());
        }

        #[cfg(not(windows))]
        Ok(vec![collector_status(
            config,
            self.name(),
            "pending",
            windows_only_message(
                "network attribution collector requires ETW/IP helper implementation",
            ),
        )])
    }
}

impl Collector for DnsCollector {
    fn name(&self) -> &'static str {
        "windows.dns"
    }

    fn collect_once(&self, config: &AgentConfig) -> Result<Vec<AegisEvent>, String> {
        #[cfg(windows)]
        {
            return collect_windows_dns(config, self.name());
        }

        #[cfg(not(windows))]
        Ok(vec![collector_status(
            config,
            self.name(),
            "pending",
            windows_only_message(
                "DNS observation collector requires Windows DNS Client ETW implementation",
            ),
        )])
    }
}

impl Collector for BrowserHistoryCollector {
    fn name(&self) -> &'static str {
        "windows.browser_history"
    }

    fn collect_once(&self, config: &AgentConfig) -> Result<Vec<AegisEvent>, String> {
        #[cfg(windows)]
        {
            return collect_windows_browser_history(config, self.name());
        }

        #[cfg(not(windows))]
        Ok(vec![collector_status(
            config,
            self.name(),
            "pending",
            windows_only_message("browser history collector requires Windows browser profiles"),
        )])
    }
}

impl Collector for BrowserExtensionCollector {
    fn name(&self) -> &'static str {
        "windows.browser_extensions"
    }

    fn collect_once(&self, config: &AgentConfig) -> Result<Vec<AegisEvent>, String> {
        #[cfg(windows)]
        {
            return collect_windows_browser_extensions(config, self.name());
        }

        #[cfg(not(windows))]
        Ok(vec![collector_status(
            config,
            self.name(),
            "pending",
            windows_only_message("browser extension collector requires Windows browser profiles"),
        )])
    }
}

impl Collector for SaseInventoryCollector {
    fn name(&self) -> &'static str {
        "windows.sase_inventory"
    }

    fn collect_once(&self, config: &AgentConfig) -> Result<Vec<AegisEvent>, String> {
        #[cfg(windows)]
        {
            return collect_windows_sase_inventory(config, self.name());
        }

        #[cfg(not(windows))]
        Ok(vec![collector_status(
            config,
            self.name(),
            "pending",
            windows_only_message("SSE/SASE inventory collector requires Windows host state"),
        )])
    }
}

/// Full process snapshot as `aegis.process.started` (legacy `--once` without lifecycle).
pub(crate) fn collect_process_snapshot_batch(
    config: &AgentConfig,
) -> Result<Vec<AegisEvent>, String> {
    let collector = ProcessCollector;
    Collector::collect_once(&collector, config)
}

/// Infer DNS process attribution from outbound flows whose remote IP appears in DNS answers.
pub fn correlate_dns_to_flow_attribution(events: &mut [AegisEvent]) {
    let mut best_by_ip: std::collections::HashMap<String, (u32, String, f32)> =
        std::collections::HashMap::new();
    for event in events.iter() {
        if let EventPayload::FlowStarted {
            remote_ip,
            pid,
            process_guid,
            attribution_confidence,
            ..
        } = &event.payload
        {
            if let (Some(pid), Some(guid)) = (*pid, process_guid.clone()) {
                let entry = best_by_ip.entry(remote_ip.clone()).or_insert((
                    pid,
                    guid.clone(),
                    *attribution_confidence,
                ));
                if *attribution_confidence > entry.2 {
                    *entry = (pid, guid, *attribution_confidence);
                }
            }
        }
    }

    if best_by_ip.is_empty() {
        return;
    }

    for event in events.iter_mut() {
        if let EventPayload::DnsObserved {
            answers,
            pid,
            process_guid,
            correlation_method,
            correlation_confidence,
            ..
        } = &mut event.payload
        {
            if pid.is_some() {
                continue;
            }
            for answer in answers.iter() {
                if !is_ip_address(answer) {
                    continue;
                }
                if let Some((matched_pid, guid, flow_conf)) = best_by_ip.get(answer) {
                    *pid = Some(*matched_pid);
                    *process_guid = Some(guid.clone());
                    *correlation_method = "aegis.dns.flow_remote_ip_match".to_string();
                    *correlation_confidence = (*flow_conf * 0.88).min(0.94);
                    break;
                }
            }
        }
    }
}

/// Enrich flow observations with hostnames from DNS answers in the same batch.
pub fn enrich_flow_hostnames(events: &mut [AegisEvent]) {
    let mut host_by_ip = std::collections::HashMap::new();
    for event in events.iter() {
        if let EventPayload::DnsObserved { query, answers, .. } = &event.payload {
            if query.trim().is_empty() {
                continue;
            }
            for answer in answers {
                if is_ip_address(answer) {
                    host_by_ip
                        .entry(answer.clone())
                        .or_insert_with(|| query.clone());
                }
            }
        }
    }

    if host_by_ip.is_empty() {
        return;
    }

    for event in events.iter_mut() {
        if let EventPayload::FlowStarted {
            remote_ip,
            remote_hostname,
            ..
        } = &mut event.payload
        {
            if remote_hostname.is_none() {
                if let Some(hostname) = host_by_ip.get(remote_ip) {
                    *remote_hostname = Some(hostname.clone());
                }
            }
        }
    }
}

fn collector_status(
    config: &AgentConfig,
    collector: &str,
    status: &str,
    message: String,
) -> AegisEvent {
    AegisEvent::new(
        config,
        "aegis.collector.status",
        SystemTime::now(),
        EventPayload::CollectorStatus {
            collector: collector.to_string(),
            status: status.to_string(),
            message,
        },
    )
}

fn is_ip_address(value: &str) -> bool {
    value.parse::<std::net::IpAddr>().is_ok()
}

#[cfg(not(windows))]
fn windows_only_message(message: &str) -> String {
    format!("{message}; running on non-Windows development host")
}

fn snapshot_message(count: usize) -> String {
    let suffix = if cfg!(windows) {
        ""
    } else {
        "; running on non-Windows development host"
    };
    format!("process snapshot collector observed {count} processes{suffix}")
}

fn process_guid(config: &AgentConfig, start_time_secs: u64, pid: u32) -> String {
    format!("{}:{}:{}", config.device_id, start_time_secs, pid)
}

pub(crate) fn command_line(parts: &[std::ffi::OsString], enabled: bool) -> Option<String> {
    if !enabled {
        return None;
    }

    if parts.is_empty() {
        return None;
    }

    Some(sanitize_command_line(
        &parts
            .iter()
            .map(|part| part.to_string_lossy())
            .collect::<Vec<_>>()
            .join(" "),
    ))
}

#[cfg(windows)]
fn collect_windows_network(
    config: &AgentConfig,
    collector_name: &str,
) -> Result<Vec<AegisEvent>, String> {
    let output = std::process::Command::new("netstat")
        .args(["-ano"])
        .output()
        .map_err(|err| format!("failed to run netstat -ano: {err}"))?;

    if !output.status.success() {
        return Ok(vec![collector_status(
            config,
            collector_name,
            "degraded",
            format!("netstat -ano failed with status {}", output.status),
        )]);
    }

    let stdout = String::from_utf8_lossy(&output.stdout);
    let flows = parse_netstat_ano(&stdout);
    let process_map = process_snapshot_map(config);
    let mut events = vec![collector_status(
        config,
        collector_name,
        "healthy",
        format!("network snapshot collector observed {} flows", flows.len()),
    )];

    for flow in flows {
        let process = flow.pid.and_then(|pid| process_map.get(&pid).cloned());
        let flow_key = flow_id(config, &flow);
        let user = process.as_ref().and_then(|process| process.user.clone());
        let process_path = process.as_ref().and_then(|process| process.path.clone());
        let attribution_confidence = if flow.pid.is_some() { 0.72 } else { 0.35 };

        if flow.protocol == "tcp" {
            if let Some(state) = &flow.connection_state {
                if is_tcp_listen_state(state) {
                    continue;
                }
                if is_tcp_terminal_state(state) {
                    events.push(AegisEvent::new(
                        config,
                        "aegis.flow.ended",
                        SystemTime::now(),
                        EventPayload::FlowEnded {
                            flow_id: flow_key,
                            process_guid: process
                                .as_ref()
                                .map(|process| process.process_guid.clone()),
                            pid: flow.pid,
                            bytes_sent: None,
                            bytes_received: None,
                            duration_ms: None,
                            connection_state: Some(state.clone()),
                            collection_method: "windows.netstat_ano.snapshot".to_string(),
                        },
                    ));
                    continue;
                }
            }
        }

        events.push(AegisEvent::new(
            config,
            "aegis.flow.started",
            SystemTime::now(),
            EventPayload::FlowStarted {
                flow_id: flow_key,
                process_guid: process.as_ref().map(|process| process.process_guid.clone()),
                pid: flow.pid,
                process_name: process.as_ref().map(|process| process.name.clone()),
                process_path,
                user,
                protocol: flow.protocol,
                direction: flow.direction,
                local_ip: flow.local_ip,
                local_port: flow.local_port,
                remote_ip: flow.remote_ip,
                remote_port: flow.remote_port,
                remote_hostname: None,
                connection_state: flow.connection_state.clone(),
                bytes_sent: None,
                bytes_received: None,
                attribution_method: "windows.netstat_ano.pid".to_string(),
                attribution_confidence,
            },
        ));
    }

    Ok(events)
}

#[cfg(windows)]
fn collect_windows_dns(
    config: &AgentConfig,
    collector_name: &str,
) -> Result<Vec<AegisEvent>, String> {
    let output = std::process::Command::new("ipconfig")
        .arg("/displaydns")
        .output()
        .map_err(|err| format!("failed to run ipconfig /displaydns: {err}"))?;

    if !output.status.success() {
        return Ok(vec![collector_status(
            config,
            collector_name,
            "degraded",
            format!("ipconfig /displaydns failed with status {}", output.status),
        )]);
    }

    let stdout = String::from_utf8_lossy(&output.stdout);
    let observations = parse_ipconfig_displaydns(&stdout);
    let mut events = vec![collector_status(
        config,
        collector_name,
        "healthy",
        format!(
            "DNS cache collector observed {} cached names",
            observations.len()
        ),
    )];

    for observation in observations {
        events.push(AegisEvent::new(
            config,
            "aegis.dns.observed",
            SystemTime::now(),
            EventPayload::DnsObserved {
                query: observation.query,
                query_type: observation.query_type,
                answers: observation.answers,
                resolver: None,
                process_guid: None,
                pid: None,
                correlation_method: "windows.ipconfig_displaydns.cache".to_string(),
                correlation_confidence: 0.35,
            },
        ));
    }

    Ok(events)
}

#[cfg(windows)]
fn collect_windows_browser_history(
    config: &AgentConfig,
    collector_name: &str,
) -> Result<Vec<AegisEvent>, String> {
    let profiles = browser_history_paths();
    let mut observed = std::collections::BTreeSet::new();
    let mut profile_count = 0usize;

    for profile in profiles {
        if !profile.history_path.exists() {
            continue;
        }
        profile_count += 1;
        match read_browser_history_hosts(&profile.history_path) {
            Ok(hosts) => {
                for host in hosts {
                    observed.insert((profile.browser.clone(), host));
                }
            }
            Err(_err) => {
                continue;
            }
        }
    }

    let mut events = vec![collector_status(
        config,
        collector_name,
        "healthy",
        format!(
            "browser history collector observed {} domains across {} profiles",
            observed.len(),
            profile_count
        ),
    )];

    for (browser, host) in observed.into_iter().take(200) {
        events.push(AegisEvent::new(
            config,
            "aegis.dns.observed",
            SystemTime::now(),
            EventPayload::DnsObserved {
                query: host,
                query_type: Some("BROWSER_HISTORY".to_string()),
                answers: Vec::new(),
                resolver: Some(browser),
                process_guid: None,
                pid: None,
                correlation_method: "windows.browser_history.recent_url".to_string(),
                correlation_confidence: 0.58,
            },
        ));
    }

    Ok(events)
}

#[cfg(windows)]
fn collect_windows_browser_extensions(
    config: &AgentConfig,
    collector_name: &str,
) -> Result<Vec<AegisEvent>, String> {
    let profiles = browser_profile_roots();
    let mut observations = Vec::new();

    for profile in profiles {
        observations.extend(read_browser_extension_observations(&profile));
    }

    observations.sort_by(|left, right| {
        left.browser
            .cmp(&right.browser)
            .then(left.profile.cmp(&right.profile))
            .then(left.extension_id.cmp(&right.extension_id))
            .then(left.version.cmp(&right.version))
    });
    observations.dedup_by(|left, right| {
        left.browser == right.browser
            && left.profile == right.profile
            && left.extension_id == right.extension_id
            && left.version == right.version
    });

    let mut events = vec![collector_status(
        config,
        collector_name,
        "healthy",
        format!(
            "browser extension collector observed {} extensions across {} profiles",
            observations.len(),
            observations
                .iter()
                .map(|observation| (&observation.browser, &observation.profile))
                .collect::<std::collections::BTreeSet<_>>()
                .len()
        ),
    )];

    for observation in observations.into_iter().take(300) {
        events.push(AegisEvent::new(
            config,
            "aegis.browser_extension.observed",
            SystemTime::now(),
            EventPayload::BrowserExtensionObserved {
                browser: observation.browser,
                profile: observation.profile,
                extension_id: observation.extension_id,
                name: observation.name,
                version: observation.version,
                manifest_version: observation.manifest_version,
                permissions: observation.permissions,
                host_permissions: observation.host_permissions,
                path: observation.path,
                collection_method: "windows.chromium_extension_manifest".to_string(),
            },
        ));
    }

    Ok(events)
}

#[cfg(windows)]
fn collect_windows_sase_inventory(
    config: &AgentConfig,
    collector_name: &str,
) -> Result<Vec<AegisEvent>, String> {
    let mut observations = Vec::new();
    observations.extend(collect_sase_installed_products());
    observations.extend(collect_sase_services());
    observations.extend(collect_sase_processes());
    observations.extend(collect_sase_network_adapters());
    observations.extend(collect_sase_proxy_config());

    observations.sort_by(|left, right| {
        left.component_type
            .cmp(&right.component_type)
            .then(left.vendor.cmp(&right.vendor))
            .then(left.product.cmp(&right.product))
            .then(left.name.cmp(&right.name))
            .then(left.source.cmp(&right.source))
    });
    observations.dedup_by(|left, right| {
        left.component_type == right.component_type
            && left.vendor == right.vendor
            && left.product == right.product
            && left.name == right.name
            && left.source == right.source
    });

    let mut events = vec![collector_status(
        config,
        collector_name,
        "healthy",
        format!(
            "SSE/SASE inventory collector observed {} components",
            observations.len()
        ),
    )];

    for observation in observations.into_iter().take(200) {
        events.push(AegisEvent::new(
            config,
            "aegis.sase_component.observed",
            SystemTime::now(),
            EventPayload::SaseComponentObserved {
                component_type: observation.component_type,
                vendor: observation.vendor,
                product: observation.product,
                name: observation.name,
                version: observation.version,
                status: observation.status,
                source: observation.source,
                evidence: observation.evidence,
                collection_method: observation.collection_method,
            },
        ));
    }

    Ok(events)
}

#[cfg(any(windows, test))]
#[derive(Debug, Clone, PartialEq, Eq)]
struct SaseComponentObservation {
    component_type: String,
    vendor: String,
    product: String,
    name: String,
    version: Option<String>,
    status: Option<String>,
    source: String,
    evidence: Vec<String>,
    collection_method: String,
}

#[cfg(any(windows, test))]
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
struct SaseVendorSignature {
    vendor: &'static str,
    product: &'static str,
    patterns: &'static [&'static str],
}

#[cfg(any(windows, test))]
const SASE_VENDOR_SIGNATURES: &[SaseVendorSignature] = &[
    SaseVendorSignature {
        vendor: "Zscaler",
        product: "Zscaler Client Connector",
        patterns: &["zscaler", "zscaler client connector", "zapp"],
    },
    SaseVendorSignature {
        vendor: "Palo Alto Networks",
        product: "GlobalProtect / Prisma Access",
        patterns: &["globalprotect", "prisma access", "palo alto networks"],
    },
    SaseVendorSignature {
        vendor: "Cisco",
        product: "Cisco Secure Client / Umbrella",
        patterns: &["cisco secure client", "anyconnect", "umbrella"],
    },
    SaseVendorSignature {
        vendor: "Netskope",
        product: "Netskope Client",
        patterns: &["netskope"],
    },
    SaseVendorSignature {
        vendor: "Cloudflare",
        product: "Cloudflare WARP",
        patterns: &["cloudflare warp", "warp-svc", "cloudflare"],
    },
    SaseVendorSignature {
        vendor: "iboss",
        product: "iboss Cloud Connector",
        patterns: &["iboss"],
    },
    SaseVendorSignature {
        vendor: "Fortinet",
        product: "FortiClient",
        patterns: &["forticlient", "fortinet"],
    },
    SaseVendorSignature {
        vendor: "Check Point",
        product: "Harmony Endpoint",
        patterns: &["checkpoint", "check point", "harmony endpoint"],
    },
    SaseVendorSignature {
        vendor: "Cato Networks",
        product: "Cato Client",
        patterns: &["cato client", "cato networks"],
    },
    SaseVendorSignature {
        vendor: "Tailscale",
        product: "Tailscale",
        patterns: &["tailscale"],
    },
];

#[cfg(windows)]
fn collect_sase_installed_products() -> Vec<SaseComponentObservation> {
    let script = r#"
$paths = @(
  'HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall\*',
  'HKLM:\Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*',
  'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\*'
)
foreach ($path in $paths) {
  Get-ItemProperty -Path $path -ErrorAction SilentlyContinue |
    Where-Object { $_.DisplayName } |
    ForEach-Object {
      "$($_.DisplayName)`t$($_.DisplayVersion)`t$($_.Publisher)`t$($_.InstallLocation)"
    }
}
"#;
    let output = run_powershell_script(script).unwrap_or_default();
    parse_sase_installed_products(&output)
}

#[cfg(any(windows, test))]
fn parse_sase_installed_products(output: &str) -> Vec<SaseComponentObservation> {
    output
        .lines()
        .filter_map(|line| {
            let fields = split_tsv_fields(line, 4);
            let display_name = fields.first()?.trim();
            if display_name.is_empty() {
                return None;
            }
            let version = empty_to_none(fields.get(1).map(String::as_str).unwrap_or_default());
            let publisher = fields.get(2).map(String::as_str).unwrap_or_default();
            let install_location = fields.get(3).map(String::as_str).unwrap_or_default();
            let haystack = format!("{display_name} {publisher} {install_location}");
            let signature = match_sase_vendor(&haystack)?;
            Some(SaseComponentObservation {
                component_type: "installed_product".to_string(),
                vendor: signature.vendor.to_string(),
                product: signature.product.to_string(),
                name: display_name.to_string(),
                version,
                status: None,
                source: "windows.uninstall_registry".to_string(),
                evidence: compact_evidence(&[display_name, publisher, install_location]),
                collection_method: "windows.registry.uninstall".to_string(),
            })
        })
        .collect()
}

#[cfg(windows)]
fn collect_sase_services() -> Vec<SaseComponentObservation> {
    let script = r#"
Get-Service |
  ForEach-Object {
    "$($_.Name)`t$($_.DisplayName)`t$($_.Status)"
  }
"#;
    let output = run_powershell_script(script).unwrap_or_default();
    parse_sase_services(&output)
}

#[cfg(any(windows, test))]
fn parse_sase_services(output: &str) -> Vec<SaseComponentObservation> {
    output
        .lines()
        .filter_map(|line| {
            let fields = split_tsv_fields(line, 3);
            let service_name = fields.first()?.trim();
            let display_name = fields.get(1).map(String::as_str).unwrap_or_default();
            let status = empty_to_none(fields.get(2).map(String::as_str).unwrap_or_default());
            let haystack = format!("{service_name} {display_name}");
            let signature = match_sase_vendor(&haystack)?;
            Some(SaseComponentObservation {
                component_type: "service".to_string(),
                vendor: signature.vendor.to_string(),
                product: signature.product.to_string(),
                name: if display_name.trim().is_empty() {
                    service_name.to_string()
                } else {
                    display_name.to_string()
                },
                version: None,
                status,
                source: service_name.to_string(),
                evidence: compact_evidence(&[service_name, display_name]),
                collection_method: "windows.service_control_manager".to_string(),
            })
        })
        .collect()
}

#[cfg(windows)]
fn collect_sase_processes() -> Vec<SaseComponentObservation> {
    let mut system = System::new();
    system.refresh_processes_specifics(
        ProcessesToUpdate::All,
        true,
        ProcessRefreshKind::nothing()
            .with_cmd(UpdateKind::Always)
            .with_exe(UpdateKind::Always),
    );

    system
        .processes()
        .values()
        .filter_map(|process| {
            let name = process.name().to_string_lossy().to_string();
            let path = process
                .exe()
                .map(|path| path.display().to_string())
                .unwrap_or_default();
            let command_line = process
                .cmd()
                .iter()
                .map(|part| part.to_string_lossy())
                .collect::<Vec<_>>()
                .join(" ");
            let haystack = format!("{name} {path} {command_line}");
            let signature = match_sase_vendor(&haystack)?;
            Some(SaseComponentObservation {
                component_type: "process".to_string(),
                vendor: signature.vendor.to_string(),
                product: signature.product.to_string(),
                name,
                version: None,
                status: Some("running".to_string()),
                source: format!("pid:{}", process.pid().as_u32()),
                evidence: compact_evidence(&[&path, &command_line]),
                collection_method: "sysinfo.process_snapshot".to_string(),
            })
        })
        .collect()
}

#[cfg(windows)]
fn collect_sase_network_adapters() -> Vec<SaseComponentObservation> {
    let script = r#"
Get-NetAdapter -ErrorAction SilentlyContinue |
  ForEach-Object {
    "$($_.Name)`t$($_.InterfaceDescription)`t$($_.Status)"
  }
"#;
    let output = run_powershell_script(script).unwrap_or_default();
    parse_sase_network_adapters(&output)
}

#[cfg(any(windows, test))]
fn parse_sase_network_adapters(output: &str) -> Vec<SaseComponentObservation> {
    output
        .lines()
        .filter_map(|line| {
            let fields = split_tsv_fields(line, 3);
            let name = fields.first()?.trim();
            let description = fields.get(1).map(String::as_str).unwrap_or_default();
            let status = empty_to_none(fields.get(2).map(String::as_str).unwrap_or_default());
            let haystack = format!("{name} {description}");
            let signature = match_sase_vendor(&haystack)?;
            Some(SaseComponentObservation {
                component_type: "network_adapter".to_string(),
                vendor: signature.vendor.to_string(),
                product: signature.product.to_string(),
                name: name.to_string(),
                version: None,
                status,
                source: "windows.net_adapter".to_string(),
                evidence: compact_evidence(&[name, description]),
                collection_method: "windows.get_net_adapter".to_string(),
            })
        })
        .collect()
}

#[cfg(windows)]
fn collect_sase_proxy_config() -> Vec<SaseComponentObservation> {
    let mut observations = Vec::new();
    if let Ok(output) = std::process::Command::new("netsh")
        .args(["winhttp", "show", "proxy"])
        .output()
    {
        let stdout = String::from_utf8_lossy(&output.stdout);
        observations.extend(parse_sase_proxy_config(
            "windows.winhttp_proxy",
            "windows.netsh_winhttp",
            &stdout,
        ));
    }

    let script = r#"
$key = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings'
$props = Get-ItemProperty -Path $key -ErrorAction SilentlyContinue
if ($props) {
  "ProxyEnable`t$($props.ProxyEnable)"
  "ProxyServer`t$($props.ProxyServer)"
  "AutoConfigURL`t$($props.AutoConfigURL)"
  "AutoDetect`t$($props.AutoDetect)"
}
"#;
    if let Ok(output) = run_powershell_script(script) {
        observations.extend(parse_sase_proxy_config(
            "windows.user_proxy",
            "windows.internet_settings_registry",
            &output,
        ));
    }

    observations
}

#[cfg(any(windows, test))]
fn parse_sase_proxy_config(
    source: &str,
    collection_method: &str,
    output: &str,
) -> Vec<SaseComponentObservation> {
    let normalized = output.trim();
    if normalized.is_empty() {
        return Vec::new();
    }
    let lower = normalized.to_ascii_lowercase();
    if lower.contains("direct access") || !proxy_config_has_value(normalized) {
        return Vec::new();
    }

    let signature = match_sase_vendor(normalized);
    vec![SaseComponentObservation {
        component_type: "proxy_config".to_string(),
        vendor: signature
            .map(|signature| signature.vendor)
            .unwrap_or("unknown")
            .to_string(),
        product: signature
            .map(|signature| signature.product)
            .unwrap_or("Enterprise proxy/PAC")
            .to_string(),
        name: source.to_string(),
        version: None,
        status: Some("configured".to_string()),
        source: source.to_string(),
        evidence: compact_evidence(
            &normalized
                .lines()
                .map(str::trim)
                .filter(|line| !line.is_empty())
                .take(8)
                .collect::<Vec<_>>(),
        ),
        collection_method: collection_method.to_string(),
    }]
}

#[cfg(any(windows, test))]
fn proxy_config_has_value(output: &str) -> bool {
    output.lines().any(|line| {
        let trimmed = line.trim();
        let lower = trimmed.to_ascii_lowercase();
        if lower.contains(".pac") {
            return true;
        }
        if let Some((key, value)) = trimmed.split_once('\t') {
            let key = key.trim().to_ascii_lowercase();
            let value = value.trim();
            return (key == "proxyenable" && value == "1")
                || ((key == "proxyserver" || key == "autoconfigurl") && !value.is_empty());
        }
        if let Some((key, value)) = trimmed.split_once(':') {
            let key = key.trim().to_ascii_lowercase();
            let value = value.trim();
            return (key.contains("proxy server") || key.contains("proxy script"))
                && !value.is_empty()
                && !value.eq_ignore_ascii_case("none");
        }
        false
    })
}

#[cfg(windows)]
fn run_powershell_script(script: &str) -> Result<String, String> {
    let output = std::process::Command::new("powershell")
        .args([
            "-NoProfile",
            "-ExecutionPolicy",
            "Bypass",
            "-Command",
            script,
        ])
        .output()
        .map_err(|err| format!("failed to run PowerShell collector script: {err}"))?;

    if !output.status.success() {
        return Err(format!(
            "PowerShell collector script failed with status {}",
            output.status
        ));
    }

    Ok(String::from_utf8_lossy(&output.stdout).to_string())
}

#[cfg(any(windows, test))]
fn match_sase_vendor(value: &str) -> Option<SaseVendorSignature> {
    let lower = value.to_ascii_lowercase();
    SASE_VENDOR_SIGNATURES.iter().copied().find(|signature| {
        signature
            .patterns
            .iter()
            .any(|pattern| lower.contains(pattern))
    })
}

#[cfg(any(windows, test))]
fn split_tsv_fields(line: &str, expected: usize) -> Vec<String> {
    let mut fields = line
        .split('\t')
        .map(|value| value.trim().to_string())
        .collect::<Vec<_>>();
    while fields.len() < expected {
        fields.push(String::new());
    }
    fields
}

#[cfg(any(windows, test))]
fn empty_to_none(value: &str) -> Option<String> {
    let trimmed = value.trim();
    if trimmed.is_empty() {
        None
    } else {
        Some(trimmed.to_string())
    }
}

#[cfg(any(windows, test))]
fn compact_evidence(values: &[&str]) -> Vec<String> {
    values
        .iter()
        .map(|value| value.trim())
        .filter(|value| !value.is_empty())
        .map(|value| value.chars().take(240).collect::<String>())
        .collect::<std::collections::BTreeSet<_>>()
        .into_iter()
        .collect()
}

#[cfg(windows)]
#[derive(Debug, Clone)]
struct BrowserHistoryProfile {
    browser: String,
    history_path: std::path::PathBuf,
}

#[cfg(windows)]
fn browser_history_paths() -> Vec<BrowserHistoryProfile> {
    browser_profile_roots()
        .into_iter()
        .map(|profile| BrowserHistoryProfile {
            browser: profile.browser,
            history_path: profile.root.join("History"),
        })
        .collect()
}

#[cfg(windows)]
#[derive(Debug, Clone)]
struct BrowserProfileRoot {
    browser: String,
    profile: String,
    root: std::path::PathBuf,
}

#[cfg(windows)]
fn browser_profile_roots() -> Vec<BrowserProfileRoot> {
    let mut profiles = Vec::new();
    for local_app_data in browser_local_app_data_roots() {
        let account = windows_account_from_local_app_data(&local_app_data);
        let browser_roots = [
            ("edge", local_app_data.join(r"Microsoft\Edge\User Data")),
            ("chrome", local_app_data.join(r"Google\Chrome\User Data")),
            (
                "brave",
                local_app_data.join(r"BraveSoftware\Brave-Browser\User Data"),
            ),
        ];

        for (browser, root) in browser_roots {
            collect_browser_profiles(browser, &account, &root, &mut profiles);
        }
    }

    profiles
}

#[cfg(windows)]
fn windows_account_from_local_app_data(local_app_data: &std::path::Path) -> String {
    local_app_data
        .parent()
        .and_then(|path| path.parent())
        .and_then(|path| path.file_name())
        .map(|value| value.to_string_lossy().to_string())
        .filter(|value| !value.is_empty())
        .unwrap_or_else(|| "current-user".to_string())
}

#[cfg(windows)]
fn browser_local_app_data_roots() -> Vec<std::path::PathBuf> {
    let mut roots = Vec::new();
    if let Ok(value) = std::env::var("LOCALAPPDATA") {
        roots.push(std::path::PathBuf::from(value));
    }

    let users_root = std::path::Path::new(r"C:\Users");
    if let Ok(entries) = std::fs::read_dir(users_root) {
        for entry in entries.flatten() {
            let local_app_data = entry.path().join(r"AppData\Local");
            if local_app_data.exists() && !roots.iter().any(|root| root == &local_app_data) {
                roots.push(local_app_data);
            }
        }
    }

    roots
}

#[cfg(windows)]
fn collect_browser_profiles(
    browser: &str,
    account: &str,
    root: &std::path::Path,
    profiles: &mut Vec<BrowserProfileRoot>,
) {
    if !root.exists() {
        return;
    }

    for profile_name in ["Default", "Profile 1", "Profile 2", "Profile 3"] {
        let profile_root = root.join(profile_name);
        profiles.push(BrowserProfileRoot {
            browser: format!("{browser}:{account}:{profile_name}"),
            profile: format!("{account}/{profile_name}"),
            root: profile_root,
        });
    }

    if let Ok(entries) = std::fs::read_dir(root) {
        for entry in entries.flatten() {
            let file_name = entry.file_name().to_string_lossy().to_string();
            if !file_name.starts_with("Profile ") {
                continue;
            }
            profiles.push(BrowserProfileRoot {
                browser: format!("{browser}:{account}:{file_name}"),
                profile: format!("{account}/{file_name}"),
                root: entry.path(),
            });
        }
    }
}

#[cfg(windows)]
#[derive(Debug, Clone, PartialEq, Eq)]
struct BrowserExtensionObservation {
    browser: String,
    profile: String,
    extension_id: String,
    name: String,
    version: String,
    manifest_version: Option<u32>,
    permissions: Vec<String>,
    host_permissions: Vec<String>,
    path: String,
}

#[cfg(windows)]
fn read_browser_extension_observations(
    profile: &BrowserProfileRoot,
) -> Vec<BrowserExtensionObservation> {
    let extensions_root = profile.root.join("Extensions");
    let mut observations = Vec::new();
    let entries = match std::fs::read_dir(extensions_root) {
        Ok(entries) => entries,
        Err(_) => return observations,
    };

    for extension_entry in entries.flatten() {
        let extension_id = extension_entry.file_name().to_string_lossy().to_string();
        if !is_chromium_extension_id(&extension_id) {
            continue;
        }
        if let Ok(version_entries) = std::fs::read_dir(extension_entry.path()) {
            for version_entry in version_entries.flatten() {
                let manifest_path = version_entry.path().join("manifest.json");
                if let Some(manifest) = parse_extension_manifest(&manifest_path) {
                    observations.push(BrowserExtensionObservation {
                        browser: profile.browser.clone(),
                        profile: profile.profile.clone(),
                        extension_id: extension_id.clone(),
                        name: manifest.name,
                        version: manifest.version,
                        manifest_version: manifest.manifest_version,
                        permissions: manifest.permissions,
                        host_permissions: manifest.host_permissions,
                        path: manifest_path.display().to_string(),
                    });
                }
            }
        }
    }

    observations
}

#[cfg(any(windows, test))]
#[derive(Debug, Clone, PartialEq, Eq)]
struct ExtensionManifestSummary {
    name: String,
    version: String,
    manifest_version: Option<u32>,
    permissions: Vec<String>,
    host_permissions: Vec<String>,
}

#[cfg(windows)]
fn parse_extension_manifest(path: &std::path::Path) -> Option<ExtensionManifestSummary> {
    let raw = std::fs::read_to_string(path).ok()?;
    parse_extension_manifest_json(&raw)
}

#[cfg(any(windows, test))]
fn parse_extension_manifest_json(raw: &str) -> Option<ExtensionManifestSummary> {
    let value: serde_json::Value = serde_json::from_str(raw).ok()?;
    let object = value.as_object()?;
    let name = object
        .get("name")
        .and_then(|value| value.as_str())
        .unwrap_or("unknown")
        .to_string();
    let version = object
        .get("version")
        .and_then(|value| value.as_str())
        .unwrap_or("unknown")
        .to_string();
    let manifest_version = object
        .get("manifest_version")
        .and_then(|value| value.as_u64())
        .and_then(|value| u32::try_from(value).ok());
    let permissions = string_array_field(object, "permissions");
    let mut host_permissions = string_array_field(object, "host_permissions");
    for permission in &permissions {
        if looks_like_host_permission(permission) && !host_permissions.contains(permission) {
            host_permissions.push(permission.clone());
        }
    }

    Some(ExtensionManifestSummary {
        name,
        version,
        manifest_version,
        permissions,
        host_permissions,
    })
}

#[cfg(any(windows, test))]
fn string_array_field(
    object: &serde_json::Map<String, serde_json::Value>,
    field: &str,
) -> Vec<String> {
    object
        .get(field)
        .and_then(|value| value.as_array())
        .map(|values| {
            values
                .iter()
                .filter_map(|value| value.as_str())
                .map(|value| value.to_string())
                .collect()
        })
        .unwrap_or_default()
}

#[cfg(any(windows, test))]
fn looks_like_host_permission(value: &str) -> bool {
    value.contains("://") || value == "<all_urls>"
}

#[cfg(windows)]
fn is_chromium_extension_id(value: &str) -> bool {
    value.len() == 32 && value.chars().all(|ch| matches!(ch, 'a'..='p'))
}

#[cfg(windows)]
fn read_browser_history_hosts(
    path: &std::path::Path,
) -> Result<std::collections::BTreeSet<String>, String> {
    let temp_path = std::env::temp_dir().join(format!(
        "aegis-browser-history-{}-{}.sqlite",
        std::process::id(),
        path.file_name()
            .and_then(|name| name.to_str())
            .unwrap_or("history")
    ));
    std::fs::copy(path, &temp_path)
        .map_err(|err| format!("failed to copy browser history {}: {err}", path.display()))?;

    let result = query_browser_history_hosts(&temp_path);
    let _ = std::fs::remove_file(&temp_path);
    result
}

#[cfg(windows)]
fn query_browser_history_hosts(
    path: &std::path::Path,
) -> Result<std::collections::BTreeSet<String>, String> {
    let connection = rusqlite::Connection::open(path)
        .map_err(|err| format!("failed to open browser history {}: {err}", path.display()))?;
    let mut statement = connection
        .prepare("SELECT url FROM urls ORDER BY last_visit_time DESC LIMIT 500")
        .map_err(|err| format!("failed to query browser history {}: {err}", path.display()))?;
    let rows = statement
        .query_map([], |row| row.get::<_, String>(0))
        .map_err(|err| format!("failed to read browser history {}: {err}", path.display()))?;

    let mut hosts = std::collections::BTreeSet::new();
    for row in rows {
        if let Ok(url) = row {
            if let Some(host) = host_from_url(&url) {
                hosts.insert(host);
            }
        }
    }
    Ok(hosts)
}

#[cfg(windows)]
#[derive(Debug, Clone)]
struct ProcessSnapshot {
    process_guid: String,
    name: String,
    path: Option<String>,
    user: Option<String>,
}

#[cfg(windows)]
fn process_snapshot_map(config: &AgentConfig) -> std::collections::HashMap<u32, ProcessSnapshot> {
    let mut system = System::new();
    system.refresh_users_list();
    system.refresh_processes_specifics(
        ProcessesToUpdate::All,
        true,
        ProcessRefreshKind::nothing().with_exe(UpdateKind::Always),
    );

    system
        .processes()
        .values()
        .map(|process| {
            let pid = process.pid().as_u32();
            let user = process
                .user_id()
                .and_then(|uid| {
                    system
                        .users()
                        .iter()
                        .find(|user| user.id() == uid)
                        .map(|user| user.name().to_string())
                })
                .filter(|name| !name.is_empty());
            (
                pid,
                ProcessSnapshot {
                    process_guid: process_guid(config, process.start_time(), pid),
                    name: process.name().to_string_lossy().to_string(),
                    path: process
                        .exe()
                        .map(|path| path.display().to_string())
                        .filter(|value| !value.is_empty()),
                    user,
                },
            )
        })
        .collect()
}

#[cfg(any(windows, test))]
#[derive(Debug, Clone, PartialEq, Eq)]
struct NetworkFlowObservation {
    protocol: String,
    direction: String,
    local_ip: String,
    local_port: Option<u16>,
    remote_ip: String,
    remote_port: Option<u16>,
    pid: Option<u32>,
    connection_state: Option<String>,
}

#[cfg(any(windows, test))]
#[derive(Debug, Clone, PartialEq, Eq)]
struct DnsCacheObservation {
    query: String,
    query_type: Option<String>,
    answers: Vec<String>,
}

#[cfg(any(windows, test))]
fn parse_netstat_ano(output: &str) -> Vec<NetworkFlowObservation> {
    output
        .lines()
        .filter_map(parse_netstat_line)
        .filter(|flow| {
            flow.direction != "local"
                && flow.remote_ip != "0.0.0.0"
                && flow.remote_ip != "::"
                && flow.remote_ip != "*"
        })
        .collect()
}

#[cfg(any(windows, test))]
fn parse_netstat_line(line: &str) -> Option<NetworkFlowObservation> {
    let parts = line.split_whitespace().collect::<Vec<_>>();
    let protocol = parts.first()?.to_ascii_lowercase();
    if protocol != "tcp" && protocol != "udp" {
        return None;
    }

    let local = parse_socket_addr(parts.get(1)?)?;
    let remote = parse_socket_addr(parts.get(2)?)?;
    let (connection_state, pid_index) = if protocol == "tcp" {
        (parts.get(3).map(|value| value.to_string()), 4)
    } else {
        (None, 3)
    };
    if connection_state
        .as_deref()
        .is_some_and(|state| is_tcp_listen_state(state))
    {
        return None;
    }

    let pid = parts
        .get(pid_index)
        .and_then(|value| value.parse::<u32>().ok());
    let direction = infer_direction(&local.0, &remote.0);

    Some(NetworkFlowObservation {
        protocol,
        direction,
        local_ip: local.0,
        local_port: local.1,
        remote_ip: remote.0,
        remote_port: remote.1,
        pid,
        connection_state,
    })
}

#[cfg(any(windows, test))]
fn parse_socket_addr(value: &str) -> Option<(String, Option<u16>)> {
    if value == "*:*" || value == "*" {
        return Some(("*".to_string(), None));
    }

    let trimmed = value.trim();
    if let Some(rest) = trimmed.strip_prefix('[') {
        let (host, port) = rest.rsplit_once("]:")?;
        return Some((host.to_string(), parse_port(port)));
    }

    let (host, port) = trimmed.rsplit_once(':')?;
    Some((host.to_string(), parse_port(port)))
}

#[cfg(any(windows, test))]
fn parse_port(value: &str) -> Option<u16> {
    if value == "*" {
        None
    } else {
        value.parse::<u16>().ok()
    }
}

#[cfg(any(windows, test))]
fn infer_direction(local_ip: &str, remote_ip: &str) -> String {
    if remote_ip == "127.0.0.1" || remote_ip == "::1" || remote_ip.eq_ignore_ascii_case("localhost")
    {
        "local".to_string()
    } else if local_ip == "0.0.0.0" || local_ip == "::" {
        "inbound".to_string()
    } else {
        "outbound".to_string()
    }
}

#[cfg(windows)]
fn flow_id(config: &AgentConfig, flow: &NetworkFlowObservation) -> String {
    format!(
        "{}:{}:{}:{}:{}:{}:{}:{}",
        config.device_id,
        flow.pid
            .map(|pid| pid.to_string())
            .unwrap_or_else(|| "0".to_string()),
        flow.protocol,
        flow.local_ip,
        flow.local_port
            .map(|port| port.to_string())
            .unwrap_or_else(|| "0".to_string()),
        flow.remote_ip,
        flow.remote_port
            .map(|port| port.to_string())
            .unwrap_or_else(|| "0".to_string()),
        "netstat"
    )
}

#[cfg(any(windows, test))]
fn is_tcp_listen_state(state: &str) -> bool {
    state.eq_ignore_ascii_case("LISTENING")
}

#[cfg(windows)]
fn is_tcp_terminal_state(state: &str) -> bool {
    matches!(
        state.to_ascii_uppercase().as_str(),
        "TIME_WAIT"
            | "FIN_WAIT_1"
            | "FIN_WAIT_2"
            | "CLOSE_WAIT"
            | "CLOSING"
            | "LAST_ACK"
            | "CLOSED"
    )
}

#[cfg(any(windows, test))]
fn parse_ipconfig_displaydns(output: &str) -> Vec<DnsCacheObservation> {
    let mut observations = Vec::new();
    let mut current_name: Option<String> = None;
    let mut current_type: Option<String> = None;
    let mut answers: Vec<String> = Vec::new();

    for line in output.lines() {
        let trimmed = line.trim();
        if trimmed.is_empty() {
            flush_dns_observation(
                &mut observations,
                &mut current_name,
                &mut current_type,
                &mut answers,
            );
            continue;
        }

        if let Some(value) = value_after_colon(trimmed, "Record Name") {
            flush_dns_observation(
                &mut observations,
                &mut current_name,
                &mut current_type,
                &mut answers,
            );
            current_name = Some(value.trim_end_matches('.').to_string());
        } else if let Some(value) = value_after_colon(trimmed, "Record Type") {
            current_type = Some(record_type_name(value));
        } else if let Some(value) = value_after_colon(trimmed, "A (Host) Record") {
            answers.push(value.to_string());
        } else if let Some(value) = value_after_colon(trimmed, "AAAA Record") {
            answers.push(value.to_string());
        } else if let Some(value) = value_after_colon(trimmed, "CNAME Record") {
            answers.push(value.trim_end_matches('.').to_string());
        }
    }

    flush_dns_observation(
        &mut observations,
        &mut current_name,
        &mut current_type,
        &mut answers,
    );
    observations
}

#[cfg(any(windows, test))]
fn flush_dns_observation(
    observations: &mut Vec<DnsCacheObservation>,
    current_name: &mut Option<String>,
    current_type: &mut Option<String>,
    answers: &mut Vec<String>,
) {
    if let Some(query) = current_name.take() {
        if !answers.is_empty() {
            observations.push(DnsCacheObservation {
                query,
                query_type: current_type.take(),
                answers: std::mem::take(answers),
            });
            return;
        }
    }

    current_type.take();
    answers.clear();
}

#[cfg(any(windows, test))]
fn value_after_colon<'a>(line: &'a str, label: &str) -> Option<&'a str> {
    if !line.starts_with(label) {
        return None;
    }
    line.rsplit_once(':').map(|(_, value)| value.trim())
}

#[cfg(any(windows, test))]
fn record_type_name(value: &str) -> String {
    match value.trim() {
        "1" => "A".to_string(),
        "5" => "CNAME".to_string(),
        "28" => "AAAA".to_string(),
        other => other.to_string(),
    }
}

#[cfg(any(windows, test))]
fn host_from_url(url: &str) -> Option<String> {
    let (_, rest) = url.split_once("://")?;
    let authority = rest
        .split(['/', '?', '#'])
        .next()
        .unwrap_or("")
        .rsplit('@')
        .next()
        .unwrap_or("")
        .trim();
    if authority.is_empty() {
        return None;
    }

    if let Some(ipv6) = authority.strip_prefix('[') {
        let host = ipv6.split(']').next()?.to_ascii_lowercase();
        return (!host.is_empty()).then_some(host);
    }

    let host = authority
        .split(':')
        .next()
        .unwrap_or("")
        .trim()
        .trim_end_matches('.')
        .to_ascii_lowercase();
    if host.is_empty() || host == "localhost" {
        None
    } else {
        Some(host)
    }
}

#[cfg(test)]
mod tests {
    use std::path::PathBuf;
    use std::time::SystemTime;

    use crate::config::AgentConfig;
    use crate::event::{AegisEvent, EventPayload};

    use super::{
        correlate_dns_to_flow_attribution, enrich_flow_hostnames, host_from_url, match_sase_vendor,
        parse_extension_manifest_json, parse_ipconfig_displaydns, parse_netstat_ano,
        parse_sase_installed_products, parse_sase_network_adapters, parse_sase_proxy_config,
        parse_sase_services, parse_socket_addr,
    };

    #[test]
    fn parses_netstat_tcp_and_udp_observations() {
        let output = r#"
  Proto  Local Address          Foreign Address        State           PID
  TCP    10.10.20.55:52944      203.0.113.10:443      ESTABLISHED     3084
  TCP    0.0.0.0:135            0.0.0.0:0             LISTENING       944
  UDP    10.10.20.55:5353       *:*                                   1200
"#;

        let flows = parse_netstat_ano(output);

        assert_eq!(flows.len(), 1);
        assert_eq!(flows[0].protocol, "tcp");
        assert_eq!(flows[0].direction, "outbound");
        assert_eq!(flows[0].local_ip, "10.10.20.55");
        assert_eq!(flows[0].local_port, Some(52944));
        assert_eq!(flows[0].remote_ip, "203.0.113.10");
        assert_eq!(flows[0].remote_port, Some(443));
        assert_eq!(flows[0].pid, Some(3084));
        assert_eq!(flows[0].connection_state.as_deref(), Some("ESTABLISHED"));
    }

    #[test]
    fn parses_bracketed_ipv6_socket_address() {
        let parsed = parse_socket_addr("[fe80::1]:443").unwrap();

        assert_eq!(parsed.0, "fe80::1");
        assert_eq!(parsed.1, Some(443));
    }

    #[test]
    fn parses_ipconfig_displaydns_cache_records() {
        let output = r#"
Windows IP Configuration

    api.model-gateway.lab
    ----------------------------------------
    Record Name . . . . . : api.model-gateway.lab
    Record Type . . . . . : 1
    Time To Live  . . . . : 120
    Data Length . . . . . : 4
    Section . . . . . . . : Answer
    A (Host) Record . . . : 203.0.113.10

    model-cname.lab
    ----------------------------------------
    Record Name . . . . . : model-cname.lab.
    Record Type . . . . . : 5
    CNAME Record  . . . . : api.model-gateway.lab.
"#;

        let observations = parse_ipconfig_displaydns(output);

        assert_eq!(observations.len(), 2);
        assert_eq!(observations[0].query, "api.model-gateway.lab");
        assert_eq!(observations[0].query_type.as_deref(), Some("A"));
        assert_eq!(observations[0].answers, vec!["203.0.113.10"]);
        assert_eq!(observations[1].query, "model-cname.lab");
        assert_eq!(observations[1].query_type.as_deref(), Some("CNAME"));
        assert_eq!(observations[1].answers, vec!["api.model-gateway.lab"]);
    }

    #[test]
    fn extracts_hosts_from_browser_urls() {
        assert_eq!(
            host_from_url("https://chatgpt.com/c/abc"),
            Some("chatgpt.com".to_string())
        );
        assert_eq!(
            host_from_url("https://user@example.openai.com:443/path"),
            Some("example.openai.com".to_string())
        );
        assert_eq!(host_from_url("file:///C:/tmp/report.txt"), None);
    }

    #[test]
    fn enriches_flow_remote_hostname_from_dns_answer() {
        let config = test_config();
        let mut events = vec![
            AegisEvent::new(
                &config,
                "aegis.flow.started",
                SystemTime::now(),
                EventPayload::FlowStarted {
                    flow_id: "flow-1".to_string(),
                    process_guid: None,
                    pid: Some(1000),
                    process_name: Some("msedge.exe".to_string()),
                    process_path: None,
                    user: None,
                    protocol: "tcp".to_string(),
                    direction: "outbound".to_string(),
                    local_ip: "192.168.12.101".to_string(),
                    local_port: Some(50000),
                    remote_ip: "203.0.113.10".to_string(),
                    remote_port: Some(443),
                    remote_hostname: None,
                    connection_state: Some("ESTABLISHED".to_string()),
                    bytes_sent: None,
                    bytes_received: None,
                    attribution_method: "test".to_string(),
                    attribution_confidence: 0.7,
                },
            ),
            AegisEvent::new(
                &config,
                "aegis.dns.observed",
                SystemTime::now(),
                EventPayload::DnsObserved {
                    query: "chatgpt.com".to_string(),
                    query_type: Some("A".to_string()),
                    answers: vec!["203.0.113.10".to_string()],
                    resolver: None,
                    process_guid: None,
                    pid: None,
                    correlation_method: "test".to_string(),
                    correlation_confidence: 0.5,
                },
            ),
        ];

        enrich_flow_hostnames(&mut events);

        match &events[0].payload {
            EventPayload::FlowStarted {
                remote_hostname, ..
            } => assert_eq!(remote_hostname.as_deref(), Some("chatgpt.com")),
            _ => panic!("expected flow event"),
        }
    }

    #[test]
    fn correlates_dns_pid_from_flow_remote_ip() {
        let config = test_config();
        let mut events = vec![
            AegisEvent::new(
                &config,
                "aegis.flow.started",
                SystemTime::now(),
                EventPayload::FlowStarted {
                    flow_id: "flow-1".to_string(),
                    process_guid: Some("proc-net".to_string()),
                    pid: Some(9001),
                    process_name: Some("git.exe".to_string()),
                    process_path: None,
                    user: None,
                    protocol: "tcp".to_string(),
                    direction: "outbound".to_string(),
                    local_ip: "10.0.0.2".to_string(),
                    local_port: Some(52000),
                    remote_ip: "203.0.113.55".to_string(),
                    remote_port: Some(443),
                    remote_hostname: None,
                    connection_state: Some("ESTABLISHED".to_string()),
                    bytes_sent: None,
                    bytes_received: None,
                    attribution_method: "test".to_string(),
                    attribution_confidence: 0.8,
                },
            ),
            AegisEvent::new(
                &config,
                "aegis.dns.observed",
                SystemTime::now(),
                EventPayload::DnsObserved {
                    query: "github.com".to_string(),
                    query_type: Some("A".to_string()),
                    answers: vec!["203.0.113.55".to_string()],
                    resolver: None,
                    process_guid: None,
                    pid: None,
                    correlation_method: "windows.ipconfig_displaydns.cache".to_string(),
                    correlation_confidence: 0.35,
                },
            ),
        ];

        correlate_dns_to_flow_attribution(&mut events);

        match &events[1].payload {
            EventPayload::DnsObserved {
                pid,
                process_guid,
                correlation_method,
                ..
            } => {
                assert_eq!(*pid, Some(9001));
                assert_eq!(process_guid.as_deref(), Some("proc-net"));
                assert_eq!(correlation_method, "aegis.dns.flow_remote_ip_match");
            }
            _ => panic!("expected dns event"),
        }
    }

    #[test]
    fn parses_extension_manifest_inventory_fields() {
        let manifest = r#"{
            "manifest_version": 3,
            "name": "AI Security Extension",
            "version": "1.2.3",
            "permissions": ["storage", "tabs", "https://chatgpt.com/*"],
            "host_permissions": ["https://*.openai.com/*", "<all_urls>"]
        }"#;

        let parsed = parse_extension_manifest_json(manifest).unwrap();

        assert_eq!(parsed.name, "AI Security Extension");
        assert_eq!(parsed.version, "1.2.3");
        assert_eq!(parsed.manifest_version, Some(3));
        assert!(parsed.permissions.contains(&"tabs".to_string()));
        assert!(parsed.host_permissions.contains(&"<all_urls>".to_string()));
        assert!(parsed
            .host_permissions
            .contains(&"https://chatgpt.com/*".to_string()));
    }

    #[test]
    fn matches_major_sase_vendor_signatures() {
        let zscaler = match_sase_vendor("Zscaler Client Connector");
        assert_eq!(zscaler.map(|signature| signature.vendor), Some("Zscaler"));

        let prisma = match_sase_vendor("Palo Alto Networks GlobalProtect");
        assert_eq!(
            prisma.map(|signature| signature.product),
            Some("GlobalProtect / Prisma Access")
        );

        let cisco = match_sase_vendor("Cisco Secure Client Umbrella Module");
        assert_eq!(cisco.map(|signature| signature.vendor), Some("Cisco"));
    }

    #[test]
    fn parses_sase_installed_product_inventory() {
        let output = "Zscaler Client Connector\t4.3.0\tZscaler, Inc.\tC:\\Program Files\\Zscaler\nNotepad++\t8.6\tNotepad++ Team\tC:\\Program Files\\Notepad++\n";

        let observations = parse_sase_installed_products(output);

        assert_eq!(observations.len(), 1);
        assert_eq!(observations[0].component_type, "installed_product");
        assert_eq!(observations[0].vendor, "Zscaler");
        assert_eq!(observations[0].version.as_deref(), Some("4.3.0"));
    }

    #[test]
    fn parses_sase_service_inventory() {
        let output = "PanGPS\tPalo Alto Networks GlobalProtect Service\tRunning\nSpooler\tPrint Spooler\tRunning\n";

        let observations = parse_sase_services(output);

        assert_eq!(observations.len(), 1);
        assert_eq!(observations[0].component_type, "service");
        assert_eq!(observations[0].vendor, "Palo Alto Networks");
        assert_eq!(observations[0].status.as_deref(), Some("Running"));
    }

    #[test]
    fn parses_sase_network_adapter_inventory() {
        let output =
            "Cloudflare WARP\tCloudflare WARP Interface Tunnel\tUp\nEthernet\tIntel Adapter\tUp\n";

        let observations = parse_sase_network_adapters(output);

        assert_eq!(observations.len(), 1);
        assert_eq!(observations[0].component_type, "network_adapter");
        assert_eq!(observations[0].vendor, "Cloudflare");
    }

    #[test]
    fn parses_proxy_config_inventory_with_unknown_vendor() {
        let output = "ProxyEnable\t1\nProxyServer\thttp=proxy.enterprise.example:8080\nAutoConfigURL\thttps://proxy.enterprise.example/proxy.pac\n";

        let observations =
            parse_sase_proxy_config("windows.user_proxy", "windows.internet_settings", output);

        assert_eq!(observations.len(), 1);
        assert_eq!(observations[0].component_type, "proxy_config");
        assert_eq!(observations[0].vendor, "unknown");
        assert_eq!(observations[0].status.as_deref(), Some("configured"));
    }

    #[test]
    fn ignores_empty_proxy_config_inventory() {
        let output = "ProxyEnable\t0\nProxyServer\t\nAutoConfigURL\t\nAutoDetect\t";

        let observations =
            parse_sase_proxy_config("windows.user_proxy", "windows.internet_settings", output);

        assert!(observations.is_empty());
    }

    fn test_config() -> AgentConfig {
        AgentConfig {
            agent_id: "test-agent".to_string(),
            device_id: "test-device".to_string(),
            sensor_version: "test".to_string(),
            backend_url: None,
            event_spool: PathBuf::from("/tmp/aegis-test.jsonl"),
            process_state_path: PathBuf::from("/tmp/aegis-test.jsonl.process-state.json"),
            collect_command_line: false,
            controller_url: None,
            detection_packs_enabled: false,
            detection_pack_cache: None,
            detection_pack_public_key: None,
            process_snapshot_limit: 256,
            visibility_post_chunk_size: 500,
        }
    }
}

fn sanitize_command_line(value: &str) -> String {
    const MAX_COMMAND_LINE_LEN: usize = 2048;
    let sensitive_markers = [
        "TOKEN",
        "SECRET",
        "PASSWORD",
        "PASSWD",
        "API_KEY",
        "ACCESS_KEY",
        "PRIVATE_KEY",
        "AUTH",
        "COOKIE",
    ];

    let mut sanitized = value
        .split_whitespace()
        .map(|part| {
            let upper = part.to_ascii_uppercase();
            if sensitive_markers
                .iter()
                .any(|marker| upper.contains(marker))
            {
                "[REDACTED]".to_string()
            } else {
                part.to_string()
            }
        })
        .collect::<Vec<_>>()
        .join(" ");

    if sanitized.len() > MAX_COMMAND_LINE_LEN {
        sanitized.truncate(MAX_COMMAND_LINE_LEN);
        sanitized.push_str("...[truncated]");
    }

    sanitized
}
