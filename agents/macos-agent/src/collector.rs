//! Visibility collectors.

use std::process::Command;
use std::time::{SystemTime, UNIX_EPOCH};

use crate::config::AgentConfig;
use crate::event::{AegisEvent, EventPayload};

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
        "macos.process"
    }

    fn collect_once(&self, config: &AgentConfig) -> Result<Vec<AegisEvent>, String> {
        let snapshot_ms = current_timestamp_ms();
        let processes = collect_process_snapshot(config)?;
        let mut events = Vec::with_capacity(processes.len() + 1);
        events.push(collector_status(
            config,
            self.name(),
            "healthy",
            snapshot_message(processes.len()),
        ));

        for process in processes {
            let process_instance_id = process_guid(config, snapshot_ms, process.pid);
            let parent_process_guid = process
                .ppid
                .map(|ppid| process_guid(config, snapshot_ms, ppid));

            events.push(AegisEvent::new(
                config,
                "aegis.process.started",
                SystemTime::now(),
                EventPayload::ProcessStarted {
                    process_guid: process_instance_id,
                    parent_process_guid,
                    pid: process.pid,
                    ppid: process.ppid,
                    name: process.name,
                    path: process.path,
                    command_line: process.command_line,
                    user: None,
                    logon_session_id: None,
                    integrity_level: None,
                    sha256: None,
                    publisher: None,
                    collection_method: "ps.snapshot".to_string(),
                },
            ));
        }

        Ok(events)
    }
}

impl Collector for NetworkCollector {
    fn name(&self) -> &'static str {
        "macos.network"
    }

    fn collect_once(&self, config: &AgentConfig) -> Result<Vec<AegisEvent>, String> {
        Ok(vec![collector_status(
            config,
            self.name(),
            "pending",
            macos_only_message("network attribution collector requires approved macOS API design"),
        )])
    }
}

impl Collector for DnsCollector {
    fn name(&self) -> &'static str {
        "macos.dns"
    }

    fn collect_once(&self, config: &AgentConfig) -> Result<Vec<AegisEvent>, String> {
        Ok(vec![collector_status(
            config,
            self.name(),
            "pending",
            macos_only_message("DNS observation collector requires approved macOS API design"),
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

#[cfg(target_os = "macos")]
fn macos_only_message(message: &str) -> String {
    message.to_string()
}

#[cfg(not(target_os = "macos"))]
fn macos_only_message(message: &str) -> String {
    format!("{message}; running on non-macOS development host")
}

#[derive(Debug)]
struct ProcessSnapshotRecord {
    pid: u32,
    ppid: Option<u32>,
    name: String,
    path: Option<String>,
    command_line: Option<String>,
}

fn collect_process_snapshot(config: &AgentConfig) -> Result<Vec<ProcessSnapshotRecord>, String> {
    let process_fields = if config.collect_command_line {
        "pid=,ppid=,command="
    } else {
        "pid=,ppid=,comm="
    };
    let output = Command::new("/bin/ps")
        .args(["-axo", process_fields])
        .output()
        .map_err(|err| format!("failed to run /bin/ps process snapshot: {err}"))?;

    if !output.status.success() {
        return Err(format!(
            "/bin/ps process snapshot failed with status {}",
            output.status
        ));
    }

    let stdout = String::from_utf8(output.stdout)
        .map_err(|err| format!("/bin/ps emitted non-UTF-8 output: {err}"))?;

    let mut records = Vec::new();
    for line in stdout.lines() {
        if records.len() >= config.process_snapshot_limit {
            break;
        }
        if let Some(record) = parse_ps_line(line, config.collect_command_line) {
            records.push(record);
        }
    }

    Ok(records)
}

fn parse_ps_line(line: &str, collect_command_line: bool) -> Option<ProcessSnapshotRecord> {
    let (pid_raw, remaining) = take_leading_token(line.trim_start())?;
    let (ppid_raw, remaining) = take_leading_token(remaining.trim_start())?;
    let pid = pid_raw.parse::<u32>().ok()?;
    let ppid = ppid_raw.parse::<u32>().ok();
    let observed = remaining.trim().to_string();
    if observed.is_empty() {
        return None;
    }

    let (path, command_line) = if collect_command_line {
        let executable = observed
            .split_whitespace()
            .next()
            .map(|value| value.to_string());
        (executable, Some(sanitize_command_line(&observed)))
    } else {
        (Some(observed), None)
    };
    let name = path
        .as_deref()
        .map(process_name)
        .unwrap_or_else(|| "unknown".to_string());

    Some(ProcessSnapshotRecord {
        pid,
        ppid,
        name,
        path,
        command_line,
    })
}

fn take_leading_token(value: &str) -> Option<(&str, &str)> {
    let token_end = value
        .char_indices()
        .find_map(|(idx, ch)| if ch.is_whitespace() { Some(idx) } else { None })?;
    Some((&value[..token_end], &value[token_end..]))
}

fn process_name(path: &str) -> String {
    path.rsplit('/').next().unwrap_or(path).to_string()
}

fn process_guid(config: &AgentConfig, snapshot_ms: u128, pid: u32) -> String {
    format!("{}:{}:{}", config.device_id, snapshot_ms, pid)
}

fn current_timestamp_ms() -> u128 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|duration| duration.as_millis())
        .unwrap_or(0)
}

fn snapshot_message(count: usize) -> String {
    let suffix = if cfg!(target_os = "macos") {
        ""
    } else {
        "; running on non-macOS development host"
    };
    format!("process snapshot collector observed {count} processes{suffix}")
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
    use super::{parse_ps_line, process_name, take_leading_token};

    #[test]
    fn parses_process_snapshot_line_without_command_line() {
        let record = parse_ps_line(
            "  123   1 /Applications/App Name.app/Contents/MacOS/App Name",
            false,
        )
        .unwrap();
        assert_eq!(record.pid, 123);
        assert_eq!(record.ppid, Some(1));
        assert_eq!(record.name, "App Name");
        assert_eq!(
            record.path.as_deref(),
            Some("/Applications/App Name.app/Contents/MacOS/App Name")
        );
        assert!(record.command_line.is_none());
    }

    #[test]
    fn parses_leading_token_with_variable_spacing() {
        assert_eq!(
            take_leading_token("123   1 /bin/zsh"),
            Some(("123", "   1 /bin/zsh"))
        );
    }

    #[test]
    fn parses_process_snapshot_line_with_command_line() {
        let record = parse_ps_line("  123   1 /usr/bin/zsh zsh -l", true).unwrap();
        assert_eq!(record.path.as_deref(), Some("/usr/bin/zsh"));
        assert_eq!(record.command_line.as_deref(), Some("/usr/bin/zsh zsh -l"));
    }

    #[test]
    fn extracts_process_name_from_path() {
        assert_eq!(
            process_name("/Applications/App.app/Contents/MacOS/App"),
            "App"
        );
    }
}
