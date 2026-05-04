# Multi-stage build: build Go binary, then run in a minimal alpine image with ffmpeg

# ----- Builder stage -----
FROM golang:1.25-alpine AS builder

# git is needed by some Go module fetchers
RUN apk add --no-cache git

WORKDIR /app

# Pre-copy go.mod/go.sum to leverage Docker layer cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build a fully static binary (CGO disabled → no libc dependency)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /app/server ./cmd

# ----- Runtime stage -----
FROM alpine:3.21

# ffmpeg + CA certs + timezone data — all in one layer, cache cleared
RUN apk add --no-cache \
        ffmpeg \
        ca-certificates \
        tzdata

WORKDIR /app

# Copy compiled binary from builder
COPY --from=builder /app/server /app/server

# Create runtime directories (app also ensures them at startup)
RUN mkdir -p /app/uploads /app/outputs

ENV GIN_MODE=release

EXPOSE 8080

ENTRYPOINT ["/app/server"]