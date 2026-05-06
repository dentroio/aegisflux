//! On-disk cache for verified active and previous packs (WO-DET-004).

use std::fs;
use std::path::{Path, PathBuf};

const ACTIVE_PACK: &str = "active_verified.pack.json";
const ACTIVE_META: &str = "active_verified.meta.json";
const PREV_PACK: &str = "previous_verified.pack.json";
const PREV_META: &str = "previous_verified.meta.json";

/// Metadata stored next to a cached pack artifact.
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct PackCacheMeta {
    /// Controller artifact id when known.
    pub artifact_id: String,
    pub pack_id: String,
    pub pack_version: String,
    pub sha256: String,
}

/// Cached verified pack bytes + metadata.
#[derive(Debug, Clone)]
pub struct CachedPack {
    pub bytes: Vec<u8>,
    pub meta: PackCacheMeta,
}

fn read_if_exists(path: &Path) -> Result<Option<Vec<u8>>, String> {
    match fs::read(path) {
        Ok(b) => Ok(Some(b)),
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(None),
        Err(e) => Err(format!("read {}: {e}", path.display())),
    }
}

/// Load active verified pack from cache directory.
pub fn load_active(dir: &Path) -> Result<Option<CachedPack>, String> {
    let bytes = match read_if_exists(&dir.join(ACTIVE_PACK))? {
        Some(b) => b,
        None => return Ok(None),
    };
    let meta_raw =
        read_if_exists(&dir.join(ACTIVE_META))?.ok_or_else(|| "active meta missing".to_string())?;
    let meta: PackCacheMeta =
        serde_json::from_slice(&meta_raw).map_err(|e| format!("active meta json: {e}"))?;
    Ok(Some(CachedPack { bytes, meta }))
}

/// Load previous verified pack (rollback source).
pub fn load_previous(dir: &Path) -> Result<Option<CachedPack>, String> {
    let bytes = match read_if_exists(&dir.join(PREV_PACK))? {
        Some(b) => b,
        None => return Ok(None),
    };
    let meta_raw =
        read_if_exists(&dir.join(PREV_META))?.ok_or_else(|| "previous meta missing".to_string())?;
    let meta: PackCacheMeta =
        serde_json::from_slice(&meta_raw).map_err(|e| format!("previous meta json: {e}"))?;
    Ok(Some(CachedPack { bytes, meta }))
}

/// Promote current active to previous, then write new active.
pub fn promote_and_write_active(
    dir: &Path,
    previous: Option<&CachedPack>,
    new_bytes: &[u8],
    new_meta: PackCacheMeta,
) -> Result<(), String> {
    fs::create_dir_all(dir).map_err(|e| format!("cache dir: {e}"))?;
    if let Some(p) = previous {
        fs::write(dir.join(PREV_PACK), &p.bytes).map_err(|e| format!("write prev pack: {e}"))?;
        let prev_json =
            serde_json::to_vec_pretty(&p.meta).map_err(|e| format!("prev meta: {e}"))?;
        fs::write(dir.join(PREV_META), prev_json).map_err(|e| format!("write prev meta: {e}"))?;
    }
    fs::write(dir.join(ACTIVE_PACK), new_bytes).map_err(|e| format!("write active pack: {e}"))?;
    let meta_json =
        serde_json::to_vec_pretty(&new_meta).map_err(|e| format!("active meta: {e}"))?;
    fs::write(dir.join(ACTIVE_META), meta_json).map_err(|e| format!("write active meta: {e}"))?;
    Ok(())
}

/// Write only active (first install).
pub fn write_active(dir: &Path, bytes: &[u8], meta: PackCacheMeta) -> Result<(), String> {
    fs::create_dir_all(dir).map_err(|e| format!("cache dir: {e}"))?;
    fs::write(dir.join(ACTIVE_PACK), bytes).map_err(|e| format!("write active pack: {e}"))?;
    let meta_json = serde_json::to_vec_pretty(&meta).map_err(|e| format!("active meta: {e}"))?;
    fs::write(dir.join(ACTIVE_META), meta_json).map_err(|e| format!("write active meta: {e}"))?;
    Ok(())
}

/// Default cache directory under the spool parent when unset.
pub fn default_cache_dir(spool: &Path) -> PathBuf {
    spool
        .parent()
        .unwrap_or_else(|| Path::new("."))
        .join("detection-pack")
}
