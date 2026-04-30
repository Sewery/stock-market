#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${1:-http://localhost:8080}"

curl -sS -X POST "$BASE_URL/stocks" \
  -H 'content-type: application/json' \
  -d '{
    "stocks": [
      {"name":"stock1","quantity":5},
      {"name":"stock2","quantity":2}
    ]
  }' >/dev/null

echo "Seeded bank on $BASE_URL"