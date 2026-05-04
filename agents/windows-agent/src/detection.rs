//! Non-blocking AI-agent and automation detections.

use std::collections::{BTreeSet, HashMap};
use std::time::SystemTime;

use crate::config::AgentConfig;
use crate::event::{AegisEvent, DetectionEvidence, EventPayload};

/// Run Phase 1 AI-agent and automation detections over one visibility batch.
pub fn detect_ai_agent_activity(config: &AgentConfig, events: &[AegisEvent]) -> Vec<AegisEvent> {
    let context = DetectionContext::from_events(events);
    let mut detections = Vec::new();

    for process in context.processes.values() {
        if let Some(candidate) = detect_command_line_agent_marker(process, &context) {
            detections.extend(candidate.into_events(config));
        } else if let Some(candidate) = detect_script_or_agent_runtime(process, &context) {
            detections.extend(candidate.into_events(config));
        } else if let Some(candidate) = detect_powershell_automation(process, &context) {
            detections.extend(candidate.into_events(config));
        }
    }

    if let Some(candidate) = detect_browser_ai_usage(&context) {
        detections.extend(candidate.into_events(config));
    }

    detections
}

#[derive(Debug)]
struct DetectionContext {
    processes: HashMap<String, ProcessObservation>,
    processes_by_pid: HashMap<u32, String>,
    flows_by_pid: HashMap<u32, Vec<FlowObservation>>,
    dns_queries: Vec<DnsObservation>,
}

impl DetectionContext {
    fn from_events(events: &[AegisEvent]) -> Self {
        let mut context = Self {
            processes: HashMap::new(),
            processes_by_pid: HashMap::new(),
            flows_by_pid: HashMap::new(),
            dns_queries: Vec::new(),
        };

        for event in events {
            match &event.payload {
                EventPayload::ProcessStarted {
                    process_guid,
                    parent_process_guid,
                    pid,
                    ppid,
                    name,
                    path,
                    command_line,
                    ..
                } => {
                    context.processes_by_pid.insert(*pid, process_guid.clone());
                    context.processes.insert(
                        process_guid.clone(),
                        ProcessObservation {
                            process_guid: process_guid.clone(),
                            parent_process_guid: parent_process_guid.clone(),
                            pid: *pid,
                            ppid: *ppid,
                            name: name.clone(),
                            path: path.clone(),
                            command_line: command_line.clone(),
                        },
                    );
                }
                EventPayload::FlowStarted {
                    flow_id,
                    pid,
                    remote_ip,
                    remote_port,
                    remote_hostname,
                    ..
                } => {
                    if let Some(pid) = pid {
                        context
                            .flows_by_pid
                            .entry(*pid)
                            .or_default()
                            .push(FlowObservation {
                                flow_id: flow_id.clone(),
                                remote_ip: remote_ip.clone(),
                                remote_port: *remote_port,
                                remote_hostname: remote_hostname.clone(),
                            });
                    }
                }
                EventPayload::DnsObserved { query, answers, .. } => {
                    context.dns_queries.push(DnsObservation {
                        query: query.clone(),
                        answers: answers.clone(),
                    });
                }
                _ => {}
            }
        }

        context
    }
}

#[derive(Debug, Clone)]
struct ProcessObservation {
    process_guid: String,
    parent_process_guid: Option<String>,
    pid: u32,
    ppid: Option<u32>,
    name: String,
    path: Option<String>,
    command_line: Option<String>,
}

#[derive(Debug, Clone)]
struct FlowObservation {
    flow_id: String,
    remote_ip: String,
    remote_port: Option<u16>,
    remote_hostname: Option<String>,
}

#[derive(Debug, Clone)]
struct DnsObservation {
    query: String,
    answers: Vec<String>,
}

#[derive(Debug)]
struct DetectionCandidate {
    classification: String,
    process_guid: Option<String>,
    flow_id: Option<String>,
    agent_likelihood: f32,
    confidence: f32,
    risk_score: u8,
    patterns: Vec<String>,
    evidence: Vec<DetectionEvidence>,
    recommended_action: String,
    title: String,
    description: String,
}

