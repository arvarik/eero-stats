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

WORKDIR /app

# The daemon expects /app/data to be mounted for session persistence
RUN mkdir -p /app/data

# Copy the compiled binary from the builder stage
COPY --from=builder /app-binary /app/eero-stats
RUN chmod +x /app/eero-stats

# User is controlled via docker-compose.yml user: directive
# so we do NOT hardcode USER here — this allows running as any UID.

CMD ["/app/eero-stats"]
