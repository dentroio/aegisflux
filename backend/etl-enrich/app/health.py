"""HTTP health and metrics endpoints for ETL enrichment."""

from __future__ import annotations

import json
import logging
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from threading import Thread
from typing import Tuple

from .runtime import RuntimeState

logger = logging.getLogger(__name__)


class HealthHTTPServer(ThreadingHTTPServer):
    """HTTP server carrying shared runtime state."""

    def __init__(self, server_address: Tuple[str, int], state: RuntimeState):
        super().__init__(server_address, HealthHandler)
        self.state = state


class HealthHandler(BaseHTTPRequestHandler):
    """Serve health, readiness, and Prometheus-style metrics."""

    server: HealthHTTPServer

    def log_message(self, format: str, *args) -> None:
        logger.debug("HTTP health request: " + format, *args)

    def do_GET(self) -> None:
        if self.path == "/healthz":
            snap = self.server.state.snapshot()
            self._write_json(
                200,
                {
                    "service": snap["service"],
                    "status": "healthy" if snap["status"] == "ready" else "degraded",
                    "dependencies": snap["dependencies"],
                    "started_at": snap.get("started_at"),
                },
            )
            return

        if self.path == "/readyz":
            snapshot = self.server.state.snapshot()
            status = 200 if snapshot["status"] == "ready" else 503
            self._write_json(status, snapshot)
            return

        if self.path == "/metrics":
            body = self.server.state.metrics_text().encode("utf-8")
            self.send_response(200)
            self.send_header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return

        self._write_json(404, {"error": "not found"})

    def _write_json(self, status: int, payload: dict) -> None:
        body = json.dumps(payload, sort_keys=True).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


def start_health_server(state: RuntimeState, host: str, port: int) -> HealthHTTPServer:
    """Start health server in a daemon thread and return the server."""

    server = HealthHTTPServer((host, port), state)
    thread = Thread(target=server.serve_forever, name="etl-health-http", daemon=True)
    thread.start()
    logger.info("Started ETL health HTTP server on %s:%s", host, server.server_port)
    return server
