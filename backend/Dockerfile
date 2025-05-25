# Multi-stage build for Kasaneha backend
FROM golang:1.23-alpine AS builder

# Install git
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the API server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/api

# Build the batch processor
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o batch ./cmd/batch

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests and bash for scripts
RUN apk --no-cache add ca-certificates bash curl

WORKDIR /app

# Copy binaries from builder stage
COPY --from=builder /app/main .
COPY --from=builder /app/batch .

# Copy scripts
COPY scripts/ ./scripts/

# Make scripts executable
RUN chmod +x ./scripts/*.sh

# Create log directory
RUN mkdir -p /var/log/kasaneha

# Expose port for API
EXPOSE 8080

# Default command runs the API server
CMD ["./main"]
