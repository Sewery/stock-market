param(
  [string]$Out = "backup.sql",
  [string]$Service = "postgres"
)

$Container = docker compose ps -q $Service
if (-not $Container) {
  Write-Error "Container for service '$Service' not found"
  exit 1
}

docker exec -t $Container pg_dump -U stock -d stock_market > $Out
Write-Host "Backup saved to $Out"