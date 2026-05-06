//! Controller fetch, verify, cache, evaluate, and status reporting (WO-DET-003 contract).

use std::path::Path;
use std::time::SystemTime;

use serde::Deserialize;
use serde_json::json;
use serde_json::Value;

use super::cache::{
    load_active, load_previous, promote_and_write_active, write_active, PackCacheMeta,
};
use super::eval::{build_batch, evaluate_pack, RuleMatch};
use super::http::HttpBase;
use super::schema_check::validate_pack_for_windows;
use super::verify::{
    sha256_hex, signature_value_b64, verify_content_sha256_header, verify_ed25519_signature,
};

use crate::config::AgentConfig;
use crate::event::{AegisEvent, DetectionEvidence, EventPayload};

/// Run one dynamic-pack cycle: discover, verify, cache, evaluate, report status.
pub fn run_dynamic_pack_pipeline(
    config: &AgentConfig,
    visibility_events: &[AegisEvent],
) -> Vec<AegisEvent> {
    if !config.detection_packs_enabled {
        return Vec::new();
    }
    let Some(pk) = config.detection_pack_public_key else {
        eprintln!(
            "aegis-windows-agent: AEGIS_DETECTION_PACKS_ENABLED but no AEGIS_DETECTION_PACK_PUBLIC_KEY; skipping dynamic packs"
        );
        return Vec::new();
    };

    let default_pack_dir = super::cache::default_cache_dir(&config.event_spool);
    let cache_dir = config
        .detection_pack_cache
        .as_deref()
        .unwrap_or(default_pack_dir.as_path());

    let active_before = match load_active(cache_dir) {
        Ok(v) => v,
        Err(e) => {
            eprintln!("aegis-windows-agent: detection pack cache read: {e}");
            None
        }
    };

    let Some(controller) = config.controller_url.as_deref() else {
        eprintln!("aegis-windows-agent: AEGIS_CONTROLLER_URL unset; using verified cache only");
        let ctx = RolloutContext::stale("no_controller_url", active_before.as_ref());
        return evaluate_cached_only(config, visibility_events, pk, cache_dir, &ctx);
    };

    let Ok(http) = HttpBase::parse(controller) else {
        eprintln!("aegis-windows-agent: invalid AEGIS_CONTROLLER_URL");
        let ctx = RolloutContext::stale("bad_controller_url", active_before.as_ref());
        return evaluate_cached_only(config, visibility_events, pk, cache_dir, &ctx);
    };

    let latest_path = format!(
        "/detection-packs/latest?os=windows&agent_version={}",
        url_encode(&config.sensor_version)
    );
    let latest = match http.get(&latest_path) {
        Ok((200, _hdr, body)) => match serde_json::from_slice::<LatestResponse>(&body) {
            Ok(l) => Some(l),
            Err(e) => {
                eprintln!("aegis-windows-agent: latest pack json: {e}");
                None
            }
        },
        Ok((code, _, _)) => {
            eprintln!("aegis-windows-agent: GET latest returned HTTP {code}");
            None
        }
        Err(e) => {
            eprintln!("aegis-windows-agent: GET latest: {e}");
            None
        }
    };

    let Some(latest) = latest else {
        let ctx = RolloutContext::stale(
            "controller_unreachable_or_bad_response",
            active_before.as_ref(),
        );
        let _ = post_status(config, &http, &ctx);
        return evaluate_cached_only(config, visibility_events, pk, cache_dir, &ctx);
    };

    let artifact_path = normalize_artifact_path(&latest.artifact_url);
    let use_cached = active_before.as_ref().is_some_and(|c| {
        c.meta.artifact_id == latest.artifact_id
            && c.meta.sha256.eq_ignore_ascii_case(&latest.sha256)
    });

    let pack_and_meta = if use_cached {
        let Some(c) = active_before.as_ref() else {
            return Vec::new();
        };
        match serde_json::from_slice::<Value>(&c.bytes) {
            Ok(v) => match verify_cached_pack(&v, &c.meta, &c.bytes, &pk, config) {
                Ok(()) => Some((v, c.meta.clone())),
                Err(e) => {
                    eprintln!("aegis-windows-agent: cached pack no longer valid: {e}");
                    None
                }
            },
            Err(e) => {
                eprintln!("aegis-windows-agent: cached pack json: {e}");
                None
            }
        }
    } else {
        match http.get(&artifact_path) {
            Ok((200, hdr, body)) => match verify_downloaded(&body, &hdr, &pk, &latest, config) {
                Ok((v, meta)) => {
                    if let Err(e) = if let Some(prev) = active_before.as_ref() {
                        promote_and_write_active(cache_dir, Some(prev), &body, meta.clone())
                    } else {
                        write_active(cache_dir, &body, meta.clone())
                    } {
                        eprintln!("aegis-windows-agent: cache write: {e}");
                    }
                    let ctx = RolloutContext::applied(&meta, active_before.as_ref());
                    let _ = post_status(config, &http, &ctx);
                    Some((v, meta))
                }
                Err(ctx) => {
                    let _ = post_status(config, &http, &ctx);
                    eprintln!(
                        "aegis-windows-agent: pack rejected: {}",
                        ctx.reason_detail.as_str()
                    );
                    return evaluate_cached_only(config, visibility_events, pk, cache_dir, &ctx);
                }
            },
            Ok((406, _, _)) => {
                let ctx = RolloutContext::incompatible(
                    "artifact_406",
                    "pack incompatible with os/agent_version",
                    active_before.as_ref(),
                );
                let _ = post_status(config, &http, &ctx);
                return evaluate_cached_only(config, visibility_events, pk, cache_dir, &ctx);
            }
            Ok((code, _, _)) => {
                eprintln!("aegis-windows-agent: GET artifact HTTP {code}");
                let ctx =
                    RolloutContext::stale(&format!("artifact_http_{code}"), active_before.as_ref());
                let _ = post_status(config, &http, &ctx);
                return evaluate_cached_only(config, visibility_events, pk, cache_dir, &ctx);
            }
            Err(e) => {
                eprintln!("aegis-windows-agent: GET artifact: {e}");
                let ctx = RolloutContext::stale("artifact_unreachable", active_before.as_ref());
                let _ = post_status(config, &http, &ctx);
                return evaluate_cached_only(config, visibility_events, pk, cache_dir, &ctx);
            }
        }
    };

    let Some((pack_value, meta)) = pack_and_meta else {
        let ctx = RolloutContext::stale("cache_invalid_or_empty", active_before.as_ref());
        let _ = post_status(config, &http, &ctx);
        return evaluate_cached_only(config, visibility_events, pk, cache_dir, &ctx);
    };

    if use_cached {
        let ctx = RolloutContext::applied(&meta, active_before.as_ref());
        let _ = post_status(config, &http, &ctx);
    }

    evaluate_verified_pack(config, &pack_value, visibility_events)
}

