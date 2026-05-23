#![forbid(unsafe_code)]
#![warn(missing_docs)]

//! Aegis Linux visibility agent.

mod collector;
mod config;
mod detection;
mod dynamic_pack;
mod event;
mod security;
mod supervisor;
mod transport;

use std::env;
use std::fs;
use std::net::UdpSocket;
use std::process::ExitCode;
use std::time::{Instant, SystemTime, UNIX_EPOCH};

use collector::CollectorSet;
use config::AgentConfig;
use event::{AegisEvent, EventPayload};
use sysinfo::{get_current_pid, ProcessRefreshKind, ProcessesToUpdate, System};
use transport::{HttpJsonTransport, HttpVisibilityTransport, JsonlSpool};

fn main() -> ExitCode {
    match run() {
        Ok(()) => ExitCode::SUCCESS,
        Err(err) => {
            eprintln!("aegis-linux-agent error: {err}");
            ExitCode::FAILURE
        }
    }
}

fn run() -> Result<(), String> {
    let args = Args::parse(env::args().skip(1));
    let config = AgentConfig::from_env()?;
    security::validate_startup_posture(&config)?;
    if args.once {
        collect_and_emit(&config, &args)?;
        return Ok(());
    }

    supervisor::notify_systemd("READY=1\nSTATUS=aegis-linux-agent running");
    loop {
        if let Err(err) = collect_and_emit(&config, &args) {
            supervisor::notify_systemd(&format!(
                "STATUS=aegis-linux-agent collection failed: {err}"
            ));
            return Err(err);
        }
        supervisor::notify_systemd("WATCHDOG=1\nSTATUS=aegis-linux-agent collection complete");
        supervisor::sleep_with_watchdog(config.collection_interval);
    }
}

fn collect_and_emit(config: &AgentConfig, args: &Args) -> Result<(), String> {
    let mut events = Vec::new();
    events.push(AegisEvent::new(
        config,
        "aegis.agent.heartbeat",
        SystemTime::now(),
        EventPayload::Heartbeat {
            status: "starting".to_string(),
            message: "linux visibility agent initialized".to_string(),
            os: env::consts::OS.to_string(),
            arch: env::consts::ARCH.to_string(),
        },
    ));

    let collectors = CollectorSet::default();
    for collector in collectors.collectors() {
        let started = Instant::now();
        events.extend(collector.collect_once(config)?);
        let runtime_ms = millis_u64(started.elapsed().as_millis());
        events.push(performance_event(
            config,
            collector.name(),
            runtime_ms,
            None,
            events.len() as u64,
        ));
    }
    let visibility_for_packs: Vec<AegisEvent> = events.clone();
    events.extend(detection::detect_ai_agent_activity(
        config,
        &visibility_for_packs,
    ));
    let pack_started = Instant::now();
    events.extend(dynamic_pack::run_dynamic_pack_pipeline(
        config,
        &visibility_for_packs,
    ));
    events.push(performance_event(
        config,
        "dynamic_pack",
        0,
        Some(millis_u64(pack_started.elapsed().as_millis())),
        events.len() as u64,
    ));

    let spool = JsonlSpool::new(config.event_spool.clone());
    for event in &events {
        spool.append(event)?;
        if args.stdout {
            println!("{}", event.to_json());
        }
    }

    if let Some(backend_url) = &config.backend_url {
        let transport = HttpVisibilityTransport::new(backend_url)?;
        transport.post_events_chunked(&events, config.visibility_post_chunk_size)?;
    }
    if let Some(actions_url) = &config.actions_heartbeat_url {
        let transport = HttpJsonTransport::new(actions_url)?;
        transport.post_json(&actions_heartbeat_json(config)?)?;
    }

    Ok(())
}

fn actions_heartbeat_json(config: &AgentConfig) -> Result<String, String> {
    let hostname = hostname().unwrap_or_else(|| config.device_id.clone());
    let primary_ip = primary_ip().unwrap_or_default();
    let last_seen = rfc3339_utc(SystemTime::now())?;
    let payload = serde_json::json!({
        "agent_uid": config.agent_id,
        "org_id": "default-org",
        "host_id": config.device_id,
        "hostname": hostname,
        "agent_version": config.sensor_version,
        "last_seen": last_seen,
        "status": "online",
        "labels": ["visibility-lab", "linux"],
        "note": "Registered from Linux lab visibility collector",
        "capabilities": {
            "visibility": true,
            "dynamic_detection_packs": config.detection_packs_enabled,
            "platform": "linux"
        },
        "platform": {
            "hostname": hostname,
            "os": "linux",
            "architecture": env::consts::ARCH,
            "kernel_version": kernel_version().unwrap_or_default(),
            "primary_ip": primary_ip
        },
        "network": {
            "primary_ip": primary_ip
        }
    });
    Ok(payload.to_string())
}

