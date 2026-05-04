# Stock market
Service that simulates a simplified stock market.

## About
The service consists of three core entities:
- Wallets - can own various number of various stocks
- Bank - controls how number of available stocks, sells and buys them
- Audit log - records all wallet actions (bank operations are excluded)

### Assumptions
- Stock price is always 1
- Wallet balance is not tracked
- Buy/sell operations are always executed immediately at face value
- The Bank acts as the sole liquidity provider
- Initially there are no wallets and bank account is empty
### Endpoints
- `POST /wallets/{wallet_id}/stocks/{stock_name}`
    - buy or sell a single stock for a wallet
- `GET /wallets/{wallet_id}`
    - returns the current state of a wallet
- `GET /wallets/{wallet_id}/stocks/{stock_name}`
    - returns the wallet's quantity for the given stock
- `POST /stocks`
    - sets the state of the bank (full replace)
- `GET /log`
    - returns the audit log of successful wallet operations in order of occurrence
- `POST /chaos`
    - kills the instance that handles this request

### Non-functional requirements
- Works on all major operating systems
- Starts with a single command
- Highly available: killing one instance does not take down the service

## Setup
### Prerequisites
- Docker + Docker Compose
- (optional) Go 1.22+ to run without Docker

### Run (one command, port as parameter)

Linux/macOS (bash):
```bash
PORT=8085 docker compose up --build
```

Windows (PowerShell):
```powershell
$env:PORT=8085; docker compose up --build
```

The app will be available at:
```
http://localhost:8085
```

Inside the containers the app listens on 8080, and nginx maps the host port.  
If `DATABASE_URL` is not set, the app uses an in-memory store (no Postgres required).

### Run without Docker (optional)
```bash
HOST=0.0.0.0 PORT=8085 go run ./cmd/server
```

## Tests
These tests assume you have a running instance (via Docker Compose or without it). You can pass the port of the running endpoint as a parameter.

### Unit/contract tests
Unit/contract tests validate store correctness and concurrency. The Postgres integration test verifies atomicity under concurrent buys.  
Run from the repository root (set `DATABASE_URL` to enable Postgres tests):
```bash
go test -p 1 ./...
```

Run in a container:

Linux/macOS:
```bash
docker compose run --rm -v "$PWD":/app -w /app app1 go test -p 1 ./...
```

Windows (PowerShell):
```powershell
docker compose run --rm -v ${PWD}:/app -w /app app1 go test -p 1 ./...
```

### Basic E2E test
Validates the full request flow against a running instance (bank seeding, buy/sell, audit log verification).  
**The test assumes an empty bank at the start.**

Linux/macOS:
```bash
scripts/test_endpoints.sh http://localhost:8085
```

Windows (PowerShell):
```powershell
scripts/test_endpoints.ps1 -BaseUrl http://localhost:8085
```

### Load/Stress tests
k6 (Grafana) is used to run E2E load tests. It generates concurrent traffic and reports key metrics such as failure rate and average latency.  
The experimental Prometheus remote-write output is used to export k6 metrics.

Scenarios:
- **read-heavy**: repeated reads of bank stocks and wallets
- **write-heavy**: creates many wallets and executes buy operations
- **mixed**: realistic mix of reads and writes
- **fanout-buy**: many wallets buy the same stock with random sizes
- **hot-wallet**: worst-case contention on a few wallets and stocks; failures are expected
- **chaos-ha**: steady load while killing one instance to verify HA and recovery

Linux/macOS:
```bash
bash ./scripts/k6/run.sh 8085
```

Windows (PowerShell):
```powershell
./scripts/k6/run.ps1 -Port 8085
```

### Full test suite

Linux/macOS:
```bash
scripts/test_all.sh 8085
```

Windows (PowerShell):
```powershell
scripts/test_all.ps1 -Port 8085
```

### Database backup and restore (PostgreSQL)

Linux/macOS:
```bash
scripts/db/backup_db.sh backup.sql
scripts/db/restore_db.sh backup.sql
```

Windows (PowerShell):
```powershell
./scripts/db/backup_db.ps1 -Out backup.sql
./scripts/db/restore_db.ps1 -In backup.sql
```

## Architecture

### Data flow
Nginx receives traffic on a single host port, distributes requests to app1/app2, and the app uses either the in-memory store or Postgres.  
The in-memory store is a lightweight fallback used when `DATABASE_URL` is not set, useful for fast local runs and tests without Postgres.

### Service ports
| Service | Host port |
|---|---|
| App (via nginx) | `${PORT}` (default 8080) |
| Prometheus | 9090 |
| Loki | 3100 |
| Grafana | 3000 |
| Postgres | 5432 |

### Healthchecks
The app exposes a lightweight endpoint `GET /healthz` used for container health checks in Docker Compose.

### Load balancer and entry point
Nginx acts as the entry point and reverse proxy. It distributes traffic round-robin and uses timeouts and failover on 5xx/timeout errors.  
Running two app instances behind nginx keeps the service available when one instance is killed (validated by the chaos test).

### Logging, monitoring and observability
The app emits structured JSON logs (request + error logs). Promtail ships container logs to **Loki**, and Grafana provides a **Logs dashboard** for request volume, slow requests, and recent HTTP logs.  
**Prometheus** scrapes app metrics and evaluates alert rules (up/down, 5xx rate, p95 latency). Grafana provides a **Metrics dashboard** for throughput, error rate, and latency.  
Grafana dashboards are provisioned from `configs/grafana/dashboards/`.

