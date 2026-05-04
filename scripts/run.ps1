param(
  [int]$Port = 8080
)

$env:PORT = $Port
docker compose up --build