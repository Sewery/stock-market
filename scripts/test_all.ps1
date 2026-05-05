param(
  [int]$Port = 8080
)

$BaseUrl = "http://localhost:$Port"
if (-not $env:DATABASE_URL) {
  $env:DATABASE_URL = "postgres://stock:stock@localhost:5432/stock_market?sslmode=disable"
}

Write-Host "== go test =="
go test -p 1 ./...

Write-Host "== E2E curl =="
./scripts/test_endpoints.ps1 -BaseUrl $BaseUrl

Write-Host "== k6 load =="
./scripts/k6/run.ps1

Write-Host "All tests OK"