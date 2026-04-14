# Architecture & System Design

## 1. System Overview
**Name:** `eero-stats`
**Repository:** `github.com/arvarik/eero-stats`
**Description:** A lightweight Go daemon that extracts real-time metrics from an Eero Mesh Network and writes them to an InfluxDB v2 time-series database. Designed for minimal flash storage wear on NAS systems (TrueNAS SCALE, Unraid, Proxmox). Ships with a fully automated Grafana dashboard covering network health, device signal quality, node telemetry, and more.

## 2. Tech Stack & Dependencies

### Language & Runtime
*   **Go:** `1.22.0` (specified in `go.mod`, CI, and Dockerfile)
*   **Python 3:** Used for `scripts/build_dashboard.py` (Grafana dashboard-as-code generator, stdlib only — no pip dependencies)

### Core Libraries & SDKs
*   **Eero API Client:** `github.com/arvarik/eero-go` (v1.1.0) — First-party Go client for the undocumented Eero cloud API.
*   **InfluxDB Client:** `github.com/influxdata/influxdb-client-go/v2` (v2.14.0) — Official Go client with non-blocking async write API.
*   **Env Config:** `github.com/joho/godotenv` (v1.5.1) — Loads `.env` files; gracefully no-ops if absent.

### Transitive Dependencies (indirect)
*   `github.com/apapsch/go-jsonmerge/v2` (v2.0.0)
*   `github.com/google/uuid` (v1.3.1)
*   `github.com/influxdata/line-protocol` (v0.0.0)
*   `github.com/oapi-codegen/runtime` (v1.0.0)
*   `golang.org/x/net` (v0.23.0)

### External Services & Providers
*   **Eero Cloud API:** Source of network telemetry. Undocumented, rate-limited consumer API.
*   **InfluxDB v2.7:** Time-series database for metric storage. Auto-bootstrapped via `DOCKER_INFLUXDB_INIT_*` env vars.
*   **Grafana OSS (latest):** Visualization dashboard. Auto-provisioned via YAML configs.

## 3. Project Layout

```
eero-stats/
├── .agent/                     # AI agent orchestration files (this directory)
├── .github/
│   ├── ISSUE_TEMPLATE/         # Bug report + feature request YAML templates
│   │   ├── bug-report.yml
│   │   ├── config.yml
│   │   └── feature-request.yml
│   └── workflows/
│       └── ci.yml              # GitHub Actions CI: tidy, build, vet, test, lint
├── .githooks/
│   └── pre-commit              # Shell script: go vet + golangci-lint before commits
├── cmd/
│   └── eero-stats/
│       └── main.go             # Daemon entry point: config, auth, poller, shutdown
├── internal/
│   ├── auth/
│   │   └── auth.go             # Eero API authentication (session cache + interactive 2FA)
│   ├── config/
│   │   ├── config.go           # Env var loading + validation (all InfluxDB vars required)
│   │   └── config_test.go      # Table-driven tests for config validation
│   ├── db/
│   │   └── influx.go           # InfluxDB client wrapper (NVMe-optimized batching)
│   ├── poller/
│   │   ├── interfaces.go       # EeroClient + MetricWriter interfaces
│   │   ├── adapter.go          # EeroClientAdapter: adapts *eero.Client → EeroClient
│   │   ├── poller.go           # Tiered poll loop orchestration (Fast/Medium/Slow)
│   │   ├── poller_test.go      # Integration test: full poller lifecycle with mocks
│   │   ├── writers.go          # InfluxDB data point writers (8 measurement types)
│   │   ├── writers_test.go     # Unit test: resolveDeviceName fallback chain
│   │   ├── retry.go            # Exponential backoff retry helper
│   │   └── retry_test.go       # Unit tests: retry success, exhaustion, ctx cancel
│   └── version/
│       └── version.go          # Build-time metadata (Version, Commit, BuildDate via ldflags)
├── grafana/
│   ├── dashboards/
│   │   └── eero.json           # Auto-generated Grafana dashboard JSON (~55 KB)
│   └── provisioning/
│       ├── dashboards/
│       │   └── dashboards.yml  # Grafana dashboard provider config
│       └── datasources/
│           └── datasource.yml  # InfluxDB Flux datasource config (UID: P951FEA4DE68E13C5)
├── scripts/
│   └── build_dashboard.py      # Python dashboard-as-code generator (755 lines, 7 sections)
├── data/                       # Runtime data (gitignored, Docker volume mount point)
│   └── app/                    # Session file storage (`.eero_session.json`)
├── docker-compose.yml          # Full stack: daemon + InfluxDB 2.7 + Grafana OSS
├── Dockerfile                  # Multi-stage build: golang:1.22-alpine → alpine:3.21
├── Makefile                    # Build, lint, test, Docker, dashboard, setup targets
├── .dockerignore               # Excludes data/, .env, *.md from Docker context
├── .env.example                # Template for all environment variables
├── .gitignore                  # Excludes bin/, data/, .env, IDE files
├── .golangci.yml               # Linter config: errcheck, govet, staticcheck, gosec, etc.
├── go.mod                      # Go module definition
├── go.sum                      # Dependency checksums
├── LICENSE                     # MIT License
└── README.md                   # User-facing documentation with Mermaid diagrams
```

