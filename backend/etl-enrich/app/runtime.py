"""Runtime state and metrics for the ETL enrichment service."""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone
from threading import Lock
from typing import Any, Dict, Optional


def _now() -> datetime:
    return datetime.now(timezone.utc)


def _iso_timestamp(value: Optional[datetime]) -> Optional[str]:
    if value is None:
        return None
    return value.isoformat()


@dataclass
class RuntimeState:
    """Thread-safe service state shared by the consumer and HTTP server."""

    service: str = "etl-enrich"
    started_at: datetime = field(default_factory=_now)
    running: bool = False
    nats_connected: bool = False
    timescale_connected: bool = False
    neo4j_connected: bool = False
    processed_messages: int = 0
    processing_errors: int = 0
    last_processed_at: Optional[datetime] = None
    last_error: Optional[str] = None
    last_error_at: Optional[datetime] = None
    _lock: Lock = field(default_factory=Lock, init=False, repr=False)

    def set_running(self, running: bool) -> None:
        with self._lock:
            self.running = running

    def set_nats_connected(self, connected: bool) -> None:
        with self._lock:
            self.nats_connected = connected

    def set_timescale_connected(self, connected: bool) -> None:
        with self._lock:
            self.timescale_connected = connected

    def set_neo4j_connected(self, connected: bool) -> None:
        with self._lock:
            self.neo4j_connected = connected

    def record_processed(self) -> None:
        with self._lock:
            self.processed_messages += 1
            self.last_processed_at = _now()

    def record_error(self, error: Any) -> None:
        with self._lock:
            self.processing_errors += 1
            self.last_error = str(error)
            self.last_error_at = _now()

    def clear_error(self) -> None:
        with self._lock:
            self.last_error = None
            self.last_error_at = None

    def is_ready(self) -> bool:
        with self._lock:
            return (
                self.running
                and self.nats_connected
                and self.timescale_connected
                and self.neo4j_connected
            )

    def snapshot(self) -> Dict[str, Any]:
        with self._lock:
            ready = (
                self.running
                and self.nats_connected
                and self.timescale_connected
                and self.neo4j_connected
            )
            return {
                "service": self.service,
                "status": "ready" if ready else "not ready",
                "started_at": _iso_timestamp(self.started_at),
                "running": self.running,
                "dependencies": {
                    "nats": self.nats_connected,
                    "timescale": self.timescale_connected,
                    "neo4j": self.neo4j_connected,
                },
                "processed_messages": self.processed_messages,
                "processing_errors": self.processing_errors,
                "last_processed_at": _iso_timestamp(self.last_processed_at),
                "last_error": self.last_error,
                "last_error_at": _iso_timestamp(self.last_error_at),
            }

    def metrics_text(self) -> str:
        snapshot = self.snapshot()
        deps = snapshot["dependencies"]
        lines = [
            "# HELP etl_enrich_up Whether the ETL enrichment process is running.",
            "# TYPE etl_enrich_up gauge",
            f"etl_enrich_up {1 if snapshot['running'] else 0}",
            "# HELP etl_enrich_ready Whether ETL dependencies are connected and the consumer is running.",
            "# TYPE etl_enrich_ready gauge",
            f"etl_enrich_ready {1 if snapshot['status'] == 'ready' else 0}",
            "# HELP etl_enrich_dependency_connected Whether an ETL dependency is connected.",
            "# TYPE etl_enrich_dependency_connected gauge",
        ]
        for name, connected in deps.items():
            lines.append(f'etl_enrich_dependency_connected{{dependency="{name}"}} {1 if connected else 0}')
        lines.extend(
            [
                "# HELP etl_enrich_processed_messages_total Total processed ETL messages.",
                "# TYPE etl_enrich_processed_messages_total counter",
                f"etl_enrich_processed_messages_total {snapshot['processed_messages']}",
                "# HELP etl_enrich_processing_errors_total Total ETL processing errors.",
                "# TYPE etl_enrich_processing_errors_total counter",
                f"etl_enrich_processing_errors_total {snapshot['processing_errors']}",
            ]
        )
        return "\n".join(lines) + "\n"
