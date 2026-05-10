//! Data-only detection_pack.v1 rule evaluation (aligned with backend `eval` package).

use std::collections::HashMap;
use std::time::{Duration, Instant};

use serde_json::Value;

use crate::event::{AegisEvent, EventPayload};

/// Parsed visibility batch for one evaluation pass.
#[derive(Debug, Default)]
pub struct EvalBatch {
    processes_by_guid: HashMap<String, ProcessObs>,
    dns: Vec<DnsObs>,
    flows: Vec<FlowObs>,
}

#[derive(Debug, Clone)]
struct ProcessObs {
    name: String,
    command_line: String,
    path: String,
    parent_process_guid: String,
}

#[derive(Debug, Clone)]
struct DnsObs {
    query: String,
    _answers: Vec<String>,
}

#[derive(Debug, Clone)]
struct FlowObs {
    flow_id: String,
    process_guid: String,
    remote_port: Option<u16>,
    remote_hostname: String,
}

/// Optional correlation hints when a clause matches.
#[derive(Debug, Default, Clone)]
pub struct MatchHint {
    pub process_guid: Option<String>,
    pub flow_id: Option<String>,
}

impl MatchHint {
    fn merge(&mut self, other: &MatchHint) {
        if self.process_guid.is_none() {
            self.process_guid.clone_from(&other.process_guid);
        }
        if self.flow_id.is_none() {
            self.flow_id.clone_from(&other.flow_id);
        }
    }
}

/// Build an evaluator batch from visibility events (no detection outputs).
pub fn build_batch(events: &[AegisEvent]) -> EvalBatch {
    let mut b = EvalBatch::default();
    for ev in events {
        match &ev.payload {
            EventPayload::ProcessStarted {
                process_guid,
                parent_process_guid,
                name,
                path,
                command_line,
                ..
            } => {
                if process_guid.is_empty() {
                    continue;
                }
                let mut p = ProcessObs {
                    name: name.clone(),
                    command_line: String::new(),
                    path: String::new(),
                    parent_process_guid: parent_process_guid.clone().unwrap_or_default(),
                };
                if let Some(cl) = command_line {
                    p.command_line = cl.clone();
                }
                if let Some(pa) = path {
                    p.path = pa.clone();
                }
                b.processes_by_guid.insert(process_guid.clone(), p);
            }
            EventPayload::DnsObserved { query, answers, .. } => {
                if !query.is_empty() {
                    b.dns.push(DnsObs {
                        query: query.clone(),
                        _answers: answers.clone(),
                    });
                }
            }
            EventPayload::FlowStarted {
                flow_id,
                process_guid,
                remote_port,
                remote_hostname,
                ..
            } => {
                let mut fg = FlowObs {
                    flow_id: flow_id.clone(),
                    process_guid: process_guid.clone().unwrap_or_default(),
                    remote_port: *remote_port,
                    remote_hostname: String::new(),
                };
                if let Some(h) = remote_hostname {
                    fg.remote_hostname = h.clone();
                }
                b.flows.push(fg);
            }
            _ => {}
        }
    }
    b
}

struct LimitState {
    max_rules: usize,
    max_string_cmp: usize,
    max_depth: usize,
    max_clauses: usize,
    max_wall: Duration,
    start: Instant,
    string_cmp: usize,
    clauses: usize,
    rules_done: usize,
}

