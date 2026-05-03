//! Agent configuration.

use std::env;
use std::path::PathBuf;

/// Runtime configuration for the Linux agent.
#[derive(Debug, Clone)]
pub struct AgentConfig {
    /// Stable agent identity.
    pub agent_id: String,
    /// Stable device identity.
    pub device_id: String,
    /// Sensor version included in event envelopes.
    pub sensor_version: String,
    /// Optional backend URL reserved for future outbound telemetry.
    pub backend_url: Option<String>,
    /// Local JSONL event spool path.
    pub event_spool: PathBuf,
    /// Whether command-line collection is enabled.
    pub collect_command_line: bool,
}

impl AgentConfig {
    /// Load configuration from environment variables.
    pub fn from_env() -> Result<Self, String> {
        let agent_id = env_or_default("AEGIS_AGENT_ID", "linux-agent-dev");
        let device_id = env::var("AEGIS_DEVICE_ID").unwrap_or_else(|_| hostname_fallback());
        let sensor_version = env_or_default("AEGIS_SENSOR_VERSION", env!("CARGO_PKG_VERSION"));
        let backend_url = env::var("AEGIS_BACKEND_URL")
            .ok()
            .filter(|value| !value.trim().is_empty());
        let event_spool = env::var("AEGIS_EVENT_SPOOL")
            .map(PathBuf::from)
            .unwrap_or_else(|_| default_spool_path());
        let collect_command_line = env_bool("AEGIS_COLLECT_COMMAND_LINE", false)?;

        require_safe_identifier("AEGIS_AGENT_ID", &agent_id)?;
        require_safe_identifier("AEGIS_DEVICE_ID", &device_id)?;

        Ok(Self {
            agent_id,
            device_id,
            sensor_version,
            backend_url,
            event_spool,
            collect_command_line,
        })
    }
}

fn env_or_default(name: &str, default: &str) -> String {
    env::var(name).unwrap_or_else(|_| default.to_string())
}

fn env_bool(name: &str, default: bool) -> Result<bool, String> {
    match env::var(name) {
        Ok(value) => match value.to_ascii_lowercase().as_str() {
            "1" | "true" | "yes" | "on" => Ok(true),
            "0" | "false" | "no" | "off" => Ok(false),
            _ => Err(format!("{name} must be a boolean value")),
        },
        Err(_) => Ok(default),
    }
}

fn require_safe_identifier(name: &str, value: &str) -> Result<(), String> {
    if value.trim().is_empty() {
        return Err(format!("{name} must not be empty"));
    }

    let safe = value
        .chars()
        .all(|ch| ch.is_ascii_alphanumeric() || matches!(ch, '-' | '_' | '.' | ':'));

    if !safe {
        return Err(format!("{name} contains unsupported characters"));
    }

    Ok(())
}

fn hostname_fallback() -> String {
    env::var("HOSTNAME")
        .or_else(|_| env::var("COMPUTERNAME"))
        .unwrap_or_else(|_| "unknown-device".to_string())
}

fn default_spool_path() -> PathBuf {
    PathBuf::from("/var/lib/aegis/linux-agent/events.jsonl")
}
