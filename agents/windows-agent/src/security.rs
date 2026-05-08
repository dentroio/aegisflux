//! Startup security checks.

use crate::config::AgentConfig;

/// Validate startup posture for the current phase.
pub fn validate_startup_posture(config: &AgentConfig) -> Result<(), String> {
    if let Some(url) = &config.backend_url {
        if !(url.starts_with("https://")
            || url.starts_with("http://localhost")
            || url.starts_with("http://127.0.0.1")
            || is_private_lab_http_url(url))
        {
            return Err(
                "AEGIS_BACKEND_URL must use https outside localhost/private lab mode".to_string(),
            );
        }
    }

    Ok(())
}

fn is_private_lab_http_url(url: &str) -> bool {
    let Some(rest) = url.strip_prefix("http://") else {
        return false;
    };
    let host = rest.split(['/', ':']).next().unwrap_or_default();
    host.starts_with("10.")
        || host.starts_with("192.168.")
        || matches!(
            host.split('.').collect::<Vec<_>>().as_slice(),
            ["172", second, _, _] if second.parse::<u8>().map(|value| (16..=31).contains(&value)).unwrap_or(false)
        )
}
