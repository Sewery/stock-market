param(
  [int]$Port = 8080
)

$BaseUrl = "http://localhost:$Port"

Write-Host "== go test =="
go test ./...

Write-Host "== E2E curl =="
bash ./scripts/test_endpoints.sh $BaseUrl

Write-Host "== k6 load =="
bash ./scripts/k6/run.sh

Write-Host "All tests OK"