#![forbid(unsafe_code)]
#![warn(missing_docs)]

//! Aegis macOS visibility agent.

mod collector;
mod config;
mod event;
mod security;
mod transport;

use std::env;
use std::process::ExitCode;
use std::time::SystemTime;

use collector::CollectorSet;
use config::AgentConfig;
use event::{AegisEvent, EventPayload};
use transport::{JsonlSpool, LabHttpPublisher};

fn main() -> ExitCode {
    match run() {
        Ok(()) => ExitCode::SUCCESS,
        Err(err) => {
            eprintln!("aegis-macos-agent error: {err}");
            ExitCode::FAILURE
        }
    }
}

fn run() -> Result<(), String> {
    let args = Args::parse(env::args().skip(1));
    let config = AgentConfig::from_env()?;
    security::validate_startup_posture(&config)?;

    let mut events = Vec::new();
    events.push(AegisEvent::new(
        &config,
        "aegis.agent.heartbeat",
        SystemTime::now(),
        EventPayload::Heartbeat {
            status: "starting".to_string(),
            message: "macOS visibility agent initialized".to_string(),
            os: env::consts::OS.to_string(),
            arch: env::consts::ARCH.to_string(),
        },
    ));

    let collectors = CollectorSet::default();
    for collector in collectors.collectors() {
        events.extend(collector.collect_once(&config)?);
    }

    let spool = JsonlSpool::new(config.event_spool.clone());
    for event in &events {
        spool.append(event)?;
        if args.stdout {
            println!("{}", event.to_json());
        }
    }

    if args.post {
        let backend_url = config
            .backend_url
            .as_deref()
            .ok_or_else(|| "--post requires AEGIS_BACKEND_URL".to_string())?;
        LabHttpPublisher::new(backend_url)?.post_events(&events)?;
    }

    if !args.once {
        return Err(
            "long-running service mode is not implemented yet; use --once for Phase 1 lab runs"
                .to_string(),
        );
    }

    Ok(())
}

#[derive(Debug, Default)]
struct Args {
    once: bool,
    stdout: bool,
    post: bool,
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
                "--post" => parsed.post = true,
                _ => {}
            }
        }
        parsed
    }
}
