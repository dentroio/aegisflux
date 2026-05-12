//! Runtime helpers for long-running supervised service mode.

use std::env;
use std::os::unix::net::UnixDatagram;
use std::path::Path;
use std::thread;
use std::time::{Duration, Instant};

/// Notify a systemd-compatible supervisor, if one is present.
pub fn notify_systemd(state: &str) {
    let Ok(socket_path) = env::var("NOTIFY_SOCKET") else {
        return;
    };
    notify_systemd_at(socket_path.trim(), state);
}

fn notify_systemd_at(socket_path: &str, state: &str) {
    if socket_path.is_empty() {
        return;
    }

    let Ok(socket) = UnixDatagram::unbound() else {
        return;
    };
    send_notify_message(&socket, socket_path, state);
}

#[cfg(target_os = "linux")]
fn send_notify_message(socket: &UnixDatagram, socket_path: &str, state: &str) {
    use std::os::linux::net::SocketAddrExt;

    if let Some(abstract_name) = socket_path.strip_prefix('@') {
        if let Ok(address) = std::os::unix::net::SocketAddr::from_abstract_name(abstract_name) {
            let _ = socket.send_to_addr(state.as_bytes(), &address);
        }
        return;
    }

    let _ = socket.send_to(state.as_bytes(), Path::new(socket_path));
}

#[cfg(not(target_os = "linux"))]
fn send_notify_message(socket: &UnixDatagram, socket_path: &str, state: &str) {
    if socket_path.starts_with('@') {
        return;
    }

    let _ = socket.send_to(state.as_bytes(), Path::new(socket_path));
}

/// Sleep in short slices so watchdog notifications continue between collection cycles.
pub fn sleep_with_watchdog(duration: Duration) {
    let started = Instant::now();
    while started.elapsed() < duration {
        notify_systemd("WATCHDOG=1");
        let remaining = duration.saturating_sub(started.elapsed());
        thread::sleep(remaining.min(Duration::from_secs(10)));
    }
}

#[cfg(test)]
mod tests {
    use std::time::Duration;

    use super::sleep_with_watchdog;

    #[cfg(target_os = "linux")]
    #[test]
    fn notify_systemd_sends_to_filesystem_socket() -> Result<(), Box<dyn std::error::Error>> {
        use std::fs;
        use std::os::unix::net::UnixDatagram;

        use super::notify_systemd_at;

        let socket_path = std::env::temp_dir().join(format!(
            "aegis-linux-agent-notify-{}.sock",
            std::process::id()
        ));
        let _ = fs::remove_file(&socket_path);
        let listener = UnixDatagram::bind(&socket_path)?;
        listener.set_read_timeout(Some(Duration::from_secs(1)))?;

        let socket_path = socket_path.to_string_lossy().into_owned();
        notify_systemd_at(&socket_path, "READY=1\nSTATUS=test");

        let mut buf = [0_u8; 128];
        let size = listener.recv(&mut buf)?;
        assert_eq!(&buf[..size], b"READY=1\nSTATUS=test");

        let _ = fs::remove_file(&socket_path);
        Ok(())
    }

    #[test]
    fn sleep_with_watchdog_returns_for_zero_duration() {
        sleep_with_watchdog(Duration::from_millis(0));
    }
}
