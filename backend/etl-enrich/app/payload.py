"""Payload decoding helpers for ingest-produced events."""

import base64
import json
from typing import Any, Dict, Optional


def decode_payload_args(payload: Optional[str]) -> Dict[str, Any]:
    """Decode the base64 JSON payload emitted by ingest."""
    if not payload:
        return {}

    payload_bytes = base64.b64decode(payload)
    try:
        decoded = base64.b64decode(payload_bytes.decode("utf-8"))
        return json.loads(decoded.decode("utf-8"))
    except Exception:
        return json.loads(payload_bytes.decode("utf-8"))
