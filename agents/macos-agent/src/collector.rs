//! Visibility collectors.

use std::time::SystemTime;

use crate::config::AgentConfig;
use crate::event::{AegisEvent, EventPayload};

/// A visibility collector.
pub trait Collector {
    /// Collector display name.
    fn name(&self) -> &'static str;

    /// Collect one batch of visibility events.
    fn collect_once(&self, config: &AgentConfig) -> Result<Vec<AegisEvent>, String>;
}

/// Phase 1 collector set.
#[derive(Debug, Default)]
pub struct CollectorSet {
    process: ProcessCollector,
    network: NetworkCollector,
    dns: DnsCollector,
}

impl CollectorSet {
    /// Return collectors in deterministic order.
    pub fn collectors(&self) -> Vec<&dyn Collector> {
        vec![&self.process, &self.network, &self.dns]
    }
}

#[derive(Debug, Default)]
struct ProcessCollector;

#[derive(Debug, Default)]
struct NetworkCollector;

#[derive(Debug, Default)]
struct DnsCollector;

impl Collector for ProcessCollector {
    fn name(&self) -> &'static str {
        "macos.process"
    }

    fn collect_once(&self, config: &AgentConfig) -> Result<Vec<AegisEvent>, String> {
        Ok(vec![collector_status(
            config,
            self.name(),
            "pending",
            macos_only_message(
                "process inventory collector requires Endpoint Security implementation",
            ),
        )])
    }
}

impl Collector for NetworkCollector {
    fn name(&self) -> &'static str {
        "macos.network"
    }

    fn collect_once(&self, config: &AgentConfig) -> Result<Vec<AegisEvent>, String> {
        Ok(vec![collector_status(
            config,
            self.name(),
            "pending",
            macos_only_message("network attribution collector requires approved macOS API design"),
        )])
    }
}

impl Collector for DnsCollector {
    fn name(&self) -> &'static str {
        "macos.dns"
    }

    fn collect_once(&self, config: &AgentConfig) -> Result<Vec<AegisEvent>, String> {
        Ok(vec![collector_status(
            config,
            self.name(),
            "pending",
            macos_only_message("DNS observation collector requires approved macOS API design"),
        )])
    }
}

fn collector_status(
    config: &AgentConfig,
    collector: &str,
    status: &str,
    message: String,
) -> AegisEvent {
    AegisEvent::new(
        config,
        "aegis.collector.status",
        SystemTime::now(),
        EventPayload::CollectorStatus {
            collector: collector.to_string(),
            status: status.to_string(),
            message,
        },
    )
}

#[cfg(target_os = "macos")]
fn macos_only_message(message: &str) -> String {
    message.to_string()
}

#[cfg(not(target_os = "macos"))]
fn macos_only_message(message: &str) -> String {
    format!("{message}; running on non-macOS development host")
}
