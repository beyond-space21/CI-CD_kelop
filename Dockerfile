# Build stage
FROM golang:1.24-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy
COPY . .
RUN go build -o main .
FROM debian:bookworm-slim

WORKDIR /app
COPY --from=builder /app/main .
COPY .env .env
COPY DB/migrations ./DB/migrations
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    update-ca-certificates && \
    rm -rf /var/lib/apt/lists/*

ENTRYPOINT ["/app/main"]
