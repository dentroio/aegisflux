"""Tests for ETL consumer payload decoding."""

import base64
import json

from app.payload import decode_payload_args


def test_decode_payload_args_accepts_single_base64_json():
    payload = base64.b64encode(json.dumps({"dst_ip": "203.0.113.10", "dst_port": 443}).encode()).decode()

    assert decode_payload_args(payload) == {
        "dst_ip": "203.0.113.10",
        "dst_port": 443,
    }


def test_decode_payload_args_accepts_double_base64_json():
    inner = base64.b64encode(json.dumps({"dst_ip": "203.0.113.11", "dst_port": 8443}).encode())
    payload = base64.b64encode(inner).decode()

    assert decode_payload_args(payload) == {
        "dst_ip": "203.0.113.11",
        "dst_port": 8443,
    }


def test_decode_payload_args_handles_empty_payload():
    assert decode_payload_args(None) == {}
