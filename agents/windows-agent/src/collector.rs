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
}

impl CollectorSet {
    /// Return collectors in deterministic order.
    pub fn collectors(&self) -> Vec<&dyn Collector> {
        vec![&self.process, &self.network, &self.dns]
    }
}

#[derive(Debug, Default)]
struct ProcessCollector;

#[derive(Debug, Default)]
struct NetworkCollector;

#[derive(Debug, Default)]
struct DnsCollector;

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

#[cfg(test)]
mod tests {
    use super::{parse_ipconfig_displaydns, parse_netstat_ano, parse_socket_addr};

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
