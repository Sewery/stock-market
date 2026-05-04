param(
  [int]$Port = 8080
)

$BaseUrl = "http://localhost:$Port"

Write-Host "== go test =="
go test ./...

Write-Host "== E2E curl =="
./scripts/test_endpoints.ps1 -BaseUrl $BaseUrl

Write-Host "== k6 load =="
./scripts/k6/run.ps1

Write-Host "All tests OK"