impl LimitState {
    fn new(limits: &Value) -> Result<Self, String> {
        let lo = limits
            .as_object()
            .ok_or_else(|| "evaluator_limits not object".to_string())?;
        let max_rules =
            lo.get("max_rules_evaluated_per_batch")
                .and_then(|v| v.as_u64())
                .ok_or_else(|| "max_rules_evaluated_per_batch".to_string())? as usize;
        let max_string_cmp =
            lo.get("max_string_comparisons_per_rule")
                .and_then(|v| v.as_u64())
                .ok_or_else(|| "max_string_comparisons_per_rule".to_string())? as usize;
        let max_depth = lo
            .get("max_clause_depth")
            .and_then(|v| v.as_u64())
            .ok_or_else(|| "max_clause_depth".to_string())? as usize;
        let max_clauses = lo
            .get("max_clauses_per_rule")
            .and_then(|v| v.as_u64())
            .ok_or_else(|| "max_clauses_per_rule".to_string())? as usize;
        let max_wall_ms = lo
            .get("max_wall_time_ms_per_batch")
            .and_then(|v| v.as_u64())
            .ok_or_else(|| "max_wall_time_ms_per_batch".to_string())?;
        Ok(Self {
            max_rules,
            max_string_cmp,
            max_depth,
            max_clauses,
            max_wall: Duration::from_millis(max_wall_ms),
            start: Instant::now(),
            string_cmp: 0,
            clauses: 0,
            rules_done: 0,
        })
    }

    fn bump_str(&mut self, n: usize) -> Result<(), String> {
        self.string_cmp = self.string_cmp.saturating_add(n);
        if self.string_cmp > self.max_string_cmp {
            return Err("max_string_comparisons_per_rule exceeded".to_string());
        }
        Ok(())
    }

    fn bump_clause(&mut self) -> Result<(), String> {
        self.clauses = self.clauses.saturating_add(1);
        if self.clauses > self.max_clauses {
            return Err("max_clauses_per_rule exceeded".to_string());
        }
        Ok(())
    }

    fn tick_time(&self) -> Result<(), String> {
        if self.start.elapsed() > self.max_wall {
            return Err("max_wall_time_ms_per_batch exceeded".to_string());
        }
        Ok(())
    }
}

/// One rule that matched, with hints for evidence attachment.
#[derive(Debug)]
pub struct RuleMatch {
    /// Full rule object from the pack.
    pub rule: Value,
    pub process_guid: Option<String>,
    pub flow_id: Option<String>,
}

/// Evaluate rules in priority order until limits hit; returns all matches.
pub fn evaluate_pack(pack: &Value, batch: &EvalBatch) -> Result<Vec<RuleMatch>, String> {
    let limits = pack
        .get("evaluator_limits")
        .ok_or_else(|| "evaluator_limits missing".to_string())?;
    let mut st = LimitState::new(limits)?;
    let rules = pack
        .get("rules")
        .and_then(|v| v.as_array())
        .ok_or_else(|| "rules missing".to_string())?;

    let mut indexed: Vec<(u64, usize)> = rules
        .iter()
        .enumerate()
        .map(|(i, r)| {
            let pr = r
                .get("priority")
                .and_then(|v| v.as_u64())
                .unwrap_or(1_000_000);
            (pr, i)
        })
        .collect();
    indexed.sort_by(|a, b| a.0.cmp(&b.0));

    let mut out = Vec::new();
    for (_, idx) in indexed {
        st.tick_time()?;
        if st.rules_done >= st.max_rules {
            break;
        }
        let rule = rules
            .get(idx)
            .ok_or_else(|| "rule index out of range".to_string())?;
        st.rules_done += 1;
        st.string_cmp = 0;
        st.clauses = 0;
        let clause = rule.get("match").cloned();
        let Some(clause) = clause else {
            continue;
        };
        let (matched, hint) = eval_clause(&clause, batch, &mut st, 1)?;
        if matched {
            out.push(RuleMatch {
                rule: rule.clone(),
                process_guid: hint.process_guid,
                flow_id: hint.flow_id,
            });
        }
    }
    Ok(out)
}

fn eval_clause(
    clause: &Value,
    batch: &EvalBatch,
    st: &mut LimitState,
    depth: usize,
) -> Result<(bool, MatchHint), String> {
    st.bump_clause()?;
    st.tick_time()?;
    if depth > st.max_depth {
        return Err("max_clause_depth exceeded".to_string());
    }
    let obj = clause
        .as_object()
        .ok_or_else(|| "clause not object".to_string())?;
    if obj.contains_key("op") {
        return eval_group(clause, batch, st, depth);
    }
    if obj.contains_key("process") {
        return eval_process_leaf(clause, batch, st);
    }
    if obj.contains_key("dns") {
        return eval_dns_leaf(clause, batch, st);
    }
    if obj.contains_key("flow") {
        return eval_flow_leaf(clause, batch, st);
    }
    if obj.contains_key("sase_component") {
        return eval_sase_leaf(clause, batch, st);
    }
    if obj.contains_key("browser_extension") {
        return eval_ext_leaf(clause, batch, st);
    }
    Err("unknown match clause shape".to_string())
}

