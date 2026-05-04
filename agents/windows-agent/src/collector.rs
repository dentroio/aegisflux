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
}

impl CollectorSet {
    /// Return collectors in deterministic order.
    pub fn collectors(&self) -> Vec<&dyn Collector> {
        vec![
            &self.process,
            &self.network,
            &self.dns,
            &self.browser_history,
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

        for process in system.processes().values() {
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

fn command_line(parts: &[std::ffi::OsString], enabled: bool) -> Option<String> {
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
        events.push(AegisEvent::new(
            config,
            "aegis.flow.started",
            SystemTime::now(),
            EventPayload::FlowStarted {
                flow_id: flow_id(config, &flow),
                process_guid: process.as_ref().map(|process| process.process_guid.clone()),
                pid: flow.pid,
                process_name: process.map(|process| process.name),
                user: None,
                protocol: flow.protocol,
                direction: flow.direction,
                local_ip: flow.local_ip,
                local_port: flow.local_port,
                remote_ip: flow.remote_ip,
                remote_port: flow.remote_port,
                remote_hostname: None,
                attribution_method: "windows.netstat_ano.pid".to_string(),
                attribution_confidence: if flow.pid.is_some() { 0.72 } else { 0.35 },
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
#[derive(Debug, Clone)]
struct BrowserHistoryProfile {
    browser: String,
    history_path: std::path::PathBuf,
}

#[cfg(windows)]
fn browser_history_paths() -> Vec<BrowserHistoryProfile> {
    let mut profiles = Vec::new();
    for local_app_data in browser_local_app_data_roots() {
        let browser_roots = [
            ("edge", local_app_data.join(r"Microsoft\Edge\User Data")),
            ("chrome", local_app_data.join(r"Google\Chrome\User Data")),
            (
                "brave",
                local_app_data.join(r"BraveSoftware\Brave-Browser\User Data"),
            ),
        ];

        for (browser, root) in browser_roots {
            collect_browser_profiles(browser, &root, &mut profiles);
        }
    }

    profiles
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
    root: &std::path::Path,
    profiles: &mut Vec<BrowserHistoryProfile>,
) {
    if !root.exists() {
        return;
    }

    for profile_name in ["Default", "Profile 1", "Profile 2", "Profile 3"] {
        let history_path = root.join(profile_name).join("History");
        profiles.push(BrowserHistoryProfile {
            browser: format!("{browser}:{profile_name}"),
            history_path,
        });
    }

    if let Ok(entries) = std::fs::read_dir(root) {
        for entry in entries.flatten() {
            let file_name = entry.file_name().to_string_lossy().to_string();
            if !file_name.starts_with("Profile ") {
                continue;
            }
            let history_path = entry.path().join("History");
            profiles.push(BrowserHistoryProfile {
                browser: format!("{browser}:{file_name}"),
                history_path,
            });
        }
    }
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
}

#[cfg(windows)]
fn process_snapshot_map(config: &AgentConfig) -> std::collections::HashMap<u32, ProcessSnapshot> {
    let mut system = System::new();
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
            (
                pid,
                ProcessSnapshot {
                    process_guid: process_guid(config, process.start_time(), pid),
                    name: process.name().to_string_lossy().to_string(),
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
    let (state, pid_index) = if protocol == "tcp" {
        (parts.get(3).copied(), 4)
    } else {
        (None, 3)
    };
    if matches!(state, Some("LISTENING")) {
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
            .unwrap_or_else(|| "unknown-pid".to_string()),
        flow.protocol,
        flow.local_ip,
        flow.local_port
            .map(|port| port.to_string())
            .unwrap_or_else(|| "any".to_string()),
        flow.remote_ip,
        flow.remote_port
            .map(|port| port.to_string())
            .unwrap_or_else(|| "any".to_string()),
        "snapshot"
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
    use super::{host_from_url, parse_ipconfig_displaydns, parse_netstat_ano, parse_socket_addr};

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