## 4. Interfaces & Dependency Injection

The poller package uses **interface-based dependency injection** to decouple from concrete implementations, enabling testability and flexibility.

### `EeroClient` Interface (`internal/poller/interfaces.go`)
```go
type EeroClient interface {
    GetNetwork(ctx context.Context, url string) (*eero.NetworkDetails, error)
    ListDevices(ctx context.Context, url string) ([]eero.Device, error)
    ListProfiles(ctx context.Context, url string) ([]eero.Profile, error)
}
```
*   Abstracts the Eero API surface.
*   Production implementation: `EeroClientAdapter` (see below).
*   Test implementation: `MockEeroClient` with injectable function fields.

### `MetricWriter` Interface (`internal/poller/interfaces.go`)
```go
type MetricWriter interface {
    WritePoint(point *write.Point)
}
```
*   Matches the `api.WriteAPI` signature from the InfluxDB client.
*   Production implementation: `api.WriteAPI` (non-blocking, async).
*   Test implementation: `MockMetricWriter` with thread-safe point collection and a `writeReady` channel for synchronization.

### `EeroClientAdapter` (`internal/poller/adapter.go`)
*   **Adapter pattern:** Wraps `*eero.Client` to satisfy the `EeroClient` interface.
*   Keeps the concrete `eero-go` dependency at the application boundary (`cmd/eero-stats/main.go`).
*   The `Poller` struct only depends on the `EeroClient` interface, never the concrete client.

## 5. Data Flow & Routing

```
1. Startup
   └── main.go
       ├── config.Load()         → Reads .env + os.Getenv, validates all required vars
       ├── auth.Init(ctx, cfg)   → Restores session OR interactive 2FA via stdin
       │   └── restoreSession()  → Read .eero_session.json → SetSessionCookie()
       │   └── interactiveLogin()→ Login challenge → OTP → Verify → saveSession()
       ├── eeroClient.Account.Get() → Discovers primary network URL
       ├── db.NewInfluxClient()  → Creates InfluxDB client with batching options
       └── poller.NewPoller()    → Initializes via EeroClientAdapter + WriteAPI

2. Polling Loop
   └── Poller.Start(ctx)
       ├── goroutine: runTier("Fast", 3min, safePollFast)
       │   └── pollFast(ctx)
       │       ├── withRetry → GetNetwork → writeNodeTimeSeries()
       │       │                           → writeNetworkHealth()
       │       └── withRetry → ListDevices → writeClientDeviceTimeSeries()
       ├── goroutine: runTier("Medium", 90min, safePollMedium)
       │   └── pollMedium(ctx)
       │       ├── withRetry → GetNetwork → writeNodeMetadata()
       │       ├── withRetry → ListDevices → writeClientMetadata()
       │       └── withRetry → ListProfiles → writeProfileMappings()
       └── goroutine: runTier("Slow", 12hr, safePollSlow)
           └── pollSlow(ctx)
               └── withRetry → GetNetwork → writeISPSpeeds()
                                           → writeNetworkConfig()

3. Write Path
   └── Each writer builds: tags{} + fields{} → influxdb2.NewPoint(measurement, tags, fields, now)
       └── MetricWriter.WritePoint(pt)  → Buffered in memory (async, non-blocking)
           └── InfluxDB client auto-flushes:
               BatchSize  = 100 points
               FlushInterval = 60,000 ms (60 seconds)
           └── Async write errors logged via background goroutine on WriteAPI.Errors() channel

4. Shutdown
   └── SIGINT/SIGTERM → cancel context → WaitGroup.Wait() → influxClientShutdown()
       └── WriteAPI.Flush() → Client.Close() (15-second timeout)
```

