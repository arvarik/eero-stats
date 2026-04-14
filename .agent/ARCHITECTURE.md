# Architecture & System Design

## 1. System Overview
**Name:** `eero-stats`
**Description:** A lightweight Go daemon that extracts real-time metrics from an Eero Mesh Network and writes them to an InfluxDB v2 time-series database. Designed for minimal flash storage wear.

## 2. Tech Stack & Dependencies
### Language & Runtime
*   **Go:** `1.22.0`

### Core Libraries & SDKs
*   **Eero API Client:** `github.com/arvarik/eero-go` (v1.1.0)
*   **InfluxDB Client:** `github.com/influxdata/influxdb-client-go/v2` (v2.14.0)
*   **Env Config:** `github.com/joho/godotenv` (v1.5.1)

### External Services & Providers
*   **Eero Cloud API:** Source of network telemetry.
*   **InfluxDB v2:** Time-series database for metric storage.
*   **Grafana:** Visualization dashboard.

## 3. Data Flow & Routing
1.  **Authentication:** Interactive OTP via stdin -> Session Cache (`.eero_session.json`).
2.  **Polling:** The daemon polls the Eero API using three tiers (Fast: 3min, Medium: 90min, Slow: 12hr).
3.  **Batching:** Metrics are converted to InfluxDB points and batched in memory to minimize NVMe/flash wear.
4.  **Storage:** Batched points are written asynchronously to InfluxDB v2 via the Go client.
5.  **Visualization:** Grafana queries InfluxDB via Flux.

## 4. Concurrency & Event Loop Model
*   **Goroutines:** The daemon uses `sync.WaitGroup` to orchestrate multiple polling loops. Each polling tier runs in its own goroutine with an immediate initial poll followed by periodic ticks (`time.Ticker`).
*   **Panic Recovery:** Polling goroutines are wrapped in panic-safe functions (`safePollFast`, `safePollMedium`, `safePollSlow`) to prevent a crash in one tier from taking down the daemon.
*   **Context & Graceful Shutdown:** `context.Context` is used to propagate cancellation. The main function listens for `SIGINT` / `SIGTERM`, cancels the context, waits for the `WaitGroup`, and performs a graceful 15-second shutdown flush of the InfluxDB client.

## 5. Environment & Configuration
Loaded from `.env` via `github.com/joho/godotenv`.
*   `PUID`, `PGID`: Container user mapping.
*   `EERO_LOGIN`: (Required) Email or phone for Eero Owner Account.
*   `EERO_SESSION_PATH`: Path to the cached session file.
*   `INFLUX_URL`: InfluxDB URL (default: `http://influxdb:8086`).
*   `INFLUX_TOKEN`: InfluxDB API token.
*   `INFLUX_ORG`: InfluxDB organization.
*   `INFLUX_BUCKET`: InfluxDB bucket.
*   `GRAFANA_PORT`: Grafana exposed port.
*   `INFLUX_PORT`: InfluxDB exposed port.
*   `GF_ADMIN_PASSWORD`: Grafana admin password.

## 6. Docker & Infrastructure Topology
The `docker-compose.yml` defines the full stack:
*   `eero-stats`: The Go daemon container. Builds from local Dockerfile. Needs `tty: true` and `stdin_open: true` for initial 2FA. Maps `./data/app` volume.
*   `influxdb`: InfluxDB 2.7 container. Maps `./data/influxdb` volume. Auto-bootstrapped using `DOCKER_INFLUXDB_INIT_*` env vars.
*   `grafana`: Grafana OSS container. Maps `./data/grafana` volume, provisions datasources and dashboards from `./grafana/provisioning/`.

## 7. Safety Invariants & Architectural Rules
*   **NEVER** block the main polling loop. Each tier must run in its own goroutine.
*   **ALWAYS** use panic-safe goroutine wrappers for long-running background tasks.
*   **NEVER** write un-batched metrics directly to disk. Aggressive memory batching is required to save NVMe/flash wear on NAS systems.
*   **ALWAYS** gracefully flush buffered writes to InfluxDB on `SIGTERM`.