fn verify_cached_pack(
    pack: &Value,
    meta: &PackCacheMeta,
    artifact_bytes: &[u8],
    pk: &[u8; 32],
    config: &AgentConfig,
) -> Result<(), String> {
    let sig = signature_value_b64(pack)?;
    verify_ed25519_signature(pack, pk, &sig)?;
    validate_pack_for_windows(pack, &config.sensor_version, SystemTime::now())?;
    if !sha256_hex(artifact_bytes).eq_ignore_ascii_case(meta.sha256.trim()) {
        return Err("cached pack sha256 mismatch vs meta".to_string());
    }
    Ok(())
}

fn evaluate_cached_only(
    config: &AgentConfig,
    visibility_events: &[AegisEvent],
    pk: [u8; 32],
    cache_dir: &Path,
    rollout: &RolloutContext,
) -> Vec<AegisEvent> {
    if let Some(ref url) = config.controller_url {
        if let Ok(http) = HttpBase::parse(url) {
            let _ = post_status(config, &http, rollout);
        }
    }
    let Some((pack, _meta)) = load_verified_pack_for_eval(cache_dir, config, &pk) else {
        return Vec::new();
    };
    evaluate_verified_pack(config, &pack, visibility_events)
}

fn load_verified_pack_for_eval(
    cache_dir: &Path,
    config: &AgentConfig,
    pk: &[u8; 32],
) -> Option<(Value, PackCacheMeta)> {
    let try_pack = |bytes: &[u8], meta: PackCacheMeta| {
        let v: Value = serde_json::from_slice(bytes).ok()?;
        verify_cached_pack(&v, &meta, bytes, pk, config).ok()?;
        Some((v, meta))
    };
    if let Ok(Some(c)) = load_active(cache_dir) {
        if let Some(r) = try_pack(&c.bytes, c.meta.clone()) {
            return Some(r);
        }
    }
    if let Ok(Some(p)) = load_previous(cache_dir) {
        if let Some(r) = try_pack(&p.bytes, p.meta.clone()) {
            return Some(r);
        }
    }
    None
}

