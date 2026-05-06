//! Structural and policy checks before evaluating a pack (observe-only, OS, semver, expiry).

use std::time::SystemTime;

use serde_json::Value;

/// Validate pack fields required for safe Windows evaluation.
pub fn validate_pack_for_windows(
    pack: &Value,
    agent_semver: &str,
    now: SystemTime,
) -> Result<(), String> {
    let obj = pack
        .as_object()
        .ok_or_else(|| "pack root must be object".to_string())?;

    if read_str(pack, "schema_version") != Some("detection_pack.v1") {
        return Err("schema_version must be detection_pack.v1".to_string());
    }

    if read_str(pack, "mode") != Some("observe") {
        return Err("pack mode must be observe".to_string());
    }

    let supported = obj
        .get("supported_os")
        .and_then(|v| v.as_array())
        .ok_or_else(|| "supported_os missing".to_string())?;
    let has_windows = supported
        .iter()
        .filter_map(|x| x.as_str())
        .any(|s| s == "windows");
    if !has_windows {
        return Err("supported_os must include windows".to_string());
    }

    let min_ver = read_str(pack, "min_agent_version")
        .ok_or_else(|| "min_agent_version missing".to_string())?;
    if !semver_gte(agent_semver, &min_ver) {
        return Err(format!(
            "agent version {agent_semver} is below min_agent_version {min_ver}"
        ));
    }

    if let Some(exp) = obj.get("expires_at") {
        if !exp.is_null() {
            let s = exp
                .as_str()
                .ok_or_else(|| "expires_at must be string or null".to_string())?;
            let exp_time =
                parse_rfc3339_utc(s).ok_or_else(|| "expires_at not parseable".to_string())?;
            if now >= exp_time {
                return Err("pack expired".to_string());
            }
        }
    }

    let rules = obj
        .get("rules")
        .and_then(|v| v.as_array())
        .ok_or_else(|| "rules must be array".to_string())?;
    if rules.is_empty() {
        return Err("rules must be non-empty".to_string());
    }
    if rules.len() > 500 {
        return Err("too many rules".to_string());
    }

    let limits = obj
        .get("evaluator_limits")
        .and_then(|v| v.as_object())
        .ok_or_else(|| "evaluator_limits missing".to_string())?;
    let max_rules = limits
        .get("max_rules_evaluated_per_batch")
        .and_then(|v| v.as_u64())
        .ok_or_else(|| "evaluator_limits.max_rules_evaluated_per_batch".to_string())?
        as usize;
    if max_rules == 0 || max_rules > 10_000 {
        return Err("invalid max_rules_evaluated_per_batch".to_string());
    }

    let sig = obj
        .get("signature")
        .and_then(|v| v.as_object())
        .ok_or_else(|| "signature missing".to_string())?;
    let alg = sig
        .get("algorithm")
        .and_then(|v| v.as_str())
        .ok_or_else(|| "signature.algorithm missing".to_string())?;
    if alg != "ed25519" {
        return Err("only ed25519 signatures are supported on windows-agent".to_string());
    }

    for (i, rule) in rules.iter().enumerate() {
        let ro = rule
            .as_object()
            .ok_or_else(|| format!("rule {i} not object"))?;
        if let Some(tos) = ro.get("target_os").and_then(|v| v.as_array()) {
            if !tos
                .iter()
                .filter_map(|x| x.as_str())
                .any(|s| s == "windows")
            {
                return Err(format!(
                    "rule {} target_os excludes windows",
                    read_rule_id(ro)
                ));
            }
        }
        let ra = ro
            .get("recommended_action")
            .and_then(|v| v.as_str())
            .ok_or_else(|| format!("rule {} recommended_action", read_rule_id(ro)))?;
        if !matches!(ra, "monitor" | "review" | "policy_candidate") {
            return Err(format!(
                "rule {} invalid recommended_action",
                read_rule_id(ro)
            ));
        }
        if !ro.contains_key("match") {
            return Err(format!("rule {} missing match", read_rule_id(ro)));
        }
    }

    Ok(())
}

fn read_rule_id(ro: &serde_json::Map<String, Value>) -> String {
    ro.get("rule_id")
        .and_then(|v| v.as_str())
        .unwrap_or("?")
        .to_string()
}

fn read_str<'a>(pack: &'a Value, key: &str) -> Option<&'a str> {
    pack.get(key)?.as_str()
}

/// `agent >= min` for `MAJOR.MINOR.PATCH` core semver (ignores pre-release for comparison).
fn semver_gte(agent: &str, min: &str) -> bool {
    let Some(a) = parse_semver_core(agent) else {
        return false;
    };
    let Some(m) = parse_semver_core(min) else {
        return false;
    };
    a >= m
}

fn parse_semver_core(s: &str) -> Option<(u32, u32, u32)> {
    let core = s.split('-').next()?.split('+').next()?;
    let mut parts = core.split('.');
    let major = parts.next()?.parse().ok()?;
    let minor = parts.next()?.parse().ok()?;
    let patch = parts.next()?.parse().ok()?;
    Some((major, minor, patch))
}

