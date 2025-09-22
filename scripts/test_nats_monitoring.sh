#!/usr/bin/env bash
set -euo pipefail
MON=${1:-http://localhost:8222}
echo "[*] varz:"
curl -fsS ${MON}/varz | jq . | head -n 30
echo "[*] connz:"
curl -fsS ${MON}/connz | jq . | head -n 30



