# ==========================================
# Eero-Stats Dockerfile
# ==========================================

# -- Stage 1: Build --
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Copy go mod and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Download dependencies and build
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /app-binary ./cmd/eero-stats

# -- Stage 2: Runtime --
FROM alpine:latest

# Install CA certificates for HTTPS requests to Eero API and timezone data
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Add an unprivileged user to execute the app securely
RUN addgroup -S eero && adduser -S eero -G eero

# The daemon expects /app/data to be mounted for session persistence
RUN mkdir -p /app/data && chown -R eero:eero /app

# Drop root privileges by shifting user context
USER eero

# Copy the compiled binary from the builder stage
COPY --chown=eero:eero --from=builder /app-binary /app/eero-stats

# Ensure it's executable
RUN chmod +x /app/eero-stats

# Command to run
CMD ["/app/eero-stats"]