### Logging configuration
Logging is controlled via environment variables (see `docker-compose.yml` for defaults):

- `LOG_ENABLED`: enable/disable logging (`true`/`false`)
- `LOG_LEVEL`: log level (`debug`, `info`, `warn`, `error`)
- `LOG_FORMAT`: output format (`json` or `text`)
- `LOG_HTTP`: enable HTTP request logging (`true`/`false`)
- `LOG_REQUEST_ID`: include request ID in logs (`true`/`false`)

Logs are written to stdout and collected by Promtail, then shipped to Loki for viewing in Grafana.

## Limitations and trade-offs
- No authentication or authorization; the API is public.
- No rate limiting.
- Migrations are linear and simple.
- No real autoscaling.
- Grafana/Loki/Prometheus use ephemeral storage; data is lost on restart.
- Hot-wallet is a worst-case contention test; failures are expected.
- No caching by design to keep data consistent.

## Operations guide

### Health and status
To verify the system is healthy, check the `GET /healthz` endpoint on each app instance — it returns 200 when the instance is up (this is also what Docker Compose uses for its healthcheck). For a broader view, open Prometheus at `http://localhost:9090/targets` and confirm that both `app1:8080` and `app2:8080` are listed as `UP`. In Grafana (`http://localhost:3000`, login: admin/admin), verify that both the Loki and Prometheus data sources are reachable.

### Alerts (defined in `configs/prometheus_rules.yml`)
| Alert | Condition | Severity |
|---|---|---|
| `StockMarketDown` | `up == 0` for 1 min | critical |
| `High5xxErrorRate` | 5xx rate > 5% for 2 min | warning |
| `HighLatencyP95` | p95 > 200ms for 30s | warning |

### Key metrics (scraped every 5s from app1 and app2)
- `stock_market_http_requests_total{status, method, path}` — request count and error rate
- `stock_market_http_request_duration_seconds` — latency histogram (p50/p95/p99)
- `stock_market_trades_total{type}` — successful buy/sell count
- `stock_market_trade_errors_total{reason}` — errors by reason (`stock_not_found`, `bank_out_of_stock`, `wallet_out_of_stock`, `invalid_type`, `internal`)
- `stock_market_bank_set_total` — bank resets via `POST /stocks`

### Logs
Structured JSON logs are collected by Promtail and available in Grafana Loki.  
Key log messages to search for:

| Message | Meaning |
|---|---|
| `trade_failed` | Trade rejected; check `reason` field |
| `wallet_not_found` | Read/trade on a non-existent wallet |
| `bank_out_of_stock` | Buy attempted when bank has 0 stock |
| `http_request` | Per-request access log with latency |

### First response
1. Check `docker compose ps` — all services should be `healthy` or `running`.
2. If an app instance is down: `docker compose restart app1` (or `app2`).
3. If Postgres is down: `docker compose restart postgres` — apps will retry on the next request.
4. Check recent logs: `docker compose logs --tail=50 app1 app2`.
5. If nginx returns 502/504, verify that both app instances are healthy.

### Nginx failover behaviour
Nginx retries a request on the other instance on `error`, `timeout`, `502`, `503`, `504`.  
Each instance allows `max_fails=2` within a `fail_timeout=5s` window before being marked down.  
Connect timeout is 2s; read/send timeout is 15s.

### Backup and restore (PostgreSQL)
```bash
# backup
scripts/db/backup_db.sh backup.sql

# restore
scripts/db/restore_db.sh backup.sql
```
Both scripts resolve the Postgres container automatically via `docker compose ps -q`.

### Known failure modes
| Mode | Symptom | Action |
|---|---|---|
| Hot-wallet contention | High `bank_out_of_stock` / `wallet_out_of_stock` errors | Expected; not a bug |
| Postgres down | All trades return 500 | Restart postgres, check logs |
| Nginx timeout | 504 from nginx, high p95 | Check DB query times, restart slow instance |
| Promtail label error | No logs in Loki | Check `docker compose logs promtail` |
| Ephemeral storage | Metrics/logs lost after restart | Expected in this setup; use volumes for production |

## Code Documentation
Code is organized by feature and layer:
- `cmd/server` - application entrypoint and config
- `internal/api` - HTTP handlers, routing, middleware
- `internal/store` - storage interface + implementations (memory, postgres)
- `internal/store/errors.go` - domain error types (`ErrStockNotFound`, `ErrBankOutOfStock`, etc.)
- `internal/domain` - request/response models
- `migrations` - SQL schema and indexes

### Sample flow: trade one stock (POST /wallets/{wallet_id}/stocks/{stock_name})
1. The router registers the route and attaches middleware for request IDs, HTTP logging, and metrics.
2. The handler parses the JSON body into `TradeRequest` and calls the store.
3. The storage contract is defined by the `Store` interface in [internal/store/store.go](internal/store/store.go).
4. The in-memory implementation updates bank and wallet state under a mutex and appends to the audit log in [internal/store/memory/memory.go](internal/store/memory/memory.go).
5. The Postgres implementation runs a transaction with CTE updates and writes to `audit_log` in [internal/store/postgres/postgres.go](internal/store/postgres/postgres.go).
6. The handler increments Prometheus counters and returns the final status code in [internal/api/handlers.go](internal/api/handlers.go).