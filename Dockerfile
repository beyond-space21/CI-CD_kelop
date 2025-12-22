# -------- Build stage --------
    FROM golang:1.24-bookworm AS builder

    WORKDIR /app
    
    # Copy only go mod files first (cache-friendly)
    COPY go.mod go.sum ./
    
    # Download dependencies (cached with BuildKit)
    RUN --mount=type=cache,target=/go/pkg/mod \
        --mount=type=cache,target=/root/.cache/go-build \
        go mod download
    
    # Copy application source
    COPY . .
    
    # Build binary
    RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
        go build -o main .
    
# -------- Runtime stage --------
    FROM gcr.io/distroless/static-debian12
    # FROM debian:bookworm-slim
    
    WORKDIR /app
    
    # Install CA certs (cached layer)
    # RUN apt-get update && \
    #     apt-get install -y --no-install-recommends ca-certificates && \
    #     update-ca-certificates && \
    #     rm -rf /var/lib/apt/lists/*
    
    # Copy binary and required files
    COPY --from=builder /app/main .
    COPY env .env
    COPY DB/migrations ./DB/migrations
    
    ENTRYPOINT ["/app/main"]
    