fn verify_downloaded(
    body: &[u8],
    headers: &str,
    pk: &[u8; 32],
    latest: &LatestResponse,
    config: &AgentConfig,
) -> Result<(Value, PackCacheMeta), RolloutContext> {
    if let Err(e) = verify_content_sha256_header(headers, body) {
        return Err(RolloutContext::rejected(
            "hash_mismatch",
            &e,
            "mismatch",
            "invalid",
            "invalid",
            "incompatible",
            latest,
        ));
    }

    let pack_value: Value = match serde_json::from_slice(body) {
        Ok(v) => v,
        Err(e) => {
            return Err(RolloutContext::rejected(
                "json_parse",
                &format!("{e}"),
                "unknown",
                "unknown",
                "invalid",
                "incompatible",
                latest,
            ));
        }
    };

    let sig_b64 = match signature_value_b64(&pack_value) {
        Ok(s) => s,
        Err(e) => {
            return Err(RolloutContext::rejected(
                "unsigned_or_bad_signature_block",
                &e,
                "invalid",
                "unknown",
                "invalid",
                "incompatible",
                latest,
            ));
        }
    };

    if let Err(e) = verify_ed25519_signature(&pack_value, pk, &sig_b64) {
        return Err(RolloutContext::rejected(
            "signature_invalid",
            &e,
            "invalid",
            "match",
            "invalid",
            "incompatible",
            latest,
        ));
    }

    if let Err(e) =
        validate_pack_for_windows(&pack_value, &config.sensor_version, SystemTime::now())
    {
        let expired = e.contains("expired");
        if expired {
            return Err(RolloutContext::expired(&e, latest));
        }
        return Err(RolloutContext::rejected(
            "schema_or_policy",
            &e,
            "valid",
            "match",
            "invalid",
            "incompatible",
            latest,
        ));
    }

    let body_hash = sha256_hex(body);
    if !body_hash.eq_ignore_ascii_case(latest.sha256.trim()) {
        return Err(RolloutContext::rejected(
            "sha256_metadata_mismatch",
            "artifact body hash does not match latest.sha256",
            "valid",
            "mismatch",
            "valid",
            "incompatible",
            latest,
        ));
    }

    let meta = PackCacheMeta {
        artifact_id: latest.artifact_id.clone(),
        pack_id: latest.pack_id.clone(),
        pack_version: latest.pack_version.clone(),
        sha256: latest.sha256.clone(),
    };
    Ok((pack_value, meta))
}

fn evaluate_verified_pack(
    config: &AgentConfig,
    pack: &Value,
    visibility_events: &[AegisEvent],
) -> Vec<AegisEvent> {
    if let Err(e) = validate_pack_for_windows(pack, &config.sensor_version, SystemTime::now()) {
        eprintln!("aegis-windows-agent: pack policy check before eval: {e}");
        return Vec::new();
    }
    let pack_id = pack
        .get("pack_id")
        .and_then(|v| v.as_str())
        .unwrap_or("unknown");
    let pack_version = pack
        .get("pack_version")
        .and_then(|v| v.as_str())
        .unwrap_or("0.0.0");
    let batch = build_batch(visibility_events);
    let matches = match evaluate_pack(pack, &batch) {
        Ok(m) => m,
        Err(e) => {
            eprintln!("aegis-windows-agent: pack evaluation: {e}");
            return Vec::new();
        }
    };

    let mut out = Vec::new();
    for m in matches {
        out.extend(emit_rule_findings(config, pack_id, pack_version, &m));
    }
    out
}

