#!/usr/bin/env bash
set -euo pipefail

IN="${1:-backup.sql}"
SERVICE="${2:-postgres}"

CONTAINER="$(docker compose ps -q "$SERVICE")"
if [[ -z "$CONTAINER" ]]; then
  echo "Container for service '$SERVICE' not found"
  exit 1
fi

cat "$IN" | docker exec -i "$CONTAINER" psql -U stock -d stock_market
echo "Restore finished from $IN"