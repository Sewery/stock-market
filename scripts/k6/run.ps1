param(
  [string]$Network = "stock-market_default",
  [string]$BaseUrl = "http://nginx:8080",
  [string]$PromUrl = "http://prometheus:9090/api/v1/write"
)

$Image = "grafana/k6:latest"

function Run-K6($Script) {
  Write-Host "Starting $Script..."
  docker run --rm -i `
    --network $Network `
    -e BASE_URL=$BaseUrl `
    -e K6_PROMETHEUS_RW_SERVER_URL=$PromUrl `
    $Image run --out experimental-prometheus-rw - < "scripts/k6/$Script"
}

Run-K6 "read-heavy.js"
Run-K6 "write-heavy.js"
Run-K6 "mixed.js"
Run-K6 "chaos-ha.js"
Run-K6 "fanout-buy.js"
Run-K6 "hot-wallet.js"

Write-Host "All k6 tests finished."