## 6. Concurrency & Event Loop Model

*   **Goroutines:** The daemon uses `sync.WaitGroup` to orchestrate multiple polling loops. Each polling tier runs in its own goroutine with an immediate initial poll followed by periodic ticks (`time.Ticker`).
*   **Panic Recovery:** Each polling tier is wrapped in a `safePoll*` function with `defer recover()` to prevent a panic in one tier from crashing the daemon. Panics are logged via `slog.Error`.
*   **Context & Graceful Shutdown:** `context.Context` propagates cancellation. The main function listens for `SIGINT` / `SIGTERM`, deregisters the signal handler (allowing force-kill on second signal), cancels the context, waits for the `WaitGroup`, and performs a graceful 15-second shutdown flush of the InfluxDB client.
*   **Signal Handling:** Second signal after initial SIGTERM/SIGINT will force-kill (signal handler deregistered via `signal.Stop(sigCh)` after first signal).

## 7. Retry Logic

`withRetry()` in `internal/poller/retry.go`:
*   **Max Retries:** 3 attempts.
*   **Backoff Formula:** `2^(attempt+1)` seconds → 2s, 4s delays between attempts.
*   **Context-Aware:** If context is cancelled during backoff, returns `ctx.Err()` immediately.
*   **Error Wrapping:** On exhaustion, returns `fmt.Errorf("after %d attempts, last error: %w", ...)`.
*   **Non-fatal:** Failed API calls in polling tiers log warnings (`slog.Warn`) and continue — they never crash the daemon.

## 8. Environment & Configuration

Loaded from `.env` via `github.com/joho/godotenv`. If no `.env` file exists, environment variables are read directly (e.g., from Docker `env_file`).

### Go Application Config (`internal/config/config.go`)
| Variable | Required | Default | Description |
| :--- | :---: | :---: | :--- |
| `EERO_LOGIN` | **Yes** | — | Email or phone for Eero Owner Account |
| `EERO_SESSION_PATH` | No | `/app/data/.eero_session.json` (Docker) or `data/app/.eero_session.json` (local) | Path to cached session file |
| `INFLUX_URL` | **Yes** | — | InfluxDB URL |
| `INFLUX_TOKEN` | **Yes** | — | InfluxDB API token |
| `INFLUX_ORG` | **Yes** | — | InfluxDB organization |
| `INFLUX_BUCKET` | **Yes** | — | InfluxDB bucket |

> **Note:** `INFLUX_URL`, `INFLUX_TOKEN`, `INFLUX_ORG`, and `INFLUX_BUCKET` have NO Go-level defaults. They are required by `config.Load()` and the daemon will exit with an error if unset. Defaults shown in `.env.example` and `docker-compose.yml` are convenience values for local development only.

### Docker Compose / Infrastructure-Only Variables
| Variable | Default | Description |
| :--- | :---: | :--- |
| `PUID` | `1000` | Container user UID |
| `PGID` | `1000` | Container user GID |
| `GRAFANA_PORT` | `3000` | Grafana exposed port |
| `INFLUX_PORT` | `8086` | InfluxDB exposed port |
| `GF_ADMIN_PASSWORD` | `admin` | Grafana admin password |

## 9. InfluxDB Measurement Schema

All measurements are written to the bucket specified by `INFLUX_BUCKET` (default: `eero`).