impl DetectionCandidate {
    fn into_events(self, config: &AgentConfig) -> Vec<AegisEvent> {
        let detection_id = detection_id(config, &self.classification, self.process_guid.as_deref());
        let finding_id = format!("finding-{detection_id}");
        let severity = severity_for_score(self.risk_score);
        let detection = AegisEvent::new(
            config,
            "aegis.agent.detected",
            SystemTime::now(),
            EventPayload::AgentDetected {
                detection_id: detection_id.clone(),
                process_guid: self.process_guid.clone(),
                flow_id: self.flow_id.clone(),
                classification: self.classification,
                agent_likelihood: self.agent_likelihood,
                confidence: self.confidence,
                risk_score: self.risk_score,
                detected_patterns: self.patterns,
                evidence: self.evidence.clone(),
                recommended_action: self.recommended_action.clone(),
            },
        );
        let finding = AegisEvent::new(
            config,
            "aegis.risk_finding.created",
            SystemTime::now(),
            EventPayload::RiskFindingCreated {
                finding_id,
                severity,
                risk_score: self.risk_score,
                title: self.title,
                description: self.description,
                process_guid: self.process_guid,
                flow_id: self.flow_id,
                detection_id: Some(detection_id),
                evidence: self.evidence,
                recommended_action: self.recommended_action,
            },
        );

        vec![detection, finding]
    }
}

fn detect_command_line_agent_marker(
    process: &ProcessObservation,
    context: &DetectionContext,
) -> Option<DetectionCandidate> {
    let command_line = process.command_line.as_deref()?;
    let command_lower = command_line.to_ascii_lowercase();
    if !contains_any(
        &command_lower,
        &[
            "agent_runner",
            "autogen",
            "crewai",
            "langchain",
            "llamaindex",
            "semantic-kernel",
            "openai",
            "anthropic",
        ],
    ) {
        return None;
    }

    let related_flow = best_flow_for_process(process.pid, context);
    let mut patterns = vec!["agent_marker_command_line".to_string()];
    let mut evidence_items = vec![
        evidence("process", &process.name, 0.7),
        evidence("command_line", command_line, 0.9),
    ];
    if let Some(path) = &process.path {
        evidence_items.push(evidence("process_path", path, 0.62));
    }
    if let Some(flow) = &related_flow {
        patterns.push("process_network_connection".to_string());
        evidence_items.push(evidence("destination", &flow.destination(), 0.64));
    }

    Some(DetectionCandidate {
        classification: "script_or_agent_runtime".to_string(),
        process_guid: Some(process.process_guid.clone()),
        flow_id: related_flow.map(|flow| flow.flow_id),
        agent_likelihood: 0.7,
        confidence: 0.76,
        risk_score: 42,
        patterns,
        evidence: evidence_items,
        recommended_action: "review".to_string(),
        title: "Agent-like command line marker observed".to_string(),
        description: "A process command line contained explicit agent or model-tooling markers."
            .to_string(),
    })
}

