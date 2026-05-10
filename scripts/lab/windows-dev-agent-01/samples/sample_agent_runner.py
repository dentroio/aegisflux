#!/usr/bin/env python3
"""Lab-only stub: WO-VIS-001 / WO-VIS-007 scenario generation (no real agent loop)."""
import os
import sys
import urllib.request

TARGET_URL = os.environ.get("AEGIS_LAB_TARGET_URL", "https://example.com/")


def main() -> None:
    print("sample_agent_runner: openai tool marker for lab detection", file=sys.stderr)
    req = urllib.request.Request(
        TARGET_URL,
        headers={"User-Agent": "aegis-lab-agent-runner/0.1"},
        method="GET",
    )
    with urllib.request.urlopen(req, timeout=10) as resp:
        print(resp.status)


if __name__ == "__main__":
    main()
