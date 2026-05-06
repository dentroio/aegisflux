//! Minimal HTTP/1.1 client for lab controller (TCP, no TLS).

use std::io::{Read, Write};
use std::net::TcpStream;

/// Parsed `http://host:port` base URL.
#[derive(Debug, Clone)]
pub struct HttpBase {
    host: String,
    port: u16,
}

impl HttpBase {
    /// Parse `http://host:port` or `http://host:port/` (path suffix ignored).
    pub fn parse(base_url: &str) -> Result<Self, String> {
        let rest = base_url
            .strip_prefix("http://")
            .ok_or_else(|| "controller URL must use http:// (lab only)".to_string())?;
        let authority = rest.split_once('/').map(|(a, _)| a).unwrap_or(rest);
        if authority.is_empty() {
            return Err("controller URL missing host".to_string());
        }
        let (host, port) = match authority.rsplit_once(':') {
            Some((h, p)) => {
                let port = p
                    .parse::<u16>()
                    .map_err(|_| "controller URL has invalid port".to_string())?;
                (h.to_string(), port)
            }
            None => (authority.to_string(), 80),
        };
        if host.is_empty() {
            return Err("controller URL missing host".to_string());
        }
        Ok(Self { host, port })
    }

    /// GET request; returns (status_code, header block as string, body bytes).
    pub fn get(&self, path_and_query: &str) -> Result<(u16, String, Vec<u8>), String> {
        self.request("GET", path_and_query, None)
    }

    /// POST JSON body.
    pub fn post_json(
        &self,
        path_and_query: &str,
        body: &str,
    ) -> Result<(u16, String, Vec<u8>), String> {
        self.request("POST", path_and_query, Some(("application/json", body)))
    }

    fn request(
        &self,
        method: &str,
        path_and_query: &str,
        body: Option<(&'static str, &str)>,
    ) -> Result<(u16, String, Vec<u8>), String> {
        let path = if path_and_query.starts_with('/') {
            path_and_query.to_string()
        } else {
            format!("/{path_and_query}")
        };
        let addr = format!("{}:{}", self.host, self.port);
        let mut stream = TcpStream::connect(&addr).map_err(|e| format!("connect {addr}: {e}"))?;

        let mut req = format!(
            "{method} {path} HTTP/1.1\r\nHost: {}\r\nConnection: close\r\n",
            self.host
        );
        if let Some((ct, b)) = body {
            req.push_str(&format!(
                "Content-Type: {ct}\r\nContent-Length: {}\r\n",
                b.len()
            ));
            req.push_str("\r\n");
            req.push_str(b);
        } else {
            req.push_str("\r\n");
        }
        stream
            .write_all(req.as_bytes())
            .map_err(|e| format!("write request: {e}"))?;

        let mut buf = Vec::new();
        stream
            .read_to_end(&mut buf)
            .map_err(|e| format!("read response: {e}"))?;

        let pos = buf
            .windows(4)
            .position(|window| window == b"\r\n\r\n")
            .ok_or_else(|| "invalid HTTP response".to_string())?;
        let header_block = std::str::from_utf8(&buf[..pos])
            .map_err(|_| "invalid HTTP header encoding".to_string())?;
        let status_line = header_block.lines().next().unwrap_or("");
        let code = parse_status(status_line)?;
        let raw_body = &buf[pos + 4..];
        let body_out = if is_chunked(header_block) {
            decode_chunked(raw_body)?
        } else {
            raw_body.to_vec()
        };
        Ok((code, header_block.to_string(), body_out))
    }
}

fn parse_status(line: &str) -> Result<u16, String> {
    let mut parts = line.split_whitespace();
    let _http = parts.next().ok_or_else(|| "bad status".to_string())?;
    let code = parts
        .next()
        .ok_or_else(|| "bad status".to_string())?
        .parse::<u16>()
        .map_err(|_| "bad status code".to_string())?;
    Ok(code)
}

/// Extract header value (case-insensitive name).
pub fn header_value<'a>(headers: &'a str, name: &str) -> Option<&'a str> {
    let needle = format!("{}:", name.to_ascii_lowercase());
    for line in headers.lines() {
        let l = line.trim();
        if l.to_ascii_lowercase().starts_with(&needle) {
            return Some(l.split_once(':').map(|(_, v)| v.trim()).unwrap_or(""));
        }
    }
    None
}

fn is_chunked(headers: &str) -> bool {
    header_value(headers, "Transfer-Encoding")
        .map(|v| {
            v.split(',')
                .any(|part| part.trim().eq_ignore_ascii_case("chunked"))
        })
        .unwrap_or(false)
}

fn decode_chunked(raw: &[u8]) -> Result<Vec<u8>, String> {
    let mut out = Vec::new();
    let mut pos = 0;
    loop {
        let line_end = find_crlf(&raw[pos..]).ok_or_else(|| "bad chunked response".to_string())?;
        let line = std::str::from_utf8(&raw[pos..pos + line_end])
            .map_err(|_| "bad chunk size encoding".to_string())?;
        let size_hex = line.split_once(';').map(|(s, _)| s).unwrap_or(line).trim();
        let size = usize::from_str_radix(size_hex, 16).map_err(|_| "bad chunk size".to_string())?;
        pos += line_end + 2;
        if size == 0 {
            return Ok(out);
        }
        if raw.len().saturating_sub(pos) < size + 2 {
            return Err("truncated chunked response".to_string());
        }
        out.extend_from_slice(&raw[pos..pos + size]);
        pos += size;
        if raw.get(pos..pos + 2) != Some(b"\r\n") {
            return Err("bad chunk terminator".to_string());
        }
        pos += 2;
    }
}

fn find_crlf(bytes: &[u8]) -> Option<usize> {
    bytes.windows(2).position(|window| window == b"\r\n")
}

#[cfg(test)]
mod tests {
    #![allow(clippy::unwrap_used)]

    use super::{decode_chunked, HttpBase};

    #[test]
    fn parses_controller_base() {
        let b = HttpBase::parse("http://127.0.0.1:8089").unwrap();
        assert_eq!(b.host, "127.0.0.1");
        assert_eq!(b.port, 8089);
    }

    #[test]
    fn decodes_chunked_body() {
        let decoded = decode_chunked(b"5\r\nhello\r\n6\r\n world\r\n0\r\n\r\n").unwrap();
        assert_eq!(decoded, b"hello world");
    }
}
