//! Process `started` / `ended` events via snapshot diff (WO-VIS-002 lifecycle path without ETW).

use std::collections::BTreeMap;
use std::fs;
use std::path::Path;
use std::time::{SystemTime, UNIX_EPOCH};

use serde::{Deserialize, Serialize};
use sysinfo::{ProcessRefreshKind, ProcessesToUpdate, System, UpdateKind};

use crate::collector::command_line;
use crate::config::AgentConfig;
use crate::event::{AegisEvent, EventPayload};

#[derive(Debug, Default, Serialize, Deserialize)]
struct ProcessStateFile {
    #[serde(default)]
    processes: BTreeMap<String, ProcessStateEntry>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct ProcessStateEntry {
    pid: u32,
    name: String,
    first_seen_ms: u128,
}

fn process_guid(config: &AgentConfig, start_time_secs: u64, pid: u32) -> String {
    format!("{}:{}:{}", config.device_id, start_time_secs, pid)
}

fn now_ms() -> u128 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|duration| duration.as_millis())
        .unwrap_or(0)
}

fn lifecycle_status(config: &AgentConfig, message: String) -> AegisEvent {
    AegisEvent::new(
        config,
        "aegis.collector.status",
        SystemTime::now(),
        EventPayload::CollectorStatus {
            collector: "windows.process_lifecycle".to_string(),
            status: "healthy".to_string(),
            message,
        },
    )
}

/// Emit process lifecycle events by diffing the current process snapshot against persisted state.
///
/// First run (missing or empty state file): writes current snapshot and returns a single status
/// event (no per-process spam). Subsequent runs emit [`EventPayload::ProcessStarted`] /
/// [`EventPayload::ProcessEnded`] for true arrivals and departures.
pub(crate) fn collect_process_lifecycle(
    config: &AgentConfig,
    state_path: &Path,
) -> Result<Vec<AegisEvent>, String> {
    let mut system = System::new();
    #[cfg(windows)]
    system.refresh_users_list();
    system.refresh_processes_specifics(
        ProcessesToUpdate::All,
        true,
        ProcessRefreshKind::nothing()
            .with_cmd(UpdateKind::Always)
            .with_exe(UpdateKind::Always),
    );

    let ts_ms = now_ms();
    let mut current_entries: BTreeMap<String, ProcessStateEntry> = BTreeMap::new();
    let mut started_payloads: BTreeMap<String, AegisEvent> = BTreeMap::new();

    for process in system.processes().values() {
        let pid = process.pid().as_u32();
        let start_secs = process.start_time();
        let guid = process_guid(config, start_secs, pid);
        let ppid = process.parent().map(|parent| parent.as_u32());
        let parent_process_guid = process.parent().and_then(|parent_pid| {
            system
                .process(parent_pid)
                .map(|parent| process_guid(config, parent.start_time(), parent_pid.as_u32()))
        });

        let name = process.name().to_string_lossy().to_string();
        let path = process
            .exe()
            .map(|path| path.display().to_string())
            .filter(|value| !value.is_empty());
        let command_line = command_line(process.cmd(), config.collect_command_line);
        let user: Option<String> = {
            #[cfg(windows)]
            {
                process
                    .user_id()
                    .and_then(|uid| {
                        system
                            .users()
                            .iter()
                            .find(|user| user.id() == uid)
                            .map(|user| user.name().to_string())
                    })
                    .filter(|value| !value.is_empty())
            }
            #[cfg(not(windows))]
            {
                None
            }
        };

        current_entries.insert(
            guid.clone(),
            ProcessStateEntry {
                pid,
                name: name.clone(),
                first_seen_ms: ts_ms,
            },
        );

        started_payloads.insert(
            guid.clone(),
            AegisEvent::new(
                config,
                "aegis.process.started",
                SystemTime::now(),
                EventPayload::ProcessStarted {
                    process_guid: guid.clone(),
                    parent_process_guid,
                    pid,
                    ppid,
                    name,
                    path,
                    command_line,
                    user,
                    logon_session_id: None,
                    integrity_level: None,
                    sha256: None,
                    publisher: None,
                    collection_method: "sysinfo.lifecycle_diff".to_string(),
                },
            ),
        );
    }

    let prev: ProcessStateFile = if state_path.exists() {
        let raw = fs::read_to_string(state_path).map_err(|err| {
            format!(
                "failed to read process state {}: {err}",
                state_path.display()
            )
        })?;
        if raw.trim().is_empty() {
            ProcessStateFile::default()
        } else {
            serde_json::from_str(&raw).unwrap_or_default()
        }
    } else {
        ProcessStateFile::default()
    };

    let mut out: Vec<AegisEvent> = Vec::new();

    if prev.processes.is_empty() {
        let next = merge_persisted_first_seen(&current_entries, &prev, ts_ms);
        persist_state(state_path, &next)?;
        out.push(lifecycle_status(
            config,
            format!(
                "process lifecycle seeded {} processes; next collection will emit start/end diffs",
                next.processes.len()
            ),
        ));
        return Ok(out);
    }

    for (guid, event) in &started_payloads {
        if !prev.processes.contains_key(guid.as_str()) {
            out.push(event.clone());
        }
    }

    for (guid, prev_entry) in &prev.processes {
        if !current_entries.contains_key(guid.as_str()) {
            let duration_ms = ts_ms.saturating_sub(prev_entry.first_seen_ms);
            let duration_ms_u64 = u64::try_from(duration_ms).ok();
            out.push(AegisEvent::new(
                config,
                "aegis.process.ended",
                SystemTime::now(),
                EventPayload::ProcessEnded {
                    process_guid: guid.clone(),
                    pid: prev_entry.pid,
                    name: Some(prev_entry.name.clone()),
                    exit_code: None,
                    duration_ms: duration_ms_u64,
                    collection_method: "sysinfo.lifecycle_diff".to_string(),
                },
            ));
        }
    }

    let next = merge_persisted_first_seen(&current_entries, &prev, ts_ms);
    persist_state(state_path, &next)?;

    out.insert(
        0,
        lifecycle_status(
            config,
            format!(
                "process lifecycle diff: {} started, {} ended in this interval",
                out.iter()
                    .filter(|event| event.event_type == "aegis.process.started")
                    .count(),
                out.iter()
                    .filter(|event| event.event_type == "aegis.process.ended")
                    .count()
            ),
        ),
    );

    Ok(out)
}