fn emit_rule_findings(
    config: &AgentConfig,
    pack_id: &str,
    pack_version: &str,
    m: &RuleMatch,
) -> Vec<AegisEvent> {
    let rule = &m.rule;
    let rule_id = rule
        .get("rule_id")
        .and_then(|v| v.as_str())
        .unwrap_or("unknown");
    let classification = rule
        .get("classification")
        .and_then(|v| v.as_str())
        .unwrap_or("dynamic_pack")
        .to_string();
    let title = rule
        .get("title")
        .and_then(|v| v.as_str())
        .unwrap_or("Detection pack rule")
        .to_string();
    let description = rule
        .get("description")
        .and_then(|v| v.as_str())
        .unwrap_or("")
        .to_string();
    let agent_likelihood = rule
        .get("agent_likelihood")
        .and_then(|v| v.as_f64())
        .map(|x| x as f32)
        .unwrap_or(0.0);
    let confidence = rule
        .get("confidence")
        .and_then(|v| v.as_f64())
        .map(|x| x as f32)
        .unwrap_or(0.0);
    let risk_score = rule
        .get("risk_score")
        .and_then(|v| v.as_u64())
        .map(|x| (x.min(100)) as u8)
        .unwrap_or(0);
    let recommended_action = rule
        .get("recommended_action")
        .and_then(|v| v.as_str())
        .unwrap_or("review")
        .to_string();
    let patterns: Vec<String> = rule
        .get("pattern_tags")
        .and_then(|v| v.as_array())
        .map(|arr| {
            arr.iter()
                .filter_map(|x| x.as_str().map(String::from))
                .collect()
        })
        .unwrap_or_default();

    let mut evidence = build_required_evidence(rule, m);
    evidence.push(DetectionEvidence {
        evidence_type: "detection_pack_id".to_string(),
        value: truncate_evidence(pack_id),
        confidence: 1.0,
    });
    evidence.push(DetectionEvidence {
        evidence_type: "detection_pack_version".to_string(),
        value: truncate_evidence(pack_version),
        confidence: 1.0,
    });
    evidence.push(DetectionEvidence {
        evidence_type: "rule_id".to_string(),
        value: truncate_evidence(rule_id),
        confidence: 1.0,
    });

    let detection_id = format!(
        "det-{}-{}-{}-{}",
        config.device_id,
        pack_id,
        rule_id,
        stable_hash(rule_id)
    );
    let finding_id = format!("finding-{detection_id}");
    let severity = severity_for_score(risk_score);

    let det = AegisEvent::new(
        config,
        "aegis.agent.detected",
        SystemTime::now(),
        EventPayload::AgentDetected {
            detection_id: detection_id.clone(),
            process_guid: m.process_guid.clone(),
            flow_id: m.flow_id.clone(),
            classification,
            agent_likelihood,
            confidence,
            risk_score,
            detected_patterns: patterns.clone(),
            evidence: evidence.clone(),
            recommended_action: recommended_action.clone(),
        },
    );
    let finding = AegisEvent::new(
        config,
        "aegis.risk_finding.created",
        SystemTime::now(),
        EventPayload::RiskFindingCreated {
            finding_id,
            severity,
            risk_score,
            title,
            description,
            process_guid: m.process_guid.clone(),
            flow_id: m.flow_id.clone(),
            detection_id: Some(detection_id),
            evidence,
            recommended_action,
        },
    );
    vec![det, finding]
}

fn build_required_evidence(rule: &Value, m: &RuleMatch) -> Vec<DetectionEvidence> {
    let Some(arr) = rule.get("required_evidence").and_then(|v| v.as_array()) else {
        return Vec::new();
    };
    let mut out = Vec::new();
    for item in arr {
        let Some(et) = item.as_str() else {
            continue;
        };
        let (value, conf) = match et {
            "process" => (
                m.process_guid
                    .as_deref()
                    .unwrap_or("unknown-process")
                    .to_string(),
                0.75,
            ),
            "network_flow" => (
                m.flow_id.as_deref().unwrap_or("unknown-flow").to_string(),
                0.7,
            ),
            _ => (format!("{et}:matched"), 0.55),
        };
        out.push(DetectionEvidence {
            evidence_type: et.to_string(),
            value: truncate_evidence(&value),
            confidence: conf,
        });
    }
    out
}

