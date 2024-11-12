# Build stage
FROM golang:1.23.2-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/paste69 ./cmd/server

# Final stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata sqlite

# Create app directory and uploads directory
WORKDIR /app
RUN mkdir -p /app/uploads

# Copy binary from builder
COPY --from=builder /app/paste69 .

# Copy config and views
COPY config/config.yaml ./config/
COPY views ./views
COPY public ./public

# Create non-root user
RUN adduser -D -g '' appuser && \
    chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose default port
EXPOSE 3000

# Set environment variables
ENV GIN_MODE=release

# Run the application
CMD ["./paste69"]