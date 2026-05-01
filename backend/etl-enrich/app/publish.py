"""Publish enriched ETL events to NATS."""
from __future__ import annotations

import json
from typing import Any, Mapping


class EnrichedEventPublisher:
    """Small NATS publisher used by tests and legacy ETL call sites."""

    def __init__(self, nats_client: Any, subject: str = "events.enriched") -> None:
        self.nats_client = nats_client
        self.subject = subject

    async def publish(self, event: Mapping[str, Any], subject: str | None = None) -> None:
        payload = json.dumps(event, separators=(",", ":"), default=str).encode("utf-8")
        await self.nats_client.publish(subject or self.subject, payload)

    async def publish_batch(self, events: list[Mapping[str, Any]], subject: str | None = None) -> None:
        for event in events:
            await self.publish(event, subject=subject)
