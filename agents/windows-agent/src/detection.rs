//! Non-blocking AI-agent and automation detections.

use std::collections::{BTreeSet, HashMap};
use std::net::IpAddr;
use std::time::SystemTime;

use crate::config::AgentConfig;
use crate::event::{AegisEvent, DetectionEvidence, EventPayload};

/// Run Phase 1 AI-agent and automation detections over one visibility batch.
pub fn detect_ai_agent_activity(config: &AgentConfig, events: &[AegisEvent]) -> Vec<AegisEvent> {
    let context = DetectionContext::from_events(events);
    let mut detections = Vec::new();
    let mut matched_processes = BTreeSet::new();

    for process in context.processes.values() {
        if matched_processes.contains(&process.process_guid) {
            continue;
        }
        let candidate = detect_command_line_agent_marker(process, &context)
            .or_else(|| detect_ide_ai_assistant(process, &context))
            .or_else(|| detect_script_or_agent_runtime(process, &context))
            .or_else(|| detect_powershell_automation(process, &context));
        if let Some(candidate) = candidate {
            matched_processes.insert(process.process_guid.clone());
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
    application_category: Option<String>,
    risk_signal: Option<String>,
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
                classification: self.classification.clone(),
                application_category: self.application_category.clone(),
                risk_signal: self.risk_signal.clone(),
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
                classification: Some(self.classification),
                application_category: self.application_category,
                risk_signal: self.risk_signal,
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
        application_category: Some("script_or_agent_runtime".to_string()),
        risk_signal: None,
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

fn detect_ide_ai_assistant(
    process: &ProcessObservation,
    context: &DetectionContext,
) -> Option<DetectionCandidate> {
    let name = process.name.to_ascii_lowercase();
    if !matches!(
        name.as_str(),
        "node.exe" | "python.exe" | "pythonw.exe" | "npm.exe" | "npx.exe"
    ) {
        return None;
    }
    let parent = parent_process(process, context)?;
    let parent_name = parent.name.to_ascii_lowercase();
    if !matches!(
        parent_name.as_str(),
        "cursor.exe" | "code.exe" | "devenv.exe"
    ) {
        return None;
    }
    let related_flow = best_flow_for_process(process.pid, context)?;
    let command_line = process.command_line.as_deref().unwrap_or_default();
    let command_lower = command_line.to_ascii_lowercase();
    let mut patterns = vec![
        "ide_helper_runtime".to_string(),
        "ide_parent_process_chain".to_string(),
    ];
    let mut evidence_items = vec![
        evidence("process", &process.name, 0.78),
        evidence("parent_process", &parent.name, 0.82),
        evidence("destination", &related_flow.destination(), 0.74),
    ];
    if !command_line.is_empty() {
        evidence_items.push(evidence("command_line", command_line, 0.8));
    }
    if let Some(path) = &process.path {
        evidence_items.push(evidence("repo_or_runtime_path", path, 0.66));
    }
    if contains_any(
        &command_lower,
        &["openai", "anthropic", "agent", "tool", "mcp", "copilot"],
    ) {
        patterns.push("helper_command_line_model_ref".to_string());
    }

    Some(DetectionCandidate {
        classification: "ide_ai_assistant".to_string(),
        application_category: Some("developer_tool".to_string()),
        risk_signal: None,
        process_guid: Some(process.process_guid.clone()),
        flow_id: Some(related_flow.flow_id),
        agent_likelihood: 0.72,
        confidence: 0.74,
        risk_score: 46,
        patterns,
        evidence: evidence_items,
        recommended_action: "review".to_string(),
        title: "IDE-launched AI helper runtime observed".to_string(),
        description: "A developer tool launched a helper runtime that reached an external destination; review for IDE AI or extension traffic."
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
        application_category: Some("script_or_agent_runtime".to_string()),
        risk_signal: None,
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

    let risk_signal = if command_lower.contains("-encodedcommand")
        || command_lower.contains("frombase64string")
    {
        Some("suspicious_automation".to_string())
    } else {
        None
    };

    Some(DetectionCandidate {
        classification: "shell_automation".to_string(),
        application_category: Some("shell_automation".to_string()),
        risk_signal,
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
    let ai_ips = ai_resolved_ips(context);
    let mut correlated_flow: Option<(&ProcessObservation, &FlowObservation)> = None;
    for process in context.processes.values() {
        if !is_browser_process(process) {
            continue;
        }
        let Some(flows) = context.flows_by_pid.get(&process.pid) else {
            continue;
        };
        for flow in flows {
            let hostname_ai = flow
                .remote_hostname
                .as_deref()
                .is_some_and(|host| is_ai_destination(host));
            let ip_hit = flow
                .remote_ip
                .parse::<IpAddr>()
                .ok()
                .is_some_and(|ip| ai_ips.contains(&ip.to_string()))
                || ai_ips.contains(&flow.remote_ip);
            if hostname_ai || ip_hit {
                correlated_flow = Some((process, flow));
                break;
            }
        }
        if correlated_flow.is_some() {
            break;
        }
    }

    let browser = context
        .processes
        .values()
        .find(|process| is_browser_process(process));
    let correlated_browser_guid = correlated_flow
        .as_ref()
        .map(|(browser_proc, _)| browser_proc.process_guid.clone());

    let mut patterns = vec!["ai_destination_dns".to_string()];
    let mut evidence_items = vec![evidence("dns_query", &ai_dns.query, 0.75)];
    if !ai_dns.answers.is_empty() {
        evidence_items.push(evidence("dns_answer", &ai_dns.answers.join(","), 0.55));
    }
    let (flow_id, confidence, agent_likelihood, description) = if let Some((browser_proc, flow)) =
        correlated_flow
    {
        patterns.push("browser_flow_to_ai_destination".to_string());
        evidence_items.push(evidence("browser_process", &browser_proc.name, 0.82));
        evidence_items.push(evidence("correlated_flow", &flow.destination(), 0.84));
        (
                Some(flow.flow_id.clone()),
                0.78_f32,
                0.34_f32,
                "Browser process network flow correlated to an AI-related DNS answer or hostname; monitor-only (not proof of an autonomous host agent)."
                    .to_string(),
            )
    } else if let Some(browser_proc) = browser {
        patterns.push("dns_without_flow_correlation".to_string());
        evidence_items.push(evidence("browser_process", &browser_proc.name, 0.55));
        evidence_items.push(evidence(
            "inference_note",
            "No matching browser flow in this batch; DNS-only inference.",
            0.5,
        ));
        (
                None,
                0.52_f32,
                0.18_f32,
                "AI-related domain in DNS cache without a correlated browser flow in this snapshot; treat as weak context."
                    .to_string(),
            )
    } else {
        patterns.push("dns_without_browser_process".to_string());
        evidence_items.push(evidence(
            "inference_note",
            "No browser process in snapshot; DNS-only.",
            0.45,
        ));
        (
            None,
            0.41_f32,
            0.12_f32,
            "AI-related DNS observation without a browser process in this snapshot.".to_string(),
        )
    };

    Some(DetectionCandidate {
        classification: "browser_ai_usage".to_string(),
        application_category: Some("browser".to_string()),
        risk_signal: None,
        process_guid: correlated_browser_guid.or_else(|| browser.map(|p| p.process_guid.clone())),
        flow_id,
        agent_likelihood,
        confidence,
        risk_score: 18,
        patterns,
        evidence: evidence_items,
        recommended_action: "monitor".to_string(),
        title: "Browser AI destination observed".to_string(),
        description,
    })
}

fn is_browser_process(process: &ProcessObservation) -> bool {
    matches!(
        process.name.to_ascii_lowercase().as_str(),
        "chrome.exe" | "msedge.exe" | "firefox.exe" | "brave.exe"
    )
}

fn ai_resolved_ips(context: &DetectionContext) -> BTreeSet<String> {
    let mut out = BTreeSet::new();
    for dns in &context.dns_queries {
        if !is_ai_destination(&dns.query) {
            continue;
        }
        for answer in &dns.answers {
            if answer.parse::<IpAddr>().is_ok() {
                out.insert(answer.clone());
            }
        }
    }
    out
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
            process_state_path: "/tmp/events.jsonl.process-state.json".into(),
            collect_command_line: true,
            controller_url: None,
            detection_packs_enabled: false,
            detection_pack_cache: None,
            detection_pack_public_key: None,
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
                    process_path: None,
                    user: None,
                    protocol: "tcp".to_string(),
                    direction: "outbound".to_string(),
                    local_ip: "10.0.0.2".to_string(),
                    local_port: Some(50000),
                    remote_ip: "203.0.113.10".to_string(),
                    remote_port: Some(443),
                    remote_hostname: Some("api.model-gateway.lab".to_string()),
                    connection_state: Some("ESTABLISHED".to_string()),
                    bytes_sent: None,
                    bytes_received: None,
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
    fn detects_ide_helper_runtime_with_parent_chain() {
        let config = config();
        let events = vec![
            AegisEvent::new(
                &config,
                "aegis.process.started",
                SystemTime::now(),
                EventPayload::ProcessStarted {
                    process_guid: "proc-ide-parent".to_string(),
                    parent_process_guid: None,
                    pid: 500,
                    ppid: None,
                    name: "cursor.exe".to_string(),
                    path: None,
                    command_line: None,
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
                "aegis.process.started",
                SystemTime::now(),
                EventPayload::ProcessStarted {
                    process_guid: "proc-node-helper".to_string(),
                    parent_process_guid: Some("proc-ide-parent".to_string()),
                    pid: 501,
                    ppid: Some(500),
                    name: "node.exe".to_string(),
                    path: Some(r"C:\Program Files\node\node.exe".to_string()),
                    command_line: Some("node extensionHost.js".to_string()),
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
                    flow_id: "flow-ide".to_string(),
                    process_guid: Some("proc-node-helper".to_string()),
                    pid: Some(501),
                    process_name: Some("node.exe".to_string()),
                    process_path: None,
                    user: None,
                    protocol: "tcp".to_string(),
                    direction: "outbound".to_string(),
                    local_ip: "10.0.0.2".to_string(),
                    local_port: Some(51000),
                    remote_ip: "203.0.113.20".to_string(),
                    remote_port: Some(443),
                    remote_hostname: Some("api.openai.com".to_string()),
                    connection_state: Some("ESTABLISHED".to_string()),
                    bytes_sent: None,
                    bytes_received: None,
                    attribution_method: "test".to_string(),
                    attribution_confidence: 0.9,
                },
            ),
        ];

        let detections = detect_ai_agent_activity(&config, &events);

        assert_eq!(detections.len(), 2);
        match &detections[0].payload {
            EventPayload::AgentDetected {
                classification,
                application_category,
                ..
            } => {
                assert_eq!(classification, "ide_ai_assistant");
                assert_eq!(application_category.as_deref(), Some("developer_tool"));
            }
            _ => panic!("expected agent detection"),
        }
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
