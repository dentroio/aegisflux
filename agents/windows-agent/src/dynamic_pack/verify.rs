//! Signature and content-hash verification (matches detection-pipeline signing).

use base64::Engine;
use ed25519_dalek::{Signature, Verifier, VerifyingKey};
use serde_json::Value;
use sha2::{Digest, Sha256};

const SIGN_DOMAIN: &[u8] = b"aegis.detection_pack.v1\0";

/// Remove `signature` key and serialize with stable key order (`serde_json::Map` = `BTreeMap`).
pub fn canonical_unsigned_bytes(pack: &Value) -> Result<Vec<u8>, String> {
    let mut v = pack.clone();
    if let Value::Object(map) = &mut v {
        map.remove("signature");
    }
    serde_json::to_vec(&v).map_err(|e| format!("canonical pack json: {e}"))
}

/// Message bytes signed by the controller: domain || sha256(unsigned_pack_json).
pub fn signing_message(unsigned_pack_json: &[u8]) -> Vec<u8> {
    let h = Sha256::digest(unsigned_pack_json);
    let mut m = Vec::with_capacity(SIGN_DOMAIN.len() + 32);
    m.extend_from_slice(SIGN_DOMAIN);
    m.extend_from_slice(&h);
    m
}

/// Read detached signature (`value_b64`) from the pack document.
pub fn signature_value_b64(pack: &Value) -> Result<String, String> {
    let sig = pack
        .get("signature")
        .and_then(|v| v.as_object())
        .ok_or_else(|| "pack.signature missing".to_string())?;
    let b64 = sig
        .get("value_b64")
        .and_then(|v| v.as_str())
        .ok_or_else(|| "pack.signature.value_b64 missing".to_string())?;
    Ok(b64.to_string())
}

/// Verify Ed25519 detached signature over the canonical unsigned pack JSON.
pub fn verify_ed25519_signature(
    pack: &Value,
    public_key: &[u8; 32],
    signature_b64: &str,
) -> Result<(), String> {
    let unsigned = canonical_unsigned_bytes(pack)?;
    let msg = signing_message(&unsigned);
    let raw = base64::engine::general_purpose::STANDARD
        .decode(signature_b64.trim())
        .map_err(|e| format!("signature base64: {e}"))?;
    let sig = Signature::try_from(raw.as_slice())
        .map_err(|_| "signature must be 64 bytes".to_string())?;
    let vk = VerifyingKey::from_bytes(public_key).map_err(|e| format!("public key: {e}"))?;
    vk.verify(&msg, &sig)
        .map_err(|_| "ed25519 signature verification failed".to_string())
}

/// If `X-Content-SHA256` is present, it must equal the hex SHA-256 of `body`.
pub fn verify_content_sha256_header(headers: &str, body: &[u8]) -> Result<(), String> {
    use super::http::header_value;
    let Some(expected) = header_value(headers, "X-Content-SHA256") else {
        return Ok(());
    };
    let got = sha256_hex(body);
    if !expected.trim().eq_ignore_ascii_case(got.trim()) {
        return Err("X-Content-SHA256 does not match artifact body".to_string());
    }
    Ok(())
}

/// Hex-encoded SHA-256 of full pack bytes (must match `X-Content-SHA256` when present).
pub fn sha256_hex(bytes: &[u8]) -> String {
    let h = Sha256::digest(bytes);
    hex_lower(&h)
}

fn hex_lower(bytes: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut s = String::with_capacity(bytes.len() * 2);
    for b in bytes {
        s.push(char::from(HEX[(b >> 4) as usize]));
        s.push(char::from(HEX[(b & 0xf) as usize]));
    }
    s
}

#[cfg(test)]
mod tests {
    #![allow(clippy::unwrap_used)]
    #![allow(clippy::expect_used)]

    use super::*;
    use base64::Engine;
    use ed25519_dalek::Signer;
    use ed25519_dalek::SigningKey;
    use rand::rngs::OsRng;
    use serde_json::json;

    #[test]
    fn round_trip_sign_verify() {
        let sk = SigningKey::generate(&mut OsRng);
        let vk = sk.verifying_key();
        let mut pack = json!({
            "schema_version": "detection_pack.v1",
            "pack_id": "p",
            "pack_version": "1.0.0",
            "mode": "observe",
        });
        let unsigned = serde_json::to_vec(&pack).expect("test json");
        let msg = signing_message(&unsigned);
        let sig = sk.sign(&msg);
        let sig_b64 = base64::engine::general_purpose::STANDARD.encode(sig.to_bytes());
        pack.as_object_mut().expect("object").insert(
            "signature".to_string(),
            json!({"algorithm":"ed25519","key_id":"k","value_b64": sig_b64}),
        );
        let read = signature_value_b64(&pack).unwrap();
        verify_ed25519_signature(&pack, vk.as_bytes(), &read).unwrap();
    }
}
