# Build Stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make build-base

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o prompt-cache ./cmd/api

# Final Stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create a non-root user and a writable data directory. Ownership is
# additionally granted to the root group (0) with group=user permissions
# so the image also runs under an arbitrary UID, as required by platforms
# like OpenShift that enforce non-root, non-fixed-UID security contexts.
RUN addgroup -g 1000 promptcache \
    && adduser -u 1000 -G promptcache -D -h /app promptcache \
    && mkdir -p /app/badger_data \
    && chown -R promptcache:0 /app \
    && chmod -R g=u /app

WORKDIR /app

# Copy binary from builder
COPY --from=builder --chown=promptcache:0 /app/prompt-cache .

USER promptcache

# Expose port
EXPOSE 8080

# Run
CMD ["./prompt-cache"]