fn eval_group(
    clause: &Value,
    batch: &EvalBatch,
    st: &mut LimitState,
    depth: usize,
) -> Result<(bool, MatchHint), String> {
    let obj = clause.as_object().ok_or_else(|| "group".to_string())?;
    let op = obj.get("op").and_then(|v| v.as_str()).unwrap_or("");
    let raw_of = obj.get("of").and_then(|v| v.as_array());
    let Some(raw_of) = raw_of else {
        return Err("group missing of".to_string());
    };
    match op {
        "all_of" => {
            let mut hint = MatchHint::default();
            for ch in raw_of {
                let (ok, h) = eval_clause(ch, batch, st, depth.saturating_add(1))?;
                if !ok {
                    return Ok((false, MatchHint::default()));
                }
                hint.merge(&h);
            }
            Ok((true, hint))
        }
        "any_of" => {
            let mut min = 1usize;
            if let Some(v) = obj.get("min_match").and_then(|x| x.as_u64()) {
                min = v as usize;
                if min < 1 {
                    min = 1;
                }
            }
            let mut n = 0usize;
            let mut hint = MatchHint::default();
            for ch in raw_of {
                let (ok, h) = eval_clause(ch, batch, st, depth.saturating_add(1))?;
                if ok {
                    n = n.saturating_add(1);
                    hint.merge(&h);
                }
            }
            Ok((n >= min, hint))
        }
        _ => Err(format!("unknown group op {op}")),
    }
}

fn eval_process_leaf(
    clause: &Value,
    batch: &EvalBatch,
    st: &mut LimitState,
) -> Result<(bool, MatchHint), String> {
    let pm = clause
        .get("process")
        .and_then(|v| v.as_object())
        .ok_or_else(|| "process leaf".to_string())?;
    let ci = pm
        .get("case_insensitive")
        .and_then(|v| v.as_bool())
        .unwrap_or(true);
    for (guid, proc) in &batch.processes_by_guid {
        if match_process(batch, pm, proc, ci, st)? {
            return Ok((
                true,
                MatchHint {
                    process_guid: Some(guid.clone()),
                    flow_id: None,
                },
            ));
        }
    }
    Ok((false, MatchHint::default()))
}

fn match_process(
    batch: &EvalBatch,
    pm: &serde_json::Map<String, Value>,
    proc: &ProcessObs,
    ci: bool,
    st: &mut LimitState,
) -> Result<bool, String> {
    let mut name = proc.name.as_str();
    let mut cmd = proc.command_line.as_str();
    let mut path = proc.path.as_str();
    let mut parent_name = "";
    if !proc.parent_process_guid.is_empty() {
        if let Some(par) = batch.processes_by_guid.get(&proc.parent_process_guid) {
            parent_name = par.name.as_str();
        }
    }
    let name_owned: String;
    let cmd_owned: String;
    let path_owned: String;
    let parent_owned: String;
    if ci {
        name_owned = name.to_ascii_lowercase();
        cmd_owned = cmd.to_ascii_lowercase();
        path_owned = path.to_ascii_lowercase();
        parent_owned = parent_name.to_ascii_lowercase();
        name = name_owned.as_str();
        cmd = cmd_owned.as_str();
        path = path_owned.as_str();
        parent_name = parent_owned.as_str();
    }
    if let Some(Value::Array(arr)) = pm.get("executable_names_any") {
        if !contains_string_any(name, arr, ci, st)? {
            return Ok(false);
        }
    }
    if let Some(Value::Array(arr)) = pm.get("executable_name_contains_any") {
        if !substring_any(name, arr, true, st)? {
            return Ok(false);
        }
    }
    if let Some(Value::Array(arr)) = pm.get("command_line_contains_any") {
        if !substring_any(cmd, arr, true, st)? {
            return Ok(false);
        }
    }
    if let Some(Value::Array(arr)) = pm.get("parent_executable_names_any") {
        if !contains_string_any(parent_name, arr, ci, st)? {
            return Ok(false);
        }
    }
    if let Some(Value::Array(arr)) = pm.get("process_path_contains_any") {
        if !substring_any(path, arr, true, st)? {
            return Ok(false);
        }
    }
    Ok(true)
}

