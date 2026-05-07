# ── Stage 1: Build ──────────────────────────────────────────
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Cache dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /api ./cmd/api

# ── Stage 2: Run ────────────────────────────────────────────
FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata wget

# Copy the binary
COPY --from=builder /api /api

# WhatsApp session data volume
VOLUME ["/data/whatsapp"]

EXPOSE 8080

ENTRYPOINT ["/api"]