fn hostname() -> Option<String> {
    fs::read_to_string("/etc/hostname")
        .ok()
        .map(|value| value.trim().to_string())
        .filter(|value| !value.is_empty())
}

fn kernel_version() -> Option<String> {
    fs::read_to_string("/proc/sys/kernel/osrelease")
        .ok()
        .map(|value| value.trim().to_string())
        .filter(|value| !value.is_empty())
}

fn primary_ip() -> Option<String> {
    let socket = UdpSocket::bind("0.0.0.0:0").ok()?;
    socket.connect("192.0.2.1:80").ok()?;
    let address = socket.local_addr().ok()?;
    Some(address.ip().to_string()).filter(|value| !value.is_empty())
}

fn performance_event(
    config: &AgentConfig,
    collector_name: &str,
    collector_runtime_ms: u64,
    pack_eval_runtime_ms: Option<u64>,
    event_queue_depth: u64,
) -> AegisEvent {
    let (process_cpu_percent, process_memory_rss_mb) = process_performance_sample();
    AegisEvent::new(
        config,
        "aegis.agent.performance",
        SystemTime::now(),
        EventPayload::AgentPerformance {
            os: env::consts::OS.to_string(),
            process_cpu_percent,
            process_memory_rss_mb,
            collector_runtime_ms,
            collector_name: collector_name.to_string(),
            collection_interval_ms: Some(millis_u64(config.collection_interval.as_millis())),
            skipped_reason: None,
            event_queue_depth,
            spool_bytes: fs::metadata(&config.event_spool)
                .map(|metadata| metadata.len())
                .unwrap_or(0),
            pack_eval_runtime_ms,
        },
    )
}

fn process_performance_sample() -> (Option<f32>, Option<f32>) {
    let Ok(pid) = get_current_pid() else {
        return (None, None);
    };
    let mut system = System::new();
    system.refresh_processes_specifics(
        ProcessesToUpdate::Some(&[pid]),
        true,
        ProcessRefreshKind::nothing().with_cpu().with_memory(),
    );
    match system.process(pid) {
        Some(process) => (
            Some(process.cpu_usage().max(0.0)),
            Some(process.memory() as f32 / (1024.0 * 1024.0)),
        ),
        None => (None, None),
    }
}

fn millis_u64(value: u128) -> u64 {
    value.min(u64::MAX as u128) as u64
}

fn rfc3339_utc(time: SystemTime) -> Result<String, String> {
    let duration = time
        .duration_since(UNIX_EPOCH)
        .map_err(|_| "system time is before Unix epoch".to_string())?;
    let total_seconds = duration.as_secs();
    let days = (total_seconds / 86_400) as i64;
    let seconds_of_day = total_seconds % 86_400;
    let (year, month, day) = civil_from_days(days);
    let hour = seconds_of_day / 3_600;
    let minute = (seconds_of_day % 3_600) / 60;
    let second = seconds_of_day % 60;
    Ok(format!(
        "{year:04}-{month:02}-{day:02}T{hour:02}:{minute:02}:{second:02}Z"
    ))
}

fn civil_from_days(days_since_epoch: i64) -> (i64, u64, u64) {
    let z = days_since_epoch + 719_468;
    let era = if z >= 0 { z } else { z - 146_096 } / 146_097;
    let day_of_era = z - era * 146_097;
    let year_of_era =
        (day_of_era - day_of_era / 1_460 + day_of_era / 36_524 - day_of_era / 146_096) / 365;
    let mut year = year_of_era + era * 400;
    let day_of_year = day_of_era - (365 * year_of_era + year_of_era / 4 - year_of_era / 100);
    let month_prime = (5 * day_of_year + 2) / 153;
    let day = day_of_year - (153 * month_prime + 2) / 5 + 1;
    let month = month_prime + if month_prime < 10 { 3 } else { -9 };
    if month <= 2 {
        year += 1;
    }
    (year, month as u64, day as u64)
}

#[derive(Debug, Default)]
struct Args {
    once: bool,
    stdout: bool,
}

impl Args {
    fn parse<I>(args: I) -> Self
    where
        I: IntoIterator<Item = String>,
    {
        let mut parsed = Self::default();
        for arg in args {
            match arg.as_str() {
                "--once" => parsed.once = true,
                "--stdout" => parsed.stdout = true,
                _ => {}
            }
        }
        parsed
    }
}
