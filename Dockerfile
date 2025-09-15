# Multi-stage build for Traffic Monitor
FROM node:18-alpine AS frontend-builder

# Set working directory for frontend
WORKDIR /app/frontend

# Copy frontend package files
COPY frontend/package*.json ./

# Install frontend dependencies
RUN npm ci --only=production

# Copy frontend source
COPY frontend/ ./

# Build frontend
RUN npm run build

# Backend builder stage
FROM golang:1.21-alpine AS backend-builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory for backend
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy backend source
COPY backend/ ./backend/
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/

# Build backend
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o traffic-sniff cmd/server/main.go

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    libpcap-dev \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy backend binary
COPY --from=backend-builder /app/traffic-sniff ./

# Copy frontend build
COPY --from=frontend-builder /app/frontend/dist ./frontend/

# Create data directory
RUN mkdir -p data && chown -R appuser:appgroup /app

# Switch to non-root user (Note: packet capture typically requires root)
# USER appuser

# Expose port
EXPOSE 8080

# Add health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

# Default command
CMD ["./traffic-sniff", "-port=8080", "-interface=eth0", "-storage=./data"]