fn eval_dns_leaf(
    clause: &Value,
    batch: &EvalBatch,
    st: &mut LimitState,
) -> Result<(bool, MatchHint), String> {
    let dm = clause
        .get("dns")
        .and_then(|v| v.as_object())
        .ok_or_else(|| "dns leaf".to_string())?;
    let Some(Value::Array(arr)) = dm.get("query_contains_any") else {
        return Ok((false, MatchHint::default()));
    };
    for d in &batch.dns {
        let q = d.query.to_ascii_lowercase();
        if substring_any(&q, arr, true, st)? {
            return Ok((true, MatchHint::default()));
        }
    }
    Ok((false, MatchHint::default()))
}

fn eval_flow_leaf(
    clause: &Value,
    batch: &EvalBatch,
    st: &mut LimitState,
) -> Result<(bool, MatchHint), String> {
    let fm = clause
        .get("flow")
        .and_then(|v| v.as_object())
        .ok_or_else(|| "flow leaf".to_string())?;
    if fm.get("has_any_flow").and_then(|v| v.as_bool()) == Some(true) {
        if let Some(fl) = batch.flows.first() {
            return Ok((
                true,
                MatchHint {
                    process_guid: if fl.process_guid.is_empty() {
                        None
                    } else {
                        Some(fl.process_guid.clone())
                    },
                    flow_id: if fl.flow_id.is_empty() {
                        None
                    } else {
                        Some(fl.flow_id.clone())
                    },
                },
            ));
        }
        return Ok((false, MatchHint::default()));
    }
    for fl in &batch.flows {
        if match_flow_map(fm, fl, st)? {
            return Ok((
                true,
                MatchHint {
                    process_guid: if fl.process_guid.is_empty() {
                        None
                    } else {
                        Some(fl.process_guid.clone())
                    },
                    flow_id: if fl.flow_id.is_empty() {
                        None
                    } else {
                        Some(fl.flow_id.clone())
                    },
                },
            ));
        }
    }
    Ok((false, MatchHint::default()))
}

fn match_flow_map(
    fm: &serde_json::Map<String, Value>,
    fl: &FlowObs,
    st: &mut LimitState,
) -> Result<bool, String> {
    if let Some(Value::Array(arr)) = fm.get("remote_ports_any") {
        if !arr.is_empty() {
            let Some(p) = fl.remote_port else {
                return Ok(false);
            };
            let mut found = false;
            for x in arr {
                let Some(pv) = x.as_u64().map(|v| v as u16) else {
                    continue;
                };
                if pv == p {
                    found = true;
                    break;
                }
            }
            if !found {
                return Ok(false);
            }
        }
    }
    if let Some(Value::Array(arr)) = fm.get("remote_host_contains_any") {
        let h = fl.remote_hostname.to_ascii_lowercase();
        if !substring_any(&h, arr, true, st)? {
            return Ok(false);
        }
    }
    Ok(true)
}

fn eval_sase_leaf(
    clause: &Value,
    batch: &EvalBatch,
    st: &mut LimitState,
) -> Result<(bool, MatchHint), String> {
    let _ = clause;
    let _ = batch;
    let _ = st;
    Ok((false, MatchHint::default()))
}