fn truncate_evidence(value: &str) -> String {
    const MAX: usize = 512;
    if value.len() <= MAX {
        value.to_string()
    } else {
        format!("{}...[truncated]", &value[..MAX])
    }
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

#[derive(Debug, Deserialize)]
struct LatestResponse {
    pack_id: String,
    pack_version: String,
    artifact_id: String,
    sha256: String,
    artifact_url: String,
}

fn normalize_artifact_path(artifact_url: &str) -> String {
    if let Some(i) = artifact_url.find("/detection-packs/") {
        artifact_url[i..].to_string()
    } else if artifact_url.starts_with('/') {
        artifact_url.to_string()
    } else {
        format!("/{artifact_url}")
    }
}

fn url_encode(s: &str) -> String {
    use core::fmt::Write;
    let mut out = String::new();
    for b in s.as_bytes() {
        match b {
            b'A'..=b'Z' | b'a'..=b'z' | b'0'..=b'9' | b'-' | b'.' | b'_' | b'~' => {
                out.push(char::from(*b))
            }
            _ => {
                let _ = write!(&mut out, "%{b:02X}");
            }
        }
    }
    out
}

struct RolloutContext {
    rollout_state: &'static str,
    reason_codes: Vec<String>,
    reason_detail: String,
    active_pack_id: String,
    active_pack_version: String,
    previous_pack_id: String,
    previous_pack_version: String,
    signature_status: String,
    hash_status: String,
    schema_status: String,
    compatibility_status: String,
}

impl RolloutContext {
    fn applied(meta: &PackCacheMeta, prev: Option<&super::cache::CachedPack>) -> Self {
        let (prev_id, prev_ver) = prev
            .map(|p| (p.meta.pack_id.clone(), p.meta.pack_version.clone()))
            .unwrap_or_default();
        Self {
            rollout_state: "applied",
            reason_codes: Vec::new(),
            reason_detail: String::new(),
            active_pack_id: meta.pack_id.clone(),
            active_pack_version: meta.pack_version.clone(),
            previous_pack_id: prev_id,
            previous_pack_version: prev_ver,
            signature_status: "valid".to_string(),
            hash_status: "match".to_string(),
            schema_status: "valid".to_string(),
            compatibility_status: "ok".to_string(),
        }
    }

    fn rejected(
        code: &str,
        detail: &str,
        signature_status: &str,
        hash_status: &str,
        schema_status: &str,
        compatibility_status: &str,
        latest: &LatestResponse,
    ) -> Self {
        Self {
            rollout_state: "rejected",
            reason_codes: vec![code.to_string()],
            reason_detail: detail.to_string(),
            active_pack_id: latest.pack_id.clone(),
            active_pack_version: latest.pack_version.clone(),
            previous_pack_id: String::new(),
            previous_pack_version: String::new(),
            signature_status: signature_status.to_string(),
            hash_status: hash_status.to_string(),
            schema_status: schema_status.to_string(),
            compatibility_status: compatibility_status.to_string(),
        }
    }

    fn incompatible(code: &str, detail: &str, prev: Option<&super::cache::CachedPack>) -> Self {
        let (active_id, active_ver) = prev
            .map(|p| (p.meta.pack_id.clone(), p.meta.pack_version.clone()))
            .unwrap_or_default();
        Self {
            rollout_state: "incompatible",
            reason_codes: vec![code.to_string()],
            reason_detail: detail.to_string(),
            active_pack_id: active_id,
            active_pack_version: active_ver,
            previous_pack_id: String::new(),
            previous_pack_version: String::new(),
            signature_status: "unknown".to_string(),
            hash_status: "unknown".to_string(),
            schema_status: "invalid".to_string(),
            compatibility_status: "incompatible".to_string(),
        }
    }

    fn expired(detail: &str, latest: &LatestResponse) -> Self {
        Self {
            rollout_state: "expired",
            reason_codes: vec!["expired".to_string()],
            reason_detail: detail.to_string(),
            active_pack_id: latest.pack_id.clone(),
            active_pack_version: latest.pack_version.clone(),
            previous_pack_id: String::new(),
            previous_pack_version: String::new(),
            signature_status: "valid".to_string(),
            hash_status: "match".to_string(),
            schema_status: "valid".to_string(),
            compatibility_status: "incompatible".to_string(),
        }
    }

    fn stale(detail: &str, prev: Option<&super::cache::CachedPack>) -> Self {
        let (pid, pver) = prev
            .map(|p| (p.meta.pack_id.clone(), p.meta.pack_version.clone()))
            .unwrap_or_default();
        Self {
            rollout_state: "stale",
            reason_codes: vec!["controller_unreachable".to_string()],
            reason_detail: detail.to_string(),
            active_pack_id: pid,
            active_pack_version: pver,
            previous_pack_id: String::new(),
            previous_pack_version: String::new(),
            signature_status: "unknown".to_string(),
            hash_status: "unknown".to_string(),
            schema_status: "unknown".to_string(),
            compatibility_status: "unknown".to_string(),
        }
    }
}

fn post_status(config: &AgentConfig, http: &HttpBase, ctx: &RolloutContext) -> Result<(), String> {
    let path = format!(
        "/agents/{}/detection-pack-status",
        url_encode(&config.agent_id)
    );
    let body = json!({
        "device_id": config.device_id,
        "reported_agent_version": config.sensor_version,
        "rollout_state": ctx.rollout_state,
        "reason_codes": ctx.reason_codes,
        "reason_detail": ctx.reason_detail,
        "active_pack_id": ctx.active_pack_id,
        "active_pack_version": ctx.active_pack_version,
        "previous_pack_id": ctx.previous_pack_id,
        "previous_pack_version": ctx.previous_pack_version,
        "signature_status": ctx.signature_status,
        "hash_status": ctx.hash_status,
        "schema_status": ctx.schema_status,
        "compatibility_status": ctx.compatibility_status,
        "emit_visibility": false,
    });
    let body_str = body.to_string();
    let (code, _, resp) = http.post_json(&path, &body_str)?;
    if code == 200 || code == 204 {
        return Ok(());
    }
    Err(format!(
        "POST detection-pack-status HTTP {code}: {}",
        String::from_utf8_lossy(&resp)
    ))
}
