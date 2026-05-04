# Stock market
Project implementing service that simulates simplified stock market.
## About
Service consists of following entites:
- Wallets - can own various number of various stocks
- Bank - controls how number of available stocks, sells and buys them
- Audit log - a log of all actions that happened on the user wallets, without bank operations
### Asumptions
- Stock price is always 1
- Wallet balance is not tracked
- Buy/sell operations are always executed immediately at face value
- The Bank acts as the sole liquidity provider
- Initially there are no wallets and bank account is empty
### Endpoints
- `POST /wallets/{wallet_id}/stocks/{stock_name}`
    - simulates sel or but of a single stock
- `GET /wallets/{wallet_id}`
    - returns current state of the particular wallet
- `GET /wallets/{wallet_id}/stocks/{stock_name}`
    - returns current state of the bank
- `POST /stocks`
    - sets the state of the bank 
- `GET /log`
    - returns entire audit log of succesful operations in order of occurrence
- `POST /chaos`
    - kills an instance that serves this request.
### Non functional requirmetns
- Working on all major operating systems
- Can be started using one command
- Solution is highly available, so killing 1 instance doesn’t kill the product
## Setup
### Prerequisites
- Docker + Docker Compose
- (optional) Go 1.22+ run without docker

### Run (one command, port as parameter)

Linux/macOS (bash):
```bash
PORT=8085 docker compose up --build
```

Windows (PowerShell):
```powershell
$env:PORT=8085; docker compose up --build
```

The App will be available at:
```
http://localhost:8085
```

The port is a startup parameter. Inside the containers the app listens on 8080, and nginx maps the host port.

### Run without Docker (optional)
```bash
HOST=0.0.0.0 PORT=8085 go run ./cmd/server
```

## Tests
These tests assume that you have a running instance (via Docker Compose or without it). You can pass the port of the running endpoint as a parameter.
### Unit/contract tests
Unit/contract tests validate store correctness and concurrency, while the Postgres integration test verifies atomicity under concurrent buys.  
Run them with `go test -p 1 ./..` from root directory (set `DATABASE_URL` to enable Postgres tests).
If you want to run them in container type:

Linux/macOS:
```bash
docker compose run --rm -v "$PWD":/app -w /app app1 go test -p 1 ./...
```

Windows (PowerShell):
```powershell
docker compose run --rm -v ${PWD}:/app -w /app app1 go test -p 1 ./...
```

### Basic E2E test
Basic E2E tests validate the full request flow against a running instance
(starting from bank seeding, through buy/sell, to audit log verification). The test assumes an empty bank at start.

Linux/macOS:
```bash
test_endpoints.sh http://localhost:8085
```

Windows (PowerShell):
```powershell
test_endpoints.ps1 -BaseUrl http://localhost:8085
```
### Load/Stress tests
k6 (Grafana) is used to run E2E load tests. It generates concurrent traffic and reports key metrics such as failure rate and average latency.  
The experimental Prometheus remote‑write output is used to export k6 metrics.

Scenarios:

- **read-heavy**: read traffic that repeatedly queries bank stocks and wallets.
- **write-heavy**: write traffic that creates many wallets and buy operations.
- **mixed**: realistic mix of both reads and writes.
- **fanout-buy**: many wallets buy the same stock, each in a random size, simulating demand.
- **hot-wallet**: worst-case contention where a few users repeatedly buy/sell the same wallet and stock; failures are expected due to the unrealistic scenario. 
- **chaos-ha**: steady load while killing one instance to verify high availability and recovery.

Linux/macOS:
```bash
bash ./scripts/k6/run.sh  8085
```

Windows (PowerShell):
```powershell
./scripts/k6/run.ps1  -Port 8085
```
### Full test suite

Linux/macOS:
```bash
test_all.sh 8085
```

Windows (PowerShell):
```powershell
test_all.ps1 -Port 8085
```
### Database backup and restore (PostgreSQL)

Linux/macOS:
```bash
backup_db.sh backup.sql.
restore_db.sh backup.sql
```
Windows (PowerShell):
```powershell
./scripts/db/backup_db.ps1 -Out backup.sql
./scripts/db/restore_db.ps1 -In backup.sql
```

## Architecture
### Data flow
Nginx receives traffic on a single host port, distributes requests to app1/app2, and the app uses either in‑memory store or Postgres.
The in‑memory store is a lightweight fallback used when `DATABASE_URL` is not set, useful for fast local runs and tests without Postgres.
### Healtchecks
The app exposes a lightweight endpoint `GET \healthz`  used for container health checks in Docker Compose

### Load balancer and entry point
Nginx acts as the entry point and reverse proxy. It distributes traffic round‑robin and uses timeouts and failover on 5xx/timeout errors.

Running two app instances behind nginx ensures the service remains available when one instance is killed (validated by the chaos test).

### Logging, Monitoring and Observability
The app emits structured JSON logs (request + error logs). Promtail ships container logs to **Loki**, and Grafana provides a **Logs dashboard** for request volume, slow requests, and recent HTTP logs.  
**Prometheus** scrapes app metrics and evaluates alert rules (up/down, 5xx rate, p95 latency). Grafana provides a **Metrics dashboard** for throughput, error rate, and latency.
Prometheus uses alert rules (up/down, 5xx error rate, p95 latency)
## Limitations and trade-offs
- Hot‑wallet is a worst‑case contention test - failures are expected.
- No autoscaling in Docker Compose.
- No caching by design to keep data consistent.

## Code Documentation
Code is organized by feature and layer:
- `cmd/server` – application entrypoint and config
- `internal/api` – HTTP handlers, routing, middleware
- `internal/store` – storage interface + implementations (memory, postgres)
- `internal/domain` – request/response models
- `migrations` – SQL schema and indexes