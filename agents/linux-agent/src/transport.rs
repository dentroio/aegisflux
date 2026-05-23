//! Event transport and local spool.

use std::fs::{self, OpenOptions};
use std::io::{Read, Write};
use std::net::TcpStream;
use std::path::PathBuf;
use std::thread;
use std::time::Duration;

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

/// HTTP transport for Phase 1 lab ingest.
#[derive(Debug, Clone)]
pub struct HttpVisibilityTransport {
    endpoint: HttpEndpoint,
}

/// HTTP JSON transport for exact lab API endpoints.
#[derive(Debug, Clone)]
pub struct HttpJsonTransport {
    endpoint: HttpEndpoint,
}

impl HttpVisibilityTransport {
    /// Create an HTTP visibility transport.
    pub fn new(base_url: &str) -> Result<Self, String> {
        Ok(Self {
            endpoint: HttpEndpoint::parse(base_url)?,
        })
    }

    /// Post a batch of events as newline-delimited JSON.
    pub fn post_events(&self, events: &[AegisEvent]) -> Result<(), String> {
        self.post_events_chunked(events, events.len())
    }

    /// Post events in bounded chunks to keep request bodies predictable at fleet scale.
    pub fn post_events_chunked(&self, events: &[AegisEvent], chunk_size: usize) -> Result<(), String> {
        if events.is_empty() {
            return Ok(());
        }
        if chunk_size == 0 {
            return Err("visibility post chunk size must be greater than zero".to_string());
        }

        for chunk in events.chunks(chunk_size) {
            let mut body = String::new();
            for event in chunk {
                body.push_str(&event.to_json());
                body.push('\n');
            }
            self.endpoint.post_body(&body, "application/x-ndjson")?;
        }
        Ok(())
    }
}

impl HttpJsonTransport {
    /// Create an HTTP JSON transport for an exact endpoint URL.
    pub fn new(url: &str) -> Result<Self, String> {
        Ok(Self {
            endpoint: HttpEndpoint::parse_exact(url, "AEGIS_ACTIONS_HEARTBEAT_URL")?,
        })
    }