fn merge_persisted_first_seen(
    current: &BTreeMap<String, ProcessStateEntry>,
    prev: &ProcessStateFile,
    ts_ms: u128,
) -> ProcessStateFile {
    let mut out = ProcessStateFile::default();
    for (guid, cur) in current {
        let first_seen_ms = prev
            .processes
            .get(guid)
            .map(|entry| entry.first_seen_ms)
            .unwrap_or(ts_ms);
        out.processes.insert(
            guid.clone(),
            ProcessStateEntry {
                pid: cur.pid,
                name: cur.name.clone(),
                first_seen_ms,
            },
        );
    }
    out
}

fn persist_state(path: &Path, state: &ProcessStateFile) -> Result<(), String> {
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent)
            .map_err(|err| format!("failed to create process state dir: {err}"))?;
    }
    let raw = serde_json::to_string_pretty(state)
        .map_err(|err| format!("failed to serialize process state: {err}"))?;
    fs::write(path, raw).map_err(|err| format!("failed to write process state: {err}"))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::AgentConfig;
    use std::path::PathBuf;

    fn test_config() -> AgentConfig {
        AgentConfig {
            agent_id: "a1".to_string(),
            device_id: "d1".to_string(),
            sensor_version: "0.1.0".to_string(),
            backend_url: None,
            event_spool: PathBuf::from("/tmp/aegis-lc.jsonl"),
            process_state_path: PathBuf::from("/tmp/aegis-lc.jsonl.process-state.json"),
            collect_command_line: false,
            controller_url: None,
            detection_packs_enabled: false,
            detection_pack_cache: None,
            detection_pack_public_key: None,
            process_snapshot_limit: 256,
            visibility_post_chunk_size: 500,
        }
    }

    #[test]
    fn seeds_without_process_events_when_no_prior_state() {
        let dir = std::env::temp_dir().join(format!("aegis-lc-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        fs::create_dir_all(&dir).expect("mkdir");
        let state = dir.join("st.json");
        let config = test_config();
        let events = collect_process_lifecycle(&config, &state).expect("lifecycle");
        assert!(state.exists());
        assert!(
            events
                .iter()
                .all(|event| event.event_type != "aegis.process.started"),
            "seed run should not emit per-process starts"
        );
        let status = events.iter().find(|event| {
            matches!(
                &event.payload,
                EventPayload::CollectorStatus { collector, .. } if collector == "windows.process_lifecycle"
            )
        });
        assert!(status.is_some());
        let _ = fs::remove_file(&state);
    }
}
