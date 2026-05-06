//! Agent configuration.

use std::env;
use std::path::PathBuf;

use base64::Engine;

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
    /// Lab-only detection-pipeline controller base URL (`http://host:port`).
    pub controller_url: Option<String>,
    /// When true, fetch and evaluate signed detection packs (observe-only).
    pub detection_packs_enabled: bool,
    /// Override directory for verified pack cache (default: sibling of spool `detection-pack/`).
    pub detection_pack_cache: Option<PathBuf>,
    /// Ed25519 verifying key (32 raw bytes) for `detection_pack.v1` signatures, standard base64.
    pub detection_pack_public_key: Option<[u8; 32]>,
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
        let controller_url = env::var("AEGIS_CONTROLLER_URL")
            .ok()
            .map(|s| s.trim().to_string())
            .filter(|s| !s.is_empty());
        let detection_packs_enabled = env_bool("AEGIS_DETECTION_PACKS_ENABLED", false)?;
        let detection_pack_cache = env::var("AEGIS_DETECTION_PACK_CACHE")
            .ok()
            .map(|s| s.trim().to_string())
            .filter(|s| !s.is_empty())
            .map(PathBuf::from);
        let detection_pack_public_key = match env::var("AEGIS_DETECTION_PACK_PUBLIC_KEY") {
            Ok(s) if !s.trim().is_empty() => Some(parse_ed25519_verifying_key_b64(&s)?),
            _ => None,
        };
        if detection_packs_enabled && detection_pack_public_key.is_none() {
            return Err(
                "AEGIS_DETECTION_PACKS_ENABLED requires AEGIS_DETECTION_PACK_PUBLIC_KEY (base64, 32 bytes)"
                    .to_string(),
            );
        }

        require_safe_identifier("AEGIS_AGENT_ID", &agent_id)?;
        require_safe_identifier("AEGIS_DEVICE_ID", &device_id)?;

        Ok(Self {
            agent_id,
            device_id,
            sensor_version,
            backend_url,
            event_spool,
            collect_command_line,
            controller_url,
            detection_packs_enabled,
            detection_pack_cache,
            detection_pack_public_key,
        })
    }
}

fn parse_ed25519_verifying_key_b64(raw: &str) -> Result<[u8; 32], String> {
    let trimmed = raw.trim();
    let dec = base64::engine::general_purpose::STANDARD
        .decode(trimmed)
        .map_err(|e| format!("AEGIS_DETECTION_PACK_PUBLIC_KEY base64: {e}"))?;
    if dec.len() != 32 {
        return Err(format!(
            "AEGIS_DETECTION_PACK_PUBLIC_KEY must decode to 32 bytes, got {}",
            dec.len()
        ));
    }
    let mut out = [0u8; 32];
    out.copy_from_slice(&dec);
    Ok(out)
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
