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

## Run Tests

```bash
go test ./... -race
```
