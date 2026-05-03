# Stock market

## About
## Requirements
### Endpoints
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

App will be available under:
```
http://localhost:8085
```

> The port is a startup parameter. Inside the containers the app listens on 8080, and nginx maps the host port.

### Run without Docker (optional)
```bash
HOST=0.0.0.0 PORT=8085 go run ./cmd/server
```

## Tests
### Stess tests
### Full test suite
Linux/macOS:
```bash
test_all.sh 8085
```

Windows (PowerShell):
```powershell
test_all.ps1 -Port 8085
```

## Architecture
### Transactions
### Logging, Monitoring and Observability
### Load balancer and entry point
Nginx is responsible for routing movement. It takes it on specified port by user, and directs incoming traffic into pool of apps using round-robin ( initiated by docker compose).  

Additionaly nginx is configured for timeouts to prevent hanging requests, connections. Also failover mechanism when errors occur to direct traffic to other instances
## Code Documentation