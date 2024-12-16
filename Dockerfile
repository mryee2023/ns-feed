# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application with CGO enabled
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/ns-rss ./src/main.go

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sqlite-libs

WORKDIR /app

# Create necessary directories with proper permissions
RUN mkdir -p /etc && mkdir -p /db && chown -R nobody:nobody /db

# Copy the binary from builder
COPY --from=builder /app/ns-rss .

# Switch to non-root user
USER nobody

# Set the entrypoint
ENTRYPOINT ["./ns-rss"]
CMD ["-f", "/etc/config.yaml", "-db", "/db/sqlite.db"]
