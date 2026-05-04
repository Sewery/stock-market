#!/usr/bin/env bash
set -euo pipefail

PORT="${1:-8080}"
export PORT

docker compose up --build