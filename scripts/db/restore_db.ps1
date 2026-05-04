param(
  [string]$In = "backup.sql",
  [string]$Service = "postgres"
)

$Container = docker compose ps -q $Service
if (-not $Container) {
  Write-Error "Container for service '$Service' not found"
  exit 1
}

Get-Content $In | docker exec -i $Container psql -U stock -d stock_market
Write-Host "Restore finished from $In"