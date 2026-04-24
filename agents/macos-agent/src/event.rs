//! Event envelope and JSON serialization.

use std::sync::atomic::{AtomicU64, Ordering};
use std::time::{SystemTime, UNIX_EPOCH};

use crate::config::AgentConfig;

static EVENT_SEQUENCE: AtomicU64 = AtomicU64::new(1);

/// A normalized Aegis event.
#[derive(Debug, Clone)]
pub struct AegisEvent {
    /// Schema version.
    pub schema_version: &'static str,
    /// Event identifier.
    pub event_id: String,
    /// Event type.
    pub event_type: String,
    /// Event timestamp as milliseconds since Unix epoch.
    pub timestamp_ms: u128,
    /// Event source.
    pub source: &'static str,
    /// Device identity.
    pub device_id: String,
    /// Agent identity.
    pub agent_id: String,
    /// Sensor version.
    pub sensor_version: String,
    /// Monotonic local sequence number.
    pub sequence: u64,
    /// Event payload.
    pub payload: EventPayload,
}

/// Supported Phase 1 event payloads.
#[derive(Debug, Clone)]
pub enum EventPayload {
    /// Agent heartbeat payload.
    Heartbeat {
        /// Agent status.
        status: String,
        /// Human-readable status message.
        message: String,
        /// Operating system family reported by the compiled agent.
        os: String,
        /// CPU architecture reported by the compiled agent.
        arch: String,
    },
    /// Collector status payload.
    CollectorStatus {
        /// Collector name.
        collector: String,
        /// Collector status.
        status: String,
        /// Human-readable status message.
        message: String,
    },
}

impl AegisEvent {
    /// Build a new event from the common agent envelope.
    pub fn new(
        config: &AgentConfig,
        event_type: &str,
        timestamp: SystemTime,
        payload: EventPayload,
    ) -> Self {
        let sequence = EVENT_SEQUENCE.fetch_add(1, Ordering::Relaxed);
        let timestamp_ms = timestamp
            .duration_since(UNIX_EPOCH)
            .map(|duration| duration.as_millis())
            .unwrap_or(0);
        let event_id = format!("{}-{}-{}", config.agent_id, timestamp_ms, sequence);

        Self {
            schema_version: "visibility.v1",
            event_id,
            event_type: event_type.to_string(),
            timestamp_ms,
            source: "aegis-macos-agent",
            device_id: config.device_id.clone(),
            agent_id: config.agent_id.clone(),
            sensor_version: config.sensor_version.clone(),
            sequence,
            payload,
        }
    }

    /// Serialize the event as compact JSON.
    pub fn to_json(&self) -> String {
        let payload = match &self.payload {
            EventPayload::Heartbeat {
                status,
                message,
                os,
                arch,
            } => format!(
                r#"{{"status":"{}","message":"{}","os":"{}","arch":"{}"}}"#,
                escape_json(status),
                escape_json(message),
                escape_json(os),
                escape_json(arch)
            ),
            EventPayload::CollectorStatus {
                collector,
                status,
                message,
            } => format!(
                r#"{{"collector":"{}","status":"{}","message":"{}"}}"#,
                escape_json(collector),
                escape_json(status),
                escape_json(message)
            ),
        };

        format!(
            r#"{{"schema_version":"{}","event_id":"{}","event_type":"{}","timestamp_ms":{},"source":"{}","device_id":"{}","agent_id":"{}","sensor_version":"{}","sequence":{},"payload":{}}}"#,
            self.schema_version,
            escape_json(&self.event_id),
            escape_json(&self.event_type),
            self.timestamp_ms,
            self.source,
            escape_json(&self.device_id),
            escape_json(&self.agent_id),
            escape_json(&self.sensor_version),
            self.sequence,
            payload
        )
    }
}

fn escape_json(value: &str) -> String {
    let mut escaped = String::with_capacity(value.len());
    for ch in value.chars() {
        match ch {
            '"' => escaped.push_str("\\\""),
            '\\' => escaped.push_str("\\\\"),
            '\n' => escaped.push_str("\\n"),
            '\r' => escaped.push_str("\\r"),
            '\t' => escaped.push_str("\\t"),
            ch if ch.is_control() => escaped.push_str(&format!("\\u{:04x}", ch as u32)),
            ch => escaped.push(ch),
        }
    }
    escaped
}

#[cfg(test)]
mod tests {
    use super::escape_json;

    #[test]
    fn escapes_json_control_characters() {
        assert_eq!(escape_json("a\"b\\c\n"), "a\\\"b\\\\c\\n");
    }
}
