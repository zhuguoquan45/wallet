.PHONY: run proto tidy docker-build docker-up test loadtest

run:
	go run ./cmd/server

tidy:
	go mod tidy

proto:
	protoc --go_out=gen/wallet --go_opt=paths=source_relative \
	       --go-grpc_out=gen/wallet --go-grpc_opt=paths=source_relative \
	       -I proto/wallet proto/wallet/wallet.proto

docker-build:
	docker build -t wallet:latest .

docker-up:
	docker compose up --build

test:
	go test ./... -v -race

loadtest:
	go run ./loadtest -addr=$(ADDR) -workers=$(WORKERS) -duration=$(DURATION)
