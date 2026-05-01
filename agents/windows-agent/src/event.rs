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
    /// Process started or observed in a snapshot payload.
    ProcessStarted {
        /// Stable local process instance identifier.
        process_guid: String,
        /// Stable local parent process instance identifier when known.
        parent_process_guid: Option<String>,
        /// Process ID.
        pid: u32,
        /// Parent process ID when known.
        ppid: Option<u32>,
        /// Process image name.
        name: String,
        /// Executable path when available.
        path: Option<String>,
        /// Command line when available.
        command_line: Option<String>,
        /// User or account when available.
        user: Option<String>,
        /// Logon session when available.
        logon_session_id: Option<String>,
        /// Integrity level when available.
        integrity_level: Option<String>,
        /// SHA-256 hash when available.
        sha256: Option<String>,
        /// Publisher or signer when available.
        publisher: Option<String>,
        /// Collection method.
        collection_method: String,
    },
    /// Network flow started or observed in a snapshot payload.
    #[cfg_attr(not(windows), allow(dead_code))]
    FlowStarted {
        /// Stable local flow identifier.
        flow_id: String,
        /// Stable local process instance identifier when known.
        process_guid: Option<String>,
        /// Process ID when known.
        pid: Option<u32>,
        /// Process image name when known.
        process_name: Option<String>,
        /// User or account when available.
        user: Option<String>,
        /// IP protocol.
        protocol: String,
        /// Flow direction.
        direction: String,
        /// Local IP address.
        local_ip: String,
        /// Local port when applicable.
        local_port: Option<u16>,
        /// Remote IP address.
        remote_ip: String,
        /// Remote port when applicable.
        remote_port: Option<u16>,
        /// Remote hostname when correlated.
        remote_hostname: Option<String>,
        /// Attribution method.
        attribution_method: String,
        /// Attribution confidence.
        attribution_confidence: f32,
    },
    /// DNS observation payload.
    #[cfg_attr(not(windows), allow(dead_code))]
    DnsObserved {
        /// Queried hostname.
        query: String,
        /// DNS query type when known.
        query_type: Option<String>,
        /// DNS answers.
        answers: Vec<String>,
        /// Resolver address when known.
        resolver: Option<String>,
        /// Stable local process instance identifier when known.
        process_guid: Option<String>,
        /// Process ID when known.
        pid: Option<u32>,
        /// Correlation method.
        correlation_method: String,
        /// Correlation confidence.
        correlation_confidence: f32,
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
            source: "aegis-windows-agent",
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
            EventPayload::ProcessStarted {
                process_guid,
                parent_process_guid,
                pid,
                ppid,
                name,
                path,
                command_line,
                user,
                logon_session_id,
                integrity_level,
                sha256,
                publisher,
                collection_method,
            } => format!(
                r#"{{"process_guid":"{}","parent_process_guid":{},"pid":{},"ppid":{},"name":"{}","path":{},"command_line":{},"user":{},"logon_session_id":{},"integrity_level":{},"sha256":{},"publisher":{},"collection_method":"{}"}}"#,
                escape_json(process_guid),
                option_json(parent_process_guid.as_deref()),
                pid,
                option_u32_json(*ppid),
                escape_json(name),
                option_json(path.as_deref()),
                option_json(command_line.as_deref()),
                option_json(user.as_deref()),
                option_json(logon_session_id.as_deref()),
                option_json(integrity_level.as_deref()),
                option_json(sha256.as_deref()),
                option_json(publisher.as_deref()),
                escape_json(collection_method)
            ),
            EventPayload::FlowStarted {
                flow_id,
                process_guid,
                pid,
                process_name,
                user,
                protocol,
                direction,
                local_ip,
                local_port,
                remote_ip,
                remote_port,
                remote_hostname,
                attribution_method,
                attribution_confidence,
            } => format!(
                r#"{{"flow_id":"{}","process_guid":{},"pid":{},"process_name":{},"user":{},"protocol":"{}","direction":"{}","local_ip":"{}","local_port":{},"remote_ip":"{}","remote_port":{},"remote_hostname":{},"attribution_method":"{}","attribution_confidence":{}}}"#,
                escape_json(flow_id),
                option_json(process_guid.as_deref()),
                option_u32_json(*pid),
                option_json(process_name.as_deref()),
                option_json(user.as_deref()),
                escape_json(protocol),
                escape_json(direction),
                escape_json(local_ip),
                option_u16_json(*local_port),
                escape_json(remote_ip),
                option_u16_json(*remote_port),
                option_json(remote_hostname.as_deref()),
                escape_json(attribution_method),
                finite_f32_json(*attribution_confidence)
            ),
            EventPayload::DnsObserved {
                query,
                query_type,
                answers,
                resolver,
                process_guid,
                pid,
                correlation_method,
                correlation_confidence,
            } => format!(
                r#"{{"query":"{}","query_type":{},"answers":{},"resolver":{},"process_guid":{},"pid":{},"correlation_method":"{}","correlation_confidence":{}}}"#,
                escape_json(query),
                option_json(query_type.as_deref()),
                string_array_json(answers),
                option_json(resolver.as_deref()),
                option_json(process_guid.as_deref()),
                option_u32_json(*pid),
                escape_json(correlation_method),
                finite_f32_json(*correlation_confidence)
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

fn option_json(value: Option<&str>) -> String {
    match value {
        Some(value) => format!("\"{}\"", escape_json(value)),
        None => "null".to_string(),
    }
}

fn option_u32_json(value: Option<u32>) -> String {
    match value {
        Some(value) => value.to_string(),
        None => "null".to_string(),
    }
}

fn option_u16_json(value: Option<u16>) -> String {
    match value {
        Some(value) => value.to_string(),
        None => "null".to_string(),
    }
}

fn string_array_json(values: &[String]) -> String {
    let escaped = values
        .iter()
        .map(|value| format!("\"{}\"", escape_json(value)))
        .collect::<Vec<_>>()
        .join(",");
    format!("[{escaped}]")
}

fn finite_f32_json(value: f32) -> String {
    if value.is_finite() {
        value.clamp(0.0, 1.0).to_string()
    } else {
        "0".to_string()
    }
}

#[cfg(test)]
mod tests {
    use super::{EventPayload, escape_json};

    #[test]
    fn escapes_json_control_characters() {
        assert_eq!(escape_json("a\"b\\c\n"), "a\\\"b\\\\c\\n");
    }

    #[test]
    fn flow_payload_variant_is_constructible_on_development_hosts() {
        let payload = EventPayload::FlowStarted {
            flow_id: "flow-1".to_string(),
            process_guid: Some("proc-1".to_string()),
            pid: Some(42),
            process_name: Some("python.exe".to_string()),
            user: None,
            protocol: "tcp".to_string(),
            direction: "outbound".to_string(),
            local_ip: "10.10.20.55".to_string(),
            local_port: Some(52944),
            remote_ip: "203.0.113.10".to_string(),
            remote_port: Some(443),
            remote_hostname: Some("api.model-gateway.lab".to_string()),
            attribution_method: "test".to_string(),
            attribution_confidence: 0.9,
        };

        assert!(matches!(payload, EventPayload::FlowStarted { .. }));
    }

    #[test]
    fn dns_payload_variant_is_constructible_on_development_hosts() {
        let payload = EventPayload::DnsObserved {
            query: "api.model-gateway.lab".to_string(),
            query_type: Some("A".to_string()),
            answers: vec!["203.0.113.10".to_string()],
            resolver: None,
            process_guid: None,
            pid: None,
            correlation_method: "test".to_string(),
            correlation_confidence: 0.4,
        };

        assert!(matches!(payload, EventPayload::DnsObserved { .. }));
    }
}
