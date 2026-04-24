//! Event transport and local spool.

use std::fs::{self, OpenOptions};
use std::io::Write;
use std::path::PathBuf;

use crate::event::AegisEvent;

/// JSONL event spool.
#[derive(Debug, Clone)]
pub struct JsonlSpool {
    path: PathBuf,
}

impl JsonlSpool {
    /// Create a JSONL spool transport.
    pub fn new(path: PathBuf) -> Self {
        Self { path }
    }

    /// Append one event to the local spool.
    pub fn append(&self, event: &AegisEvent) -> Result<(), String> {
        if let Some(parent) = self.path.parent() {
            fs::create_dir_all(parent).map_err(|err| {
                format!(
                    "failed to create spool directory {}: {err}",
                    parent.display()
                )
            })?;
        }

        let mut file = OpenOptions::new()
            .create(true)
            .append(true)
            .open(&self.path)
            .map_err(|err| format!("failed to open spool {}: {err}", self.path.display()))?;

        file.write_all(event.to_json().as_bytes())
            .and_then(|_| file.write_all(b"\n"))
            .map_err(|err| format!("failed to write event spool {}: {err}", self.path.display()))
    }
}