/// RFC3339 instant with `Z` UTC suffix (pack `expires_at` profile).
fn parse_rfc3339_utc(s: &str) -> Option<SystemTime> {
    use std::time::{Duration, UNIX_EPOCH};

    let s = s.trim();
    let body = s.strip_suffix('Z')?;
    let (date_part, time_part) = body.split_once('T')?;
    let mut frac_ns: u32 = 0;
    let time_core = if let Some((t, frac)) = time_part.split_once('.') {
        let digits: String = frac.chars().take_while(|c| c.is_ascii_digit()).collect();
        if !digits.is_empty() {
            let mut padded = format!("{:0<9}", digits);
            padded.truncate(9);
            frac_ns = padded.parse().ok()?;
        }
        t
    } else {
        time_part
    };

    let mut dp = date_part.split('-');
    let y: i32 = dp.next()?.parse().ok()?;
    let mo: u32 = dp.next()?.parse().ok()?;
    let d: u32 = dp.next()?.parse().ok()?;
    let mut tp = time_core.split(':');
    let h: u32 = tp.next()?.parse().ok()?;
    let mi: u32 = tp.next()?.parse().ok()?;
    let se: u32 = tp.next()?.parse().ok()?;
    if h > 23 || mi > 59 || se > 59 {
        return None;
    }

    let days = days_since_unix_epoch(y, mo, d)?;
    let day_secs = days.checked_mul(86_400)?;
    let offset = i64::from(h) * 3600 + i64::from(mi) * 60 + i64::from(se);
    let secs_total = day_secs.checked_add(offset)?;
    if secs_total < 0 {
        return None;
    }
    let base = UNIX_EPOCH.checked_add(Duration::from_secs(secs_total as u64))?;
    base.checked_add(Duration::from_nanos(u64::from(frac_ns)))
}

fn days_in_month(y: i64, month: u32) -> u32 {
    let md: [u32; 12] = [31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
    let mut dim = md[(month - 1) as usize];
    if month == 2 && is_leap_year(y) {
        dim += 1;
    }
    dim
}

fn days_since_unix_epoch(y: i32, mo: u32, d: u32) -> Option<i64> {
    if mo < 1 || mo > 12 || d < 1 || d > days_in_month(i64::from(y), mo) {
        return None;
    }
    let yy = i64::from(y);
    let mut total = 0i64;
    for year in 1970..yy {
        total += if is_leap_year(year) { 366 } else { 365 };
    }
    let md: [u32; 12] = [31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
    let mut idx = 0i64;
    for m in 1..mo {
        let mut dim = md[(m - 1) as usize];
        if m == 2 && is_leap_year(yy) {
            dim += 1;
        }
        idx += i64::from(dim);
    }
    Some(total + idx + i64::from(d) - 1)
}

fn is_leap_year(y: i64) -> bool {
    (y % 4 == 0 && y % 100 != 0) || (y % 400 == 0)
}

#[cfg(test)]
mod tests {
    #![allow(clippy::unwrap_used)]

    use super::*;
    use std::time::Duration;

    #[test]
    fn semver_gte_orders_versions() {
        assert!(semver_gte("1.0.0", "1.0.0"));
        assert!(semver_gte("1.0.1", "1.0.0"));
        assert!(!semver_gte("0.9.9", "1.0.0"));
    }

    #[test]
    fn rfc3339_parses_utc() {
        let t = parse_rfc3339_utc("2099-01-01T00:00:00Z").unwrap();
        assert!(t > SystemTime::UNIX_EPOCH + Duration::from_secs(4_000_000_000));
    }

    #[test]
    fn rfc3339_epoch() {
        assert_eq!(
            parse_rfc3339_utc("1970-01-01T00:00:00Z"),
            Some(SystemTime::UNIX_EPOCH)
        );
    }

    #[test]
    fn rejects_non_observe_mode() {
        let pack = serde_json::json!({
            "schema_version": "detection_pack.v1",
            "mode": "enforce",
            "supported_os": ["windows"],
            "min_agent_version": "0.0.1",
            "rules": [{"rule_id":"r","priority":1,"title":"t","description":"d","classification":"c",
                "pattern_tags":[],"agent_likelihood":0.1,"confidence":0.2,"risk_score":1,
                "recommended_action":"review","required_evidence":["process"],
                "match":{"process":{"executable_names_any":["bash"]}}}],
            "evaluator_limits": {
                "max_wall_time_ms_per_batch": 1000,
                "max_heap_bytes": 65536,
                "max_rules_evaluated_per_batch": 100,
                "max_cpu_percent_soft": 50.0,
                "max_string_comparisons_per_rule": 1000,
                "max_clause_depth": 8,
                "max_clauses_per_rule": 64
            },
            "signature": {"algorithm":"ed25519","key_id":"k","value_b64":"AA=="},
            "author": "a"
        });
        let r = validate_pack_for_windows(&pack, "1.0.0", SystemTime::UNIX_EPOCH);
        assert!(r.is_err());
    }

    #[test]
    fn rejects_missing_windows() {
        let pack = serde_json::json!({
            "schema_version": "detection_pack.v1",
            "mode": "observe",
            "supported_os": ["linux"],
            "min_agent_version": "0.0.1",
            "rules": [{"rule_id":"r","priority":1,"title":"t","description":"d","classification":"c",
                "pattern_tags":[],"agent_likelihood":0.1,"confidence":0.2,"risk_score":1,
                "recommended_action":"review","required_evidence":["process"],
                "match":{"process":{"executable_names_any":["bash"]}}}],
            "evaluator_limits": {
                "max_wall_time_ms_per_batch": 1000,
                "max_heap_bytes": 65536,
                "max_rules_evaluated_per_batch": 100,
                "max_cpu_percent_soft": 50.0,
                "max_string_comparisons_per_rule": 1000,
                "max_clause_depth": 8,
                "max_clauses_per_rule": 64
            },
            "signature": {"algorithm":"ed25519","key_id":"k","value_b64":"AA=="},
            "author": "a"
        });
        let r = validate_pack_for_windows(&pack, "1.0.0", SystemTime::UNIX_EPOCH);
        assert!(r.is_err());
    }
}