fn detect_script_or_agent_runtime(
    process: &ProcessObservation,
    context: &DetectionContext,
) -> Option<DetectionCandidate> {
    let name = process.name.to_ascii_lowercase();
    let command_line = process.command_line.as_deref().unwrap_or_default();
    let command_lower = command_line.to_ascii_lowercase();
    let is_runtime = matches!(
        name.as_str(),
        "python.exe" | "pythonw.exe" | "node.exe" | "npm.exe" | "npx.exe" | "bun.exe" | "uv.exe"
    );
    if !is_runtime {
        return None;
    }

    let mut patterns = BTreeSet::new();
    let mut evidence_items = vec![evidence("process", &process.name, 0.75)];
    if !command_line.is_empty() {
        evidence_items.push(evidence("command_line", command_line, 0.86));
    }
    if let Some(path) = &process.path {
        evidence_items.push(evidence("process_path", path, 0.62));
    }
    if contains_any(
        &command_lower,
        &[
            "agent",
            "autogen",
            "crewai",
            "langchain",
            "llamaindex",
            "semantic-kernel",
            "openai",
            "anthropic",
            "tool",
        ],
    ) {
        patterns.insert("agent_runtime_command_line".to_string());
    }
    if name.contains("python") {
        patterns.insert("python_runtime".to_string());
    }
    if name == "node.exe" || name == "npm.exe" || name == "npx.exe" {
        patterns.insert("node_runtime".to_string());
    }

    if let Some(parent) = parent_process(process, context) {
        let parent_name = parent.name.to_ascii_lowercase();
        evidence_items.push(evidence("parent_process", &parent.name, 0.7));
        if matches!(
            parent_name.as_str(),
            "cursor.exe" | "code.exe" | "devenv.exe"
        ) {
            patterns.insert("developer_tool_parent".to_string());
        }
    }

    let related_flow = best_flow_for_process(process.pid, context);
    if let Some(flow) = &related_flow {
        patterns.insert("runtime_network_connection".to_string());
        evidence_items.push(evidence("destination", &flow.destination(), 0.7));
    }

    if patterns.len() < 2 {
        return None;
    }

    let risk_score = if patterns.contains("agent_runtime_command_line") {
        48
    } else {
        32
    };

    Some(DetectionCandidate {
        classification: "script_or_agent_runtime".to_string(),
        process_guid: Some(process.process_guid.clone()),
        flow_id: related_flow.map(|flow| flow.flow_id),
        agent_likelihood: if risk_score >= 48 { 0.78 } else { 0.52 },
        confidence: 0.72,
        risk_score,
        patterns: patterns.into_iter().collect(),
        evidence: evidence_items,
        recommended_action: "review".to_string(),
        title: "Possible local AI agent runtime observed".to_string(),
        description:
            "A script/runtime process showed agent-like command-line, parent, or network evidence."
                .to_string(),
    })
}

fn detect_powershell_automation(
    process: &ProcessObservation,
    context: &DetectionContext,
) -> Option<DetectionCandidate> {
    let name = process.name.to_ascii_lowercase();
    if name != "powershell.exe" && name != "pwsh.exe" {
        return None;
    }

    let command_line = process.command_line.as_deref().unwrap_or_default();
    let command_lower = command_line.to_ascii_lowercase();
    let mut patterns = BTreeSet::new();
    if contains_any(
        &command_lower,
        &[
            "-encodedcommand",
            "frombase64string",
            "downloadstring",
            "invoke-webrequest",
            "invoke-restmethod",
            " iwr ",
            " irm ",
            "curl ",
        ],
    ) {
        patterns.insert("scripted_powershell_command".to_string());
    }

    let related_flow = best_flow_for_process(process.pid, context);
    if related_flow.is_some() {
        patterns.insert("powershell_network_connection".to_string());
    }

    if patterns.is_empty() {
        return None;
    }

    let mut evidence_items = vec![evidence("process", &process.name, 0.75)];
    if !command_line.is_empty() {
        evidence_items.push(evidence("command_line", command_line, 0.86));
    }
    if let Some(flow) = &related_flow {
        evidence_items.push(evidence("destination", &flow.destination(), 0.68));
    }

    Some(DetectionCandidate {
        classification: "shell_automation".to_string(),
        process_guid: Some(process.process_guid.clone()),
        flow_id: related_flow.map(|flow| flow.flow_id),
        agent_likelihood: 0.35,
        confidence: 0.65,
        risk_score: 35,
        patterns: patterns.into_iter().collect(),
        evidence: evidence_items,
        recommended_action: "review".to_string(),
        title: "PowerShell automation signal observed".to_string(),
        description: "PowerShell command shape or network activity matched automation heuristics."
            .to_string(),
    })
}

