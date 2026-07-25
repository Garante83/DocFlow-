# Dockerfile for Dokumentenscanner Backend
# Multi-stage build for smaller final image

# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go module files
COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download

# Copy source code
COPY backend/ ./backend/

# Build the server binary
RUN cd backend && \
    CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/server

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install CA certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=builder /app/server /app/server

# Copy default config files (can be overridden by volume mount)
COPY --from=builder /app/backend/config /app/config

# Create non-root user for security
RUN adduser -D -u 1000 appuser && \
    chown -R appuser:appuser /app

USER appuser

# Expose default port (can be overridden via DSCAN_SERVER_PORT)
EXPOSE 8082

# Run the server
CMD ["/app/server"]
