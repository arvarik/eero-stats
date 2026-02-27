# ==========================================
# Eero-Stats Dockerfile
# ==========================================

# -- Stage 1: Build --
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Copy go mod and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build the binary with version metadata
COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w \
    -X github.com/arvarik/eero-stats/internal/version.Version=${VERSION} \
    -X github.com/arvarik/eero-stats/internal/version.Commit=${COMMIT} \
    -X github.com/arvarik/eero-stats/internal/version.BuildDate=${BUILD_DATE}" \
    -o /app-binary ./cmd/eero-stats

# -- Stage 2: Runtime --
FROM alpine:3.21

# Install CA certificates for HTTPS requests to Eero API and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Create a non-root user and group
RUN addgroup -S eero -g 1000 && \
    adduser -S eero -u 1000 -G eero
# Default session path for Docker environment
ENV EERO_SESSION_PATH=/app/data/.eero_session.json

WORKDIR /app

# The daemon expects /app/data to be mounted for session persistence
RUN mkdir -p /app/data && \
    chown -R eero:eero /app

# Copy the compiled binary from the builder stage with correct ownership
COPY --from=builder --chown=eero:eero /app-binary /app/eero-stats
RUN chmod +x /app/eero-stats

# Use the non-root user
USER eero

CMD ["/app/eero-stats"]