### Fast Poll (every 3 minutes)
| Measurement | Tags | Key Fields | Writer Function |
| :--- | :--- | :--- | :--- |
| `eero_client_timeseries` | `mac`, `device_name`, `source_location`, `node_name`, `connection_type`, `frequency`, `frequency_unit` | `connected`, `score_bars`, `score`, `signal` (dBm, parsed), `signal_avg`, `rx_rate_bps`, `tx_rate_bps`, `rx_channel_width`, `rx_mcs`, `paused`, `is_guest`, `blacklisted`, `channel` | `writeClientDeviceTimeSeries()` |
| `eero_node_timeseries` | `serial`, `location`, `model`, `node_name` | `connected_clients_count`, `mesh_quality_bars`, `heartbeat_ok`, `status`, `state`, `using_wan`, `power_source`, `connection_type`, `led_on` | `writeNodeTimeSeries()` |
| `eero_network_health` | `network_name` | `isp_up`, `internet_status`, `eero_network_status` | `writeNetworkHealth()` |

### Medium Poll (every 90 minutes)
| Measurement | Tags | Key Fields | Writer Function |
| :--- | :--- | :--- | :--- |
| `eero_node_metadata` | `serial`, `node_name` | `ip_address`, `mac_address`, `os_version`, `model_number`, `update_available`, `wired`, `gateway`, `is_primary_node`, `led_on`, `last_heartbeat`, `joined`, `ethernet_addresses`, `wifi_bssids`, `bands` | `writeNodeMetadata()` |
| `eero_client_metadata` | `mac`, `device_name` | `device_type`, `ipv4`, `ip`, `ipv6`, `is_proxied_node`, `is_private_mac`, `is_guest`, `blacklisted`, `paused`, `auth`, `ssid`, `subnet_kind`, `vlan_name`, `vlan_id`, `manufacturer`, `first_active`, `last_active` | `writeClientMetadata()` |
| `eero_profile_mappings` | `profile_name` | `devices` (comma-separated MACs), `paused`, `block_apps`, `safe_search_active` | `writeProfileMappings()` |

### Slow Poll (every 12 hours)
| Measurement | Tags | Key Fields | Writer Function |
| :--- | :--- | :--- | :--- |
| `eero_isp_speed` | `network_name` | `speed_down_mbps`, `speed_up_mbps` | `writeISPSpeeds()` |
| `eero_network_config` | `network_name` | `premium_status`, `premium_tier`, `dns_policies_enabled`, `ad_block_enabled`, `block_malware_enabled`, `dhcp_mode`, `dhcp_router`, `dns_mode`, `dns_caching`, `dns_parent_ips`, `geoip_*` (7 fields), `wan_type`, `wireless_mode`, `mlo_mode`, `band_steering`, `wpa3_enabled`, `upnp_enabled`, `ipv6_upstream`, `thread_enabled`, `sqm_enabled`, `double_nat`, `public_ip`, `guest_network_enabled`, `guest_network_name`, `firmware_has_update`, `firmware_target`, `firmware_update_req` | `writeNetworkConfig()` |

## 10. Docker & Infrastructure Topology

The `docker-compose.yml` defines the full stack:

### `eero-stats` (Go Daemon)
*   **Build:** Local Dockerfile (multi-stage: `golang:1.22-alpine` builder → `alpine:3.21` runtime).
*   **Runtime dependencies:** `ca-certificates`, `tzdata` only.
*   **User:** Controlled via `${PUID}:${PGID}` (default `1000:1000`).
*   **2FA Support:** `tty: true` + `stdin_open: true` for `docker attach` during initial auth.
*   **Volume:** `./data/app:/app/data` for session file persistence.
*   **Depends on:** `influxdb` (with `condition: service_healthy`).

### `influxdb` (InfluxDB 2.7)
*   **Image:** `influxdb:2.7`.
*   **Auto-bootstrap:** `DOCKER_INFLUXDB_INIT_MODE: setup` with admin user, org, bucket, and token.
*   **Health check:** `influx ping` every 10s, 5s timeout, 5 retries.
*   **Volume:** `./data/influxdb:/var/lib/influxdb2`.