fn detect_browser_ai_usage(context: &DetectionContext) -> Option<DetectionCandidate> {
    let ai_dns = context
        .dns_queries
        .iter()
        .find(|dns| is_ai_destination(&dns.query))?;
    let browser = context.processes.values().find(|process| {
        matches!(
            process.name.to_ascii_lowercase().as_str(),
            "chrome.exe" | "msedge.exe" | "firefox.exe" | "brave.exe"
        )
    });

    let mut evidence_items = vec![evidence("dns_query", &ai_dns.query, 0.75)];
    if !ai_dns.answers.is_empty() {
        evidence_items.push(evidence("dns_answer", &ai_dns.answers.join(","), 0.55));
    }
    if let Some(browser) = browser {
        evidence_items.push(evidence("browser_process", &browser.name, 0.65));
    }

    Some(DetectionCandidate {
        classification: "browser_ai_usage".to_string(),
        process_guid: browser.map(|process| process.process_guid.clone()),
        flow_id: None,
        agent_likelihood: 0.22,
        confidence: if browser.is_some() { 0.62 } else { 0.45 },
        risk_score: 18,
        patterns: vec!["ai_destination_dns".to_string()],
        evidence: evidence_items,
        recommended_action: "monitor".to_string(),
        title: "Browser AI destination observed".to_string(),
        description:
            "A known AI or model-service domain appeared in DNS cache; this is monitor-only evidence."
                .to_string(),
    })
}

fn parent_process<'a>(
    process: &ProcessObservation,
    context: &'a DetectionContext,
) -> Option<&'a ProcessObservation> {
    if let Some(parent_guid) = &process.parent_process_guid {
        context.processes.get(parent_guid)
    } else if let Some(ppid) = process.ppid {
        context
            .processes_by_pid
            .get(&ppid)
            .and_then(|guid| context.processes.get(guid))
    } else {
        None
    }
}

fn best_flow_for_process(pid: u32, context: &DetectionContext) -> Option<FlowObservation> {
    context
        .flows_by_pid
        .get(&pid)
        .and_then(|flows| {
            flows
                .iter()
                .find(|flow| flow.remote_port == Some(443))
                .or(flows.first())
        })
        .cloned()
}

fn contains_any(value: &str, needles: &[&str]) -> bool {
    needles.iter().any(|needle| value.contains(needle))
}

fn is_ai_destination(query: &str) -> bool {
    let query = query.to_ascii_lowercase();
    contains_any(
        &query,
        &[
            "openai.com",
            "chatgpt.com",
            "chat.openai.com",
            "oaistatic.com",
            "oaiusercontent.com",
            "openaiapi-site.azureedge.net",
            "anthropic.com",
            "claude.ai",
            "gemini.google.com",
            "generativelanguage.googleapis.com",
            "copilot.microsoft.com",
            "cursor.com",
            "model-gateway",
        ],
    )
}

fn evidence(evidence_type: &str, value: &str, confidence: f32) -> DetectionEvidence {
    DetectionEvidence {
        evidence_type: evidence_type.to_string(),
        value: truncate_evidence(value),
        confidence,
    }
}

fn truncate_evidence(value: &str) -> String {
    const MAX_EVIDENCE_LEN: usize = 512;
    if value.len() <= MAX_EVIDENCE_LEN {
        value.to_string()
    } else {
        format!("{}...[truncated]", &value[..MAX_EVIDENCE_LEN])
    }
}

fn detection_id(config: &AgentConfig, classification: &str, process_guid: Option<&str>) -> String {
    let target = process_guid.unwrap_or("device");
    format!(
        "det-{}-{}-{}",
        config.device_id,
        classification,
        stable_hash(target)
    )
}

fn stable_hash(value: &str) -> u64 {
    let mut hash = 1469598103934665603_u64;
    for byte in value.as_bytes() {
        hash ^= u64::from(*byte);
        hash = hash.wrapping_mul(1099511628211);
    }
    hash
}

