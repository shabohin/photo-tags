# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o processor ./cmd/main.go

# Final stage
FROM alpine:latest

# Install ExifTool and runtime dependencies
RUN apk add --no-cache \
    exiftool \
    perl \
    ca-certificates \
    tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/processor .

# Create temp directory for image processing
RUN mkdir -p /tmp/processor && chmod 777 /tmp/processor

# Run as non-root user
RUN adduser -D -u 1000 processor && \
    chown -R processor:processor /app /tmp/processor

USER processor

# Run the application
CMD ["./processor"]