fn eval_ext_leaf(
    clause: &Value,
    batch: &EvalBatch,
    st: &mut LimitState,
) -> Result<(bool, MatchHint), String> {
    let _ = clause;
    let _ = batch;
    let _ = st;
    Ok((false, MatchHint::default()))
}

fn substring_any(
    s: &str,
    needles: &[Value],
    lower_needles: bool,
    st: &mut LimitState,
) -> Result<bool, String> {
    for n in needles {
        let Some(ns) = n.as_str() else {
            continue;
        };
        st.bump_str(1)?;
        let needle = if lower_needles {
            ns.to_ascii_lowercase()
        } else {
            ns.to_string()
        };
        if !needle.is_empty() && s.contains(needle.as_str()) {
            return Ok(true);
        }
    }
    Ok(false)
}

fn contains_string_any(
    s: &str,
    names: &[Value],
    ci: bool,
    st: &mut LimitState,
) -> Result<bool, String> {
    for n in names {
        let Some(ns) = n.as_str() else {
            continue;
        };
        st.bump_str(1)?;
        let (a, b) = if ci {
            (s.to_ascii_lowercase(), ns.to_ascii_lowercase())
        } else {
            (s.to_string(), ns.to_string())
        };
        if a == b {
            return Ok(true);
        }
    }
    Ok(false)
}

#[cfg(test)]
mod tests {
    #![allow(clippy::unwrap_used)]

    use super::*;
    use crate::config::AgentConfig;
    use crate::event::{AegisEvent, EventPayload};
    use serde_json::json;
    use std::time::SystemTime;

    fn sample_pack_with_rule(match_clause: Value) -> Value {
        json!({
            "schema_version": "detection_pack.v1",
            "pack_id": "lab",
            "pack_version": "1.0.0",
            "mode": "observe",
            "supported_os": ["windows"],
            "min_agent_version": "0.0.1",
            "author": "test",
            "signature": {"algorithm":"ed25519","key_id":"k","value_b64":"AA=="},
            "evaluator_limits": {
                "max_wall_time_ms_per_batch": 60000,
                "max_heap_bytes": 65536,
                "max_rules_evaluated_per_batch": 100,
                "max_cpu_percent_soft": 50.0,
                "max_string_comparisons_per_rule": 10000,
                "max_clause_depth": 8,
                "max_clauses_per_rule": 64
            },
            "rules": [{
                "rule_id": "r1",
                "priority": 1,
                "title": "t",
                "description": "d",
                "classification": "c",
                "pattern_tags": ["p"],
                "agent_likelihood": 0.5,
                "confidence": 0.6,
                "risk_score": 10,
                "recommended_action": "review",
                "required_evidence": ["process"],
                "match": match_clause
            }]
        })
    }

    #[test]
    fn eval_matches_process_name() {
        let cfg = AgentConfig {
            agent_id: "a".to_string(),
            device_id: "d".to_string(),
            sensor_version: "1.0.0".to_string(),
            backend_url: None,
            event_spool: "/tmp/e".into(),
            process_state_path: "/tmp/e.process-state.json".into(),
            collect_command_line: false,
            controller_url: None,
            detection_packs_enabled: false,
            detection_pack_cache: None,
            detection_pack_public_key: None,
        };
        let events = vec![AegisEvent::new(
            &cfg,
            "aegis.process.started",
            SystemTime::UNIX_EPOCH,
            EventPayload::ProcessStarted {
                process_guid: "pg-1".to_string(),
                parent_process_guid: None,
                pid: 7,
                ppid: None,
                name: "bash".to_string(),
                path: None,
                command_line: None,
                user: None,
                logon_session_id: None,
                integrity_level: None,
                sha256: None,
                publisher: None,
                collection_method: "test".to_string(),
            },
        )];
        let batch = build_batch(&events);
        let pack = sample_pack_with_rule(json!({
            "process": { "executable_names_any": ["bash"] }
        }));
        let m = evaluate_pack(&pack, &batch).unwrap();
        assert_eq!(m.len(), 1);
        assert_eq!(m[0].process_guid.as_deref(), Some("pg-1"));
    }
}
