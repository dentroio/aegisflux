#![forbid(unsafe_code)]
#![warn(missing_docs)]

//! Aegis Windows visibility agent.

mod collector;
mod config;
mod detection;
mod dynamic_pack;
mod event;
mod process_lifecycle;
mod security;
mod transport;

use std::env;
use std::fs;
use std::process::ExitCode;
use std::thread;
use std::time::{Duration, Instant, SystemTime};

use collector::CollectorSet;
use config::AgentConfig;
use event::{AegisEvent, EventPayload};
use sysinfo::{get_current_pid, ProcessRefreshKind, ProcessesToUpdate, System};
use transport::{HttpVisibilityTransport, JsonlSpool};

fn main() -> ExitCode {
    match run() {
        Ok(()) => ExitCode::SUCCESS,
        Err(err) => {
            eprintln!("aegis-windows-agent error: {err}");
            ExitCode::FAILURE
        }
    }
}

fn run() -> Result<(), String> {
    let args = Args::parse(env::args().skip(1));
    let config = AgentConfig::from_env()?;
    security::validate_startup_posture(&config)?;

    let use_lifecycle = args.lifecycle || args.interval_secs.is_some();

    if args.interval_secs.is_none() && !args.once {
        return Err(
            "pass --once for a single collection, or --interval-secs <N> for continuous mode"
                .to_string(),
        );
    }

    if let Some(secs) = args.interval_secs {
        loop {
            run_one_cycle(&config, &args, use_lifecycle)?;
            thread::sleep(Duration::from_secs(secs));
        }
    }

    run_one_cycle(&config, &args, use_lifecycle)
}

fn run_one_cycle(config: &AgentConfig, args: &Args, use_lifecycle: bool) -> Result<(), String> {
    let mut events = Vec::new();
    events.push(AegisEvent::new(
        config,
        "aegis.agent.heartbeat",
        SystemTime::now(),
        EventPayload::Heartbeat {
            status: "collecting".to_string(),
            message: if use_lifecycle {
                "windows visibility agent collection (process lifecycle diff enabled)".to_string()
            } else {
                "windows visibility agent initialized".to_string()
            },
            os: env::consts::OS.to_string(),
            arch: env::consts::ARCH.to_string(),
        },
    ));

    if use_lifecycle {
        events.extend(process_lifecycle::collect_process_lifecycle(
            config,
            &config.process_state_path,
        )?);
    } else {
        events.extend(collector::collect_process_snapshot_batch(config)?);
    }

    let collectors = CollectorSet::default();
    for collector in collectors.collectors() {
        if use_lifecycle && collector.name() == "windows.process" {
            continue;
        }
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
    collector::enrich_flow_hostnames(&mut events);
    collector::correlate_dns_to_flow_attribution(&mut events);
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
        transport.post_events(&events)?;
    }

    Ok(())
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
            collection_interval_ms: None,
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

#[derive(Debug)]
struct Args {
    once: bool,
    stdout: bool,
    lifecycle: bool,
    interval_secs: Option<u64>,
}

impl Default for Args {
    fn default() -> Self {
        Self {
            once: false,
            stdout: false,
            lifecycle: false,
            interval_secs: None,
        }
    }
}

impl Args {
    fn parse<I>(args: I) -> Self
    where
        I: IntoIterator<Item = String>,
    {
        let mut parsed = Self::default();
        let mut it = args.into_iter();
        while let Some(arg) = it.next() {
            match arg.as_str() {
                "--once" => parsed.once = true,
                "--stdout" => parsed.stdout = true,
                "--lifecycle" => parsed.lifecycle = true,
                "--interval-secs" => {
                    let raw = it.next().unwrap_or_default();
                    let secs: u64 = raw.parse().unwrap_or(0);
                    if secs == 0 {
                        eprintln!("aegis-windows-agent: ignoring invalid --interval-secs (need positive integer)");
                    } else {
                        parsed.interval_secs = Some(secs);
                    }
                }
                _ => {}
            }
        }
        parsed
    }
}
