#!/usr/bin/env bash
set -euo pipefail

PORT="${1:-8080}"
BASE_URL="http://localhost:${PORT}"

echo "== go test =="
go test ./...

echo "== E2E curl =="
./scripts/test_endpoints.sh "$BASE_URL"

echo "== k6 load =="
./scripts/k6/run.sh

echo "All tests OK"