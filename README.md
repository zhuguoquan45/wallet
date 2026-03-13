# Wallet Service

A simple wallet service written in Go, exposing both a REST API and a gRPC API.

## Requirements

- Go 1.25+
- (Optional) Docker & Docker Compose
- (Optional) `protoc` + `protoc-gen-go` + `protoc-gen-go-grpc` for regenerating proto files

## Run

### In-memory (default)

```bash
go run ./cmd/server
```

### With PostgreSQL

```bash
STORAGE_TYPE=postgres POSTGRES_DSN="postgres://user:pass@localhost/wallet?sslmode=disable" go run ./cmd/server
```

The database is created automatically if it does not exist.

The service starts:
- HTTP on `:8080`
- gRPC on `:5505`

Override addresses via environment variables:

```bash
HTTP_ADDR=:9090 GRPC_ADDR=:9091 go run ./cmd/server
```

## Docker

Starts the wallet service + PostgreSQL with a single command:

```bash
docker compose up --build
```

| Service  | Port  |
|----------|-------|
| HTTP     | 8080  |
| gRPC     | 5505  |

## REST API

Amounts are in **cents** (integer). e.g. `1000` = $10.00.

Import `api.openapi.json` into Apifox / Postman for a full interactive collection.

### Create Wallet

```bash
curl -X POST http://localhost:8080/wallets
# {"id":"...","balance":0}
```

### Get Wallet

```bash
curl http://localhost:8080/wallets/{wallet_id}
# {"id":"...","balance":0}
```

### Deposit

```bash
curl -X POST http://localhost:8080/wallets/{wallet_id}/deposit \
  -H "Content-Type: application/json" \
  -d '{"amount":1000}'
# {"id":"...","balance":1000}
```

### Transfer Funds

```bash
curl -X POST http://localhost:8080/wallets/transfer \
  -H "Content-Type: application/json" \
  -d '{"from_id":"<id>","to_id":"<id>","amount":500}'
# 204 No Content on success
```

## gRPC API

Proto definition: [`proto/wallet/wallet.proto`](proto/wallet/wallet.proto)

Regenerate generated code:

```bash
make proto
```

## Load / Concurrency Testing

A built-in Go load-test tool is included under [`loadtest/`](loadtest/).

It pre-creates wallets, funds them, then hammers the service concurrently with a mix of **get / deposit / transfer** requests and prints a summary at the end.

```bash
# default: 50 workers, 30s against http://localhost:8080
go run ./loadtest

# custom
go run ./loadtest -addr http://localhost:8080 -workers 100 -duration 60s -wallets 30

# via make (override defaults with env vars)
make loadtest ADDR=http://localhost:8080 WORKERS=100 DURATION=60s
```

Sample output:

```
Wallet Load Test
  addr=http://localhost:8080  wallets=20  workers=50  duration=30s

Creating 20 wallets...
Setup done. Starting 50 workers for 30s...

  progress: total=12483    ok=12391    fail=92     avg_lat=11.43ms
  progress: total=24901    ok=24762    fail=139    avg_lat=11.38ms
  ...

── Results ──────────────────────────────────────
  Duration   : 30s
  Workers    : 50
  Total reqs : 37254
  Success    : 37089 (99.6%)
  Failed     : 165
  Throughput : 1241.8 req/s
  Avg latency: 11.41 ms
```

## Run Tests

```bash
go test ./... -race
```
