# Build stage
FROM golang:1.23-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code and optional graph file
COPY *.go ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o motion-planner .

# Runtime stage
FROM alpine:latest

# Install ca-certificates and wget for health checks
RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/motion-planner .

# Copy the graph file if it exists in the build context
# Note: This will fail build if file doesn't exist. Comment out if not needed.
# COPY prm_graph.json ./

# Expose port
EXPOSE 8080

# Run the application
CMD ["./motion-planner"]
