# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /wallet ./cmd/server

FROM alpine:3.19
RUN adduser -D -u 1000 appuser
USER appuser
COPY --from=builder /wallet /wallet
EXPOSE 8080 5505
ENTRYPOINT ["/wallet"]
