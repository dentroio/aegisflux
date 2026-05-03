//! Linux visibility collectors.

use std::collections::HashMap;
use std::fs;
use std::process::Command;
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
        "linux.process"
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
            format!(
                "process snapshot collector observed {} processes",
                system.processes().len()
            ),
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
                    command_line: command_line(process.cmd(), config.collect_command_line),
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
        "linux.network"
    }

    fn collect_once(&self, config: &AgentConfig) -> Result<Vec<AegisEvent>, String> {
        let output = Command::new("ss")
            .args(["-tunap"])
            .output()
            .map_err(|err| format!("failed to run ss -tunap: {err}"))?;

        if !output.status.success() {
            return Ok(vec![collector_status(
                config,
                self.name(),
                "degraded",
                format!("ss -tunap failed with status {}", output.status),
            )]);
        }

        let stdout = String::from_utf8_lossy(&output.stdout);
        let flows = parse_ss_tunap(&stdout);
        let process_map = process_snapshot_map(config);
        let mut events = vec![collector_status(
            config,
            self.name(),
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
                    attribution_method: "linux.ss.pid".to_string(),
                    attribution_confidence: if flow.pid.is_some() { 0.72 } else { 0.35 },
                },
            ));
        }

        Ok(events)
    }
}

impl Collector for DnsCollector {
    fn name(&self) -> &'static str {
        "linux.dns"
    }

    fn collect_once(&self, config: &AgentConfig) -> Result<Vec<AegisEvent>, String> {
        let resolv_conf = fs::read_to_string("/etc/resolv.conf")
            .map_err(|err| format!("failed to read /etc/resolv.conf: {err}"))?;
        let observations = parse_resolv_conf(&resolv_conf);
        let mut events = vec![collector_status(
            config,
            self.name(),
            "healthy",
            format!(
                "resolver snapshot collector observed {} nameservers",
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
                    query_type: None,
                    answers: observation.answers,
                    resolver: observation.resolver,
                    process_guid: None,
                    pid: None,
                    correlation_method: "linux.resolv_conf.nameserver".to_string(),
                    correlation_confidence: 0.25,
                },
            ));
        }

        Ok(events)
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

#[derive(Debug, Clone)]
struct ProcessSnapshot {
    process_guid: String,
    name: String,
}

fn process_snapshot_map(config: &AgentConfig) -> HashMap<u32, ProcessSnapshot> {
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

#[derive(Debug, Clone, PartialEq, Eq)]
struct DnsObservation {
    query: String,
    answers: Vec<String>,
    resolver: Option<String>,
}

fn parse_ss_tunap(output: &str) -> Vec<NetworkFlowObservation> {
    output
        .lines()
        .filter_map(parse_ss_line)
        .filter(|flow| {
            flow.direction != "local"
                && flow.remote_ip != "0.0.0.0"
                && flow.remote_ip != "::"
                && flow.remote_ip != "*"
        })
        .collect()
}

fn parse_ss_line(line: &str) -> Option<NetworkFlowObservation> {
    let parts = line.split_whitespace().collect::<Vec<_>>();
    let netid = parts.first()?.to_ascii_lowercase();
    if netid != "tcp" && netid != "udp" {
        return None;
    }

    let state = parts.get(1).copied().unwrap_or_default();
    if state.eq_ignore_ascii_case("LISTEN") || state.eq_ignore_ascii_case("UNCONN") {
        return None;
    }

    let local = parse_socket_addr(parts.get(4)?)?;
    let remote = parse_socket_addr(parts.get(5)?)?;
    let pid = parse_pid(line);

    Some(NetworkFlowObservation {
        protocol: netid,
        direction: infer_direction(&local.0, &remote.0),
        local_ip: local.0,
        local_port: local.1,
        remote_ip: remote.0,
        remote_port: remote.1,
        pid,
    })
}

fn parse_socket_addr(value: &str) -> Option<(String, Option<u16>)> {
    let trimmed = value.trim();
    if trimmed == "*:*" || trimmed == "*" {
        return Some(("*".to_string(), None));
    }

    if let Some(rest) = trimmed.strip_prefix('[') {
        let (host, port) = rest.rsplit_once("]:")?;
        return Some((normalize_ip(host), parse_port(port)));
    }

    let (host, port) = trimmed.rsplit_once(':')?;
    Some((normalize_ip(host), parse_port(port)))
}

fn parse_port(value: &str) -> Option<u16> {
    if value == "*" {
        None
    } else {
        value.parse::<u16>().ok()
    }
}

fn normalize_ip(value: &str) -> String {
    value
        .trim_start_matches("[::ffff:")
        .trim_end_matches(']')
        .to_string()
}

fn parse_pid(line: &str) -> Option<u32> {
    let (_, rest) = line.split_once("pid=")?;
    rest.split(|ch: char| !ch.is_ascii_digit())
        .next()
        .and_then(|value| value.parse::<u32>().ok())
}

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

fn parse_resolv_conf(input: &str) -> Vec<DnsObservation> {
    input
        .lines()
        .filter_map(|line| {
            let trimmed = line.trim();
            if trimmed.starts_with('#') {
                return None;
            }
            let mut parts = trimmed.split_whitespace();
            if parts.next()? != "nameserver" {
                return None;
            }
            let resolver = parts.next()?.to_string();
            Some(DnsObservation {
                query: "_linux_resolver".to_string(),
                answers: vec![resolver.clone()],
                resolver: Some(resolver),
            })
        })
        .collect()
}

fn process_guid(config: &AgentConfig, start_time_secs: u64, pid: u32) -> String {
    format!("{}:{}:{}", config.device_id, start_time_secs, pid)
}

fn command_line(parts: &[std::ffi::OsString], enabled: bool) -> Option<String> {
    if !enabled || parts.is_empty() {
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

#[cfg(test)]
mod tests {
    use super::{parse_resolv_conf, parse_socket_addr, parse_ss_tunap};

    #[test]
    fn parses_ss_established_observation() {
        let output = r#"
Netid State Recv-Q Send-Q Local Address:Port Peer Address:Port Process
tcp ESTAB 0 0 192.168.101.31:52344 192.0.2.10:443 users:(("curl",pid=2311,fd=7))
tcp LISTEN 0 4096 0.0.0.0:22 0.0.0.0:* users:(("sshd",pid=730,fd=3))
"#;

        let flows = parse_ss_tunap(output);

        assert_eq!(flows.len(), 1);
        assert_eq!(flows[0].protocol, "tcp");
        assert_eq!(flows[0].direction, "outbound");
        assert_eq!(flows[0].local_ip, "192.168.101.31");
        assert_eq!(flows[0].local_port, Some(52344));
        assert_eq!(flows[0].remote_ip, "192.0.2.10");
        assert_eq!(flows[0].remote_port, Some(443));
        assert_eq!(flows[0].pid, Some(2311));
    }

    #[test]
    fn parses_bracketed_ipv6_socket_address() {
        let parsed = parse_socket_addr("[fe80::1]:443").unwrap();

        assert_eq!(parsed.0, "fe80::1");
        assert_eq!(parsed.1, Some(443));
    }

    #[test]
    fn parses_resolv_conf_nameservers() {
        let observations = parse_resolv_conf(
            r#"
# generated
nameserver 192.168.101.1
search netlab.net
nameserver 1.1.1.1
"#,
        );

        assert_eq!(observations.len(), 2);
        assert_eq!(observations[0].query, "_linux_resolver");
        assert_eq!(observations[0].answers, vec!["192.168.101.1"]);
        assert_eq!(observations[0].resolver.as_deref(), Some("192.168.101.1"));
    }
}
