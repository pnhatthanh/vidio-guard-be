# Multi-stage build: build Go binary, then run in a slim image with ffmpeg

# ----- Builder stage -----
FROM golang:1.25 AS builder

WORKDIR /app

# Pre-copy go.mod/go.sum to leverage Docker cache
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the rest of the source
COPY . .

# Build the binary (strip debug symbols to keep it small)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o server ./cmd

# ----- Runtime stage -----
FROM debian:12-slim

# Install ffmpeg and CA certs (clean apt caches to keep the image small)
RUN apt-get update \
     && apt-get install -y --no-install-recommends \
         ffmpeg \
         ca-certificates \
     && apt-get clean \
     && rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/*

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server /app/server

# Create folders for uploads and outputs (app also ensures them, this is just explicit)
RUN mkdir -p /app/uploads /app/outputs

ENV GIN_MODE=release
EXPOSE 8080

CMD ["/app/server"]
