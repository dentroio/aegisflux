//! Startup security checks.

use crate::config::AgentConfig;

/// Validate startup posture for the current phase.
pub fn validate_startup_posture(config: &AgentConfig) -> Result<(), String> {
    if let Some(url) = &config.backend_url {
        if !(url.starts_with("http://localhost") || url.starts_with("http://127.0.0.1")) {
            return Err(
                "AEGIS_BACKEND_URL must use localhost HTTP for the no-dependency Phase 1 lab publisher"
                    .to_string(),
            );
        }
    }

    Ok(())
}
