#!/usr/bin/env bash
set -euo pipefail

OUT="${1:-backup.sql}"
SERVICE="${2:-postgres}"

CONTAINER="$(docker compose ps -q "$SERVICE")"
if [[ -z "$CONTAINER" ]]; then
  echo "Container for service '$SERVICE' not found"
  exit 1
fi

docker exec -t "$CONTAINER" pg_dump -U stock -d stock_market > "$OUT"
echo "Backup saved to $OUT"