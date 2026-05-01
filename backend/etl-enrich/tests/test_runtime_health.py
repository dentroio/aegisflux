"""Tests for ETL runtime state and health endpoints."""

import json
import urllib.error
import urllib.request

from app.health import start_health_server
from app.runtime import RuntimeState


def test_runtime_state_readiness_and_metrics():
    state = RuntimeState()

    assert state.is_ready() is False
    snapshot = state.snapshot()
    assert snapshot["service"] == "etl-enrich"
    assert snapshot["status"] == "not ready"

    state.set_running(True)
    state.set_nats_connected(True)
    state.set_timescale_connected(True)
    state.set_neo4j_connected(True)
    state.record_processed()
    state.record_error("sample error")

    assert state.is_ready() is True
    snapshot = state.snapshot()
    assert snapshot["status"] == "ready"
    assert snapshot["processed_messages"] == 1
    assert snapshot["processing_errors"] == 1
    assert snapshot["last_error"] == "sample error"

    metrics = state.metrics_text()
    assert "etl_enrich_ready 1" in metrics
    assert "etl_enrich_processed_messages_total 1" in metrics
    assert "etl_enrich_processing_errors_total 1" in metrics


def test_health_server_readiness_status_codes():
    state = RuntimeState()
    server = start_health_server(state, "127.0.0.1", 0)
    base_url = f"http://127.0.0.1:{server.server_port}"

    try:
        with urllib.request.urlopen(f"{base_url}/healthz", timeout=2) as response:
            assert response.status == 200
            body = json.loads(response.read())
            assert body["status"] == "healthy"

        try:
            urllib.request.urlopen(f"{base_url}/readyz", timeout=2)
            assert False, "readyz should return 503 before dependencies are connected"
        except urllib.error.HTTPError as exc:
            assert exc.code == 503
            body = json.loads(exc.read())
            assert body["status"] == "not ready"

        state.set_running(True)
        state.set_nats_connected(True)
        state.set_timescale_connected(True)
        state.set_neo4j_connected(True)

        with urllib.request.urlopen(f"{base_url}/readyz", timeout=2) as response:
            assert response.status == 200
            body = json.loads(response.read())
            assert body["status"] == "ready"

        with urllib.request.urlopen(f"{base_url}/metrics", timeout=2) as response:
            assert response.status == 200
            assert b"etl_enrich_ready 1" in response.read()
    finally:
        server.shutdown()
        server.server_close()
