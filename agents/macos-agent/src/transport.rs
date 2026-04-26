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

use std::io::Read;
use std::net::TcpStream;
use std::time::Duration;

/// Minimal HTTP client for Phase 1 localhost lab ingest.
#[derive(Debug, Clone)]
pub struct LabHttpPublisher {
    endpoint: HttpEndpoint,
}

impl LabHttpPublisher {
    /// Create a localhost-only HTTP publisher from `AEGIS_BACKEND_URL`.
    pub fn new(base_url: &str) -> Result<Self, String> {
        Ok(Self {
            endpoint: HttpEndpoint::parse(base_url)?,
        })
    }

    /// Post events to the ingest visibility endpoint as newline-delimited JSON.
    pub fn post_events(&self, events: &[AegisEvent]) -> Result<(), String> {
        if events.is_empty() {
            return Ok(());
        }

        let mut body = String::new();
        for event in events {
            body.push_str(&event.to_json());
            body.push('\n');
        }

        let mut stream = TcpStream::connect(self.endpoint.address())
            .map_err(|err| format!("failed to connect to ingest endpoint: {err}"))?;
        stream
            .set_read_timeout(Some(Duration::from_secs(5)))
            .map_err(|err| format!("failed to set ingest read timeout: {err}"))?;
        stream
            .set_write_timeout(Some(Duration::from_secs(5)))
            .map_err(|err| format!("failed to set ingest write timeout: {err}"))?;

        let request = format!(
            "POST {} HTTP/1.1\r\nHost: {}\r\nContent-Type: application/x-ndjson\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{}",
            self.endpoint.path,
            self.endpoint.host_header(),
            body.len(),
            body
        );
        stream
            .write_all(request.as_bytes())
            .map_err(|err| format!("failed to send ingest request: {err}"))?;

        let mut response = String::new();
        stream
            .read_to_string(&mut response)
            .map_err(|err| format!("failed to read ingest response: {err}"))?;
        validate_http_success(&response)
    }
}

#[derive(Debug, Clone)]
struct HttpEndpoint {
    host: String,
    port: u16,
    path: String,
}

impl HttpEndpoint {
    fn parse(base_url: &str) -> Result<Self, String> {
        let without_scheme = base_url.strip_prefix("http://").ok_or_else(|| {
            "macOS lab publisher only supports http://localhost or http://127.0.0.1".to_string()
        })?;
        let (authority, base_path) = match without_scheme.split_once('/') {
            Some((authority, path)) => (authority, format!("/{path}")),
            None => (without_scheme, String::new()),
        };
        let (host, port) = parse_authority(authority)?;
        if !is_localhost(&host) {
            return Err("macOS lab publisher only supports localhost HTTP endpoints".to_string());
        }

        Ok(Self {
            host,
            port,
            path: visibility_events_path(&base_path),
        })
    }

    fn address(&self) -> String {
        format!("{}:{}", self.host, self.port)
    }

    fn host_header(&self) -> String {
        format!("{}:{}", self.host, self.port)
    }
}

fn parse_authority(authority: &str) -> Result<(String, u16), String> {
    if authority.trim().is_empty() {
        return Err("AEGIS_BACKEND_URL is missing a host".to_string());
    }

    let (host, port) = match authority.rsplit_once(':') {
        Some((host, port_raw)) => {
            let port = port_raw
                .parse::<u16>()
                .map_err(|_| "AEGIS_BACKEND_URL port must be a number".to_string())?;
            (host.to_string(), port)
        }
        None => (authority.to_string(), 80),
    };

    Ok((host, port))
}

fn visibility_events_path(base_path: &str) -> String {
    let trimmed = base_path.trim_end_matches('/');
    if trimmed.is_empty() {
        "/v1/visibility/events".to_string()
    } else if trimmed.ends_with("/v1/visibility/events") {
        trimmed.to_string()
    } else {
        format!("{trimmed}/v1/visibility/events")
    }
}

fn is_localhost(host: &str) -> bool {
    matches!(host, "localhost" | "127.0.0.1" | "[::1]")
}

fn validate_http_success(response: &str) -> Result<(), String> {
    let status_line = response
        .lines()
        .next()
        .ok_or_else(|| "ingest returned an empty response".to_string())?;
    if status_line.starts_with("HTTP/1.1 2") || status_line.starts_with("HTTP/1.0 2") {
        Ok(())
    } else {
        Err(format!("ingest returned non-success status: {status_line}"))
    }
}

#[cfg(test)]
mod tests {
    use super::{HttpEndpoint, visibility_events_path};

    #[test]
    fn appends_visibility_events_path() {
        assert_eq!(visibility_events_path(""), "/v1/visibility/events");
        assert_eq!(
            visibility_events_path("/ingest"),
            "/ingest/v1/visibility/events"
        );
        assert_eq!(
            visibility_events_path("/v1/visibility/events"),
            "/v1/visibility/events"
        );
    }

    #[test]
    fn parses_localhost_endpoint() {
        let endpoint = HttpEndpoint::parse("http://127.0.0.1:9090").unwrap();
        assert_eq!(endpoint.address(), "127.0.0.1:9090");
        assert_eq!(endpoint.path, "/v1/visibility/events");
    }

    #[test]
    fn rejects_remote_http_endpoint() {
        assert!(HttpEndpoint::parse("http://example.com:9090").is_err());
    }
}
