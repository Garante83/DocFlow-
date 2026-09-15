# Dockerfile for DocFlow Backend
# Multi-stage build: frontend -> backend -> runtime
# Produces a self-contained image with the embedded frontend.

# Stage 1: Build frontend
FROM node:22-alpine AS frontend

WORKDIR /app/frontend

# Copy frontend package files and install dependencies
COPY frontend/dokumentenscanner/package.json frontend/dokumentenscanner/package-lock.json ./
RUN npm ci

# Copy frontend source and build
COPY frontend/dokumentenscanner/ ./
RUN npm run build

# Stage 2: Build backend binary with embedded frontend
FROM golang:1.26-alpine AS backend

WORKDIR /app/backend

# Copy go module files and download dependencies
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy backend source
COPY backend/ ./

# Copy fresh frontend build into the embed directory
COPY --from=frontend /app/frontend/dist/index.html cmd/server/index.html
COPY --from=frontend /app/frontend/dist/favicon.ico cmd/server/favicon.ico
COPY --from=frontend /app/frontend/dist/favicon.svg cmd/server/favicon.svg
COPY --from=frontend /app/frontend/dist/assets/ cmd/server/assets/

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./cmd/server

# Stage 3: Runtime
FROM alpine:latest

WORKDIR /app

# Install CA certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=backend /app/server /app/server

# Copy default config files (can be overridden by volume mount)
COPY --from=backend /app/backend/config /app/config

# Create non-root user for security
RUN adduser -D -u 1000 appuser && \
    chown -R appuser:appuser /app

USER appuser

# Expose default port (can be overridden via DSCAN_SERVER_PORT)
EXPOSE 8082

# Run the server
CMD ["/app/server"]