    /// Post one JSON document.
    pub fn post_json(&self, body: &str) -> Result<(), String> {
        self.endpoint.post_body(body, "application/json")
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
struct HttpEndpoint {
    host: String,
    port: u16,
    path: String,
}

impl HttpEndpoint {
    fn parse(base_url: &str) -> Result<Self, String> {
        let (host, port, raw_path) = parse_http_url(base_url, "AEGIS_BACKEND_URL")?;
        let path = visibility_events_path(&raw_path);
        Ok(Self { host, port, path })
    }

    fn parse_exact(url: &str, name: &str) -> Result<Self, String> {
        let (host, port, path) = parse_http_url(url, name)?;
        if path.is_empty() {
            return Err(format!("{name} is missing a path"));
        }
        Ok(Self { host, port, path })
    }

    fn post_body(&self, body: &str, content_type: &str) -> Result<(), String> {
        const MAX_ATTEMPTS: usize = 3;
        let address = format!("{}:{}", self.host, self.port);
        let mut last_error = String::new();
        for attempt in 1..=MAX_ATTEMPTS {
            match self.post_body_once(body, content_type, &address) {
                Ok(()) => return Ok(()),
                Err(PostError::Retryable(error)) => {
                    last_error = error;
                    if attempt < MAX_ATTEMPTS {
                        thread::sleep(Duration::from_millis((attempt as u64) * 250));
                    }
                }
                Err(PostError::NonRetryable(error)) => return Err(error),
            }
        }

        Err(format!(
            "failed to deliver HTTP payload after {MAX_ATTEMPTS} attempts: {last_error}"
        ))
    }

    fn post_body_once(
        &self,
        body: &str,
        content_type: &str,
        address: &str,
    ) -> Result<(), PostError> {
        let mut stream = TcpStream::connect(address).map_err(|err| {
            PostError::Retryable(format!(
                "failed to connect to HTTP endpoint at {address}: {err}"
            ))
        })?;

        let request = format!(
            "POST {} HTTP/1.1\r\nHost: {}\r\nContent-Type: {}\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{}",
            self.path,
            self.host,
            content_type,
            body.len(),
            body
        );

        stream
            .write_all(request.as_bytes())
            .map_err(|err| PostError::Retryable(format!("failed to send HTTP payload: {err}")))?;

        let mut response = String::new();
        stream
            .read_to_string(&mut response)
            .map_err(|err| PostError::Retryable(format!("failed to read HTTP response: {err}")))?;

        let status_line = response.lines().next().unwrap_or_default();
        if status_line.starts_with("HTTP/1.1 2") || status_line.starts_with("HTTP/1.0 2") {
            return Ok(());
        }
        if status_line.starts_with("HTTP/1.1 4") || status_line.starts_with("HTTP/1.0 4") {
            return Err(PostError::NonRetryable(format!(
                "HTTP endpoint rejected payload: {status_line}"
            )));
        }
        Err(PostError::Retryable(format!(
            "HTTP endpoint rejected payload: {status_line}"
        )))
    }
}

fn parse_http_url(raw_url: &str, name: &str) -> Result<(String, u16, String), String> {
    let without_scheme = raw_url
        .strip_prefix("http://")
        .ok_or_else(|| format!("{name} lab transport supports http:// only"))?;

    let (authority, raw_path) = match without_scheme.split_once('/') {
        Some((authority, path)) => (authority, format!("/{path}")),
        None => (without_scheme, String::new()),
    };

    if authority.is_empty() {
        return Err(format!("{name} is missing a host"));
    }

    let (host, port) = parse_authority(authority, name)?;
    Ok((host, port, raw_path))
}

enum PostError {
    Retryable(String),
    NonRetryable(String),
}

fn parse_authority(authority: &str, name: &str) -> Result<(String, u16), String> {
    let (host, port) = match authority.rsplit_once(':') {
        Some((host, port)) => {
            let parsed_port = port
                .parse::<u16>()
                .map_err(|_| format!("{name} has an invalid port"))?;
            (host, parsed_port)
        }
        None => (authority, 80),
    };

    if host.is_empty() {
        return Err(format!("{name} is missing a host"));
    }

    Ok((host.to_string(), port))
}

fn visibility_events_path(raw_path: &str) -> String {
    let trimmed = raw_path.trim_end_matches('/');
    if trimmed.is_empty() {
        "/v1/visibility/events".to_string()
    } else if trimmed.ends_with("/v1/visibility/events") {
        trimmed.to_string()
    } else {
        format!("{trimmed}/v1/visibility/events")
    }
}

#[cfg(test)]
mod tests {
    use super::{visibility_events_path, HttpEndpoint};

    #[test]
    fn parses_localhost_endpoint_with_default_path() -> Result<(), String> {
        let endpoint = HttpEndpoint::parse("http://127.0.0.1:9090")?;

        assert_eq!(endpoint.host, "127.0.0.1");
        assert_eq!(endpoint.port, 9090);
        assert_eq!(endpoint.path, "/v1/visibility/events");
        Ok(())
    }

    #[test]
    fn appends_visibility_path_to_base_path() {
        assert_eq!(
            visibility_events_path("/aegis"),
            "/aegis/v1/visibility/events"
        );
        assert_eq!(
            visibility_events_path("/v1/visibility/events"),
            "/v1/visibility/events"
        );
    }

    #[test]
    fn rejects_https_for_dependency_free_lab_transport() -> Result<(), String> {
        let err = match HttpEndpoint::parse("https://aegis.example.com") {
            Ok(_) => return Err("https endpoint unexpectedly parsed".to_string()),
            Err(err) => err,
        };

        assert!(err.contains("http:// only"));
        Ok(())
    }
}
