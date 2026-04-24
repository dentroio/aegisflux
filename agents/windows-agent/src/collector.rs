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

#[cfg(windows)]
fn windows_only_message(message: &str) -> String {
    message.to_string()
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
