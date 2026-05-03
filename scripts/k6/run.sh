#!/bin/bash

set -e

NETWORK=stock-market_default
BASE_URL=http://nginx:8080
PROM_URL=http://prometheus:9090/api/v1/write
IMAGE=grafana/k6:latest

run_k6() {
  local script=$1
  echo "Starting $script..."
  docker run --rm -i \
    --network "$NETWORK" \
    -e BASE_URL="$BASE_URL" \
    -e K6_PROMETHEUS_RW_SERVER_URL="$PROM_URL" \
    "$IMAGE" run \
    --out experimental-prometheus-rw \
    - < "scripts/k6/$script"
}

run_k6 read-heavy.js
run_k6 write-heavy.js
run_k6 chaos-ha.js
run_k6 hot-wallet.js

echo "All k6 tests finished."