fn severity_for_score(score: u8) -> String {
    match score {
        0..=19 => "info",
        20..=39 => "low",
        40..=69 => "medium",
        70..=89 => "high",
        _ => "critical",
    }
    .to_string()
}

impl FlowObservation {
    fn destination(&self) -> String {
        let host = self
            .remote_hostname
            .as_deref()
            .unwrap_or(self.remote_ip.as_str());
        match self.remote_port {
            Some(port) => format!("{host}:{port}"),
            None => host.to_string(),
        }
    }
}

#[cfg(test)]
mod tests {
    use std::time::SystemTime;

    use super::detect_ai_agent_activity;
    use crate::config::AgentConfig;
    use crate::event::{AegisEvent, EventPayload};

    fn config() -> AgentConfig {
        AgentConfig {
            agent_id: "agent-1".to_string(),
            device_id: "device-1".to_string(),
            sensor_version: "0.1.0".to_string(),
            backend_url: None,
            event_spool: "/tmp/events.jsonl".into(),
            collect_command_line: true,
        }
    }

    #[test]
    fn detects_agent_like_python_runtime() {
        let config = config();
        let events = vec![
            AegisEvent::new(
                &config,
                "aegis.process.started",
                SystemTime::now(),
                EventPayload::ProcessStarted {
                    process_guid: "proc-1".to_string(),
                    parent_process_guid: None,
                    pid: 42,
                    ppid: None,
                    name: "python.exe".to_string(),
                    path: None,
                    command_line: Some("python agent_runner.py --tool openai".to_string()),
                    user: None,
                    logon_session_id: None,
                    integrity_level: None,
                    sha256: None,
                    publisher: None,
                    collection_method: "test".to_string(),
                },
            ),
            AegisEvent::new(
                &config,
                "aegis.flow.started",
                SystemTime::now(),
                EventPayload::FlowStarted {
                    flow_id: "flow-1".to_string(),
                    process_guid: Some("proc-1".to_string()),
                    pid: Some(42),
                    process_name: Some("python.exe".to_string()),
                    user: None,
                    protocol: "tcp".to_string(),
                    direction: "outbound".to_string(),
                    local_ip: "10.0.0.2".to_string(),
                    local_port: Some(50000),
                    remote_ip: "203.0.113.10".to_string(),
                    remote_port: Some(443),
                    remote_hostname: Some("api.model-gateway.lab".to_string()),
                    attribution_method: "test".to_string(),
                    attribution_confidence: 0.9,
                },
            ),
        ];

        let detections = detect_ai_agent_activity(&config, &events);

        assert_eq!(detections.len(), 2);
        assert_eq!(detections[0].event_type, "aegis.agent.detected");
        assert_eq!(detections[1].event_type, "aegis.risk_finding.created");
    }

    #[test]
    fn detects_agent_marker_in_cmd_command_line() {
        let config = config();
        let events = vec![AegisEvent::new(
            &config,
            "aegis.process.started",
            SystemTime::now(),
            EventPayload::ProcessStarted {
                process_guid: "proc-cmd".to_string(),
                parent_process_guid: None,
                pid: 132,
                ppid: None,
                name: "cmd.exe".to_string(),
                path: Some(r"C:\Windows\System32\cmd.exe".to_string()),
                command_line: Some(
                    r"C:\WINDOWS\system32\cmd.exe /k title agent_runner openai tool".to_string(),
                ),
                user: None,
                logon_session_id: None,
                integrity_level: None,
                sha256: None,
                publisher: None,
                collection_method: "test".to_string(),
            },
        )];

        let detections = detect_ai_agent_activity(&config, &events);

        assert_eq!(detections.len(), 2);
        assert_eq!(detections[0].event_type, "aegis.agent.detected");
        match &detections[0].payload {
            EventPayload::AgentDetected {
                classification,
                detected_patterns,
                ..
            } => {
                assert_eq!(classification, "script_or_agent_runtime");
                assert_eq!(detected_patterns, &["agent_marker_command_line"]);
            }
            _ => panic!("expected agent detection payload"),
        }
    }
}