### `grafana` (Grafana OSS latest)
*   **User:** `${PUID}:${PGID}` (must match InfluxDB for shared volume permissions).
*   **Auto-provisioned:** Datasource YAML + dashboard provider YAML → dashboard JSON loaded on startup.
*   **Datasource UID:** `P951FEA4DE68E13C5` (must match `scripts/build_dashboard.py` `DS` constant).
*   **Volumes:**
    *   `./data/grafana:/var/lib/grafana`
    *   `./grafana/provisioning/datasources` (read-only)
    *   `./grafana/provisioning/dashboards/dashboards.yml` (read-only)
    *   `./grafana/dashboards` (read-only, contains generated `eero.json`)

## 11. Build Metadata & Version Injection

Build metadata is injected at compile-time via `ldflags`:
```
-X github.com/arvarik/eero-stats/internal/version.Version=$(VERSION)
-X github.com/arvarik/eero-stats/internal/version.Commit=$(COMMIT)
-X github.com/arvarik/eero-stats/internal/version.BuildDate=$(BUILD_DATE)
```
*   `VERSION`: `git describe --tags --always --dirty` (falls back to `dev`).
*   `COMMIT`: `git rev-parse --short HEAD` (falls back to `none`).
*   `BUILD_DATE`: UTC ISO-8601 timestamp.
*   Logged on startup: `slog.Info("eero-stats daemon starting up", ...)`.

## 12. CI/CD Pipeline

### GitHub Actions (`ci.yml`)
**Triggers:** Push to `main`, PRs targeting `main`.

**Steps (ubuntu-latest, Go 1.22):**
1.  `actions/checkout@v4`
2.  `actions/setup-go@v5` (Go 1.22)
3.  `go mod tidy`
4.  `go build ./cmd/... ./internal/...`
5.  `go vet ./cmd/... ./internal/...`
6.  `go test -race -count=1 ./...`
7.  `golangci/golangci-lint-action@v3` (latest, 5min timeout, `continue-on-error: true`)

### Pre-Commit Hook (`.githooks/pre-commit`)
*   Runs `go vet ./...` and `make lint` before every commit.
*   Bypass: `git commit --no-verify` (not recommended).
*   Setup: `make setup` (sets `core.hooksPath` to `.githooks/`).

## 13. Grafana Dashboard Architecture

The dashboard is **generated programmatically** by `scripts/build_dashboard.py` (755 lines):
*   Output: `grafana/dashboards/eero.json` (~55 KB).
*   **7 dashboard sections:** Executive Summary, ISP & Connectivity, Mesh Node Health, Node Deep-Dive, Client Device Health, Device Deep-Dive, Alerts & Anomalies.
*   **Panel types:** stat, gauge, timeseries, table, state-timeline, piechart, barchart, bargauge.
*   **Template variables:** `$Node` (query), `$Device` (query), `$Frequency` (custom: 2.4, 5, 6, wired).
*   **Datasource UID:** Hardcoded `P951FEA4DE68E13C5` — must match `grafana/provisioning/datasources/datasource.yml`.
*   **Regenerate:** `make dashboard` (runs `python3 scripts/build_dashboard.py`).

## 14. Safety Invariants & Architectural Rules

*   **NEVER** block the main polling loop. Each tier MUST run in its own goroutine.
*   **ALWAYS** use panic-safe goroutine wrappers (`safePoll*`) for long-running background tasks.
*   **NEVER** write un-batched metrics directly to disk. Aggressive memory batching (`BatchSize=100`, `FlushInterval=60s`) is required to save NVMe/flash wear on NAS systems.
*   **ALWAYS** gracefully flush buffered writes to InfluxDB on `SIGTERM` (15-second timeout).
*   **NEVER** use the concrete `*eero.Client` directly in the `poller` package. Always go through the `EeroClient` interface.
*   **ALWAYS** use `withRetry()` for API calls. Polling failures MUST be logged and recovered, not fatal.
*   **NEVER** mutate the Grafana dashboard JSON directly. Always regenerate via `make dashboard`.
*   **ALWAYS** keep the datasource UID in `build_dashboard.py` in sync with `datasource.yml`.
