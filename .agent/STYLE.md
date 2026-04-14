# Coding Style & Conventions

## 1. Project Layout & File Organization

### Go Package Structure
*   **Standard Go layout:** `cmd/` for entry points, `internal/` for private packages.
*   `cmd/eero-stats/main.go` — Daemon bootstrap, signal handling, graceful shutdown.
*   `internal/auth/` — Eero API authentication (session cache + interactive 2FA).
*   `internal/config/` — Environment variable loading and startup validation.
*   `internal/db/` — InfluxDB client wrapper with NVMe-optimized batching.
*   `internal/poller/` — Tiered polling engine, metric writers, retry logic, interfaces.
*   `internal/version/` — Build-time metadata (`Version`, `Commit`, `BuildDate`).

### File Naming Conventions
*   Go source files: `snake_case.go` (e.g., `config.go`, `writers.go`, `retry.go`).
*   Test files: `*_test.go` suffix, same package as source (white-box testing).
*   Interface definitions: Collected in `interfaces.go` in the consuming package.
*   Adapter implementations: Named `adapter.go` in the consuming package.

### Identifier Naming
*   Standard Go naming: `camelCase` for unexported, `PascalCase` for exported.
*   Acronyms are uppercased: `URL` not `Url`, `MAC` not `Mac`, `IP` not `Ip`.
*   Constructor functions: `New<Type>()` (e.g., `NewPoller()`, `NewInfluxClient()`, `NewEeroClientAdapter()`).
*   Interface names: Use the capability name, not the `I` prefix (e.g., `EeroClient` not `IEeroClient`, `MetricWriter` not `IMetricWriter`).

## 2. Interface & Dependency Injection Patterns

### Interface Definition
*   Interfaces are defined in the **consuming** package, not the implementing package. `EeroClient` and `MetricWriter` live in `internal/poller/interfaces.go` because `poller` is the consumer.
*   Interfaces should be as small as possible — only the methods the consumer actually needs.

### Adapter Pattern
*   When adapting a third-party client to an interface, create an `Adapter` struct in the consuming package:
```go
// adapter.go
type EeroClientAdapter struct {
    client *eero.Client
}

func NewEeroClientAdapter(client *eero.Client) *EeroClientAdapter {
    return &EeroClientAdapter{client: client}
}

func (a *EeroClientAdapter) GetNetwork(ctx context.Context, url string) (*eero.NetworkDetails, error) {
    return a.client.Network.Get(ctx, url)
}
```
*   The concrete dependency (`*eero.Client`) is injected at the application boundary in `main.go`, never inside `internal/poller/`.

### Constructor Injection
*   Structs receive dependencies via constructors:
```go
func NewPoller(client EeroClient, influx MetricWriter, networkURL string) *Poller
```
*   Never use global variables or `init()` functions for dependency wiring.

## 3. Error Handling

### Pattern
*   Standard Go `if err != nil` pattern.
*   Errors are wrapped with context using `fmt.Errorf("doing X: %w", err)`.
*   Sentinel errors: Only `errors.New()` for simple cases (e.g., `"EERO_LOGIN environment variable is required"`).

### Logging vs. Fatal
*   **Startup failures** (config, auth, account discovery) → `slog.Error()` + `os.Exit(1)`.
*   **Runtime failures** (polling, API calls) → `slog.Warn()` + continue. Polling failures are **never fatal**.
*   **Async write errors** (InfluxDB) → `slog.Error()` in background goroutine on `WriteAPI.Errors()` channel.

## 4. Logging

*   **ALWAYS** use `log/slog` for all operational logging.
*   Use structured key-value pairs: `slog.Info("message", "key1", val1, "key2", val2)`.
*   Log levels:
    *   `slog.Info` — Normal operational events (startup, poll start, shutdown).
    *   `slog.Warn` — Recoverable issues (failed API calls, expired sessions).
    *   `slog.Error` — Critical issues (panics, InfluxDB write failures, startup failures).
*   Logger configuration: `slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})`.

## 5. Metric Writer Pattern

All metric writer functions in `internal/poller/writers.go` follow this consistent pattern:

```go
func (p *Poller) write<MetricName>(data *SomeType) {
    now := time.Now()
    for i := range data.Items {
        item := &data.Items[i]

        // 1. Build tags (indexed dimensions, used for grouping/filtering)
        tags := map[string]string{
            "primary_key": item.Key,
            "dimension":   item.Dimension,
        }

        // 2. Build fields (metric values, not indexed)
        fields := map[string]interface{}{
            "metric_name": item.Value,
        }

        // 3. Handle optional/nullable fields
        if item.OptionalField != nil {
            fields["optional"] = *item.OptionalField
        }

        // 4. Write the point
        pt := influxdb2.NewPoint("measurement_name", tags, fields, now)
        p.influx.WritePoint(pt)
    }
}
```

**Rules:**
*   Use `time.Now()` once at the top of the writer, reuse for all points in the same collection.
*   Iterate with `for i := range` and take address with `&items[i]` to avoid copying.
*   Tags are for dimensions you'll filter/group by in Flux queries.
*   Fields are for values that change over time.
*   Nullable/optional fields from the Eero API MUST be nil-checked before dereferencing.
*   String-to-numeric parsing (e.g., signal strength `"NN dBm"` → int) should be done inline with error handling.

## 6. Polling Tier Organization

When adding a new API call:
1. Add the method to the `EeroClient` interface in `interfaces.go`.
2. Implement it on `EeroClientAdapter` in `adapter.go`.
3. Create the writer function in `writers.go` following the writer pattern.
4. Call the writer from the appropriate tier function in `poller.go`:
   *   **pollFast** — Data that changes frequently (device connectivity, signal strength).
   *   **pollMedium** — Data that changes slowly (metadata, profiles).
   *   **pollSlow** — Data that changes rarely (ISP speeds, network config).
5. Wrap API calls with `p.withRetry(ctx, func() error { ... })`.
6. Add the mock implementation to `poller_test.go`.

## 7. Test Conventions

### File Structure
*   Test files live alongside source: `config.go` → `config_test.go`.
*   Test files use the **same package** as the source (white-box testing).

### Table-Driven Tests
*   All tests use the table-driven pattern:
```go
tests := []struct {
    name    string
    input   SomeType
    want    string
    wantErr string
}{
    {name: "descriptive case name", ...},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got := functionUnderTest(tt.input)
        if got != tt.want {
            t.Errorf("functionUnderTest() = %q, want %q", got, tt.want)
        }
    })
}
```

### Mock Conventions
*   Mock structs are defined in `_test.go` files, not in separate mock packages.
*   Mock naming: `Mock<InterfaceName>` (e.g., `MockEeroClient`, `MockMetricWriter`).
*   Mock functions use injectable fields: `GetNetworkFunc func(...)` with nil-check fallback to sensible defaults.
*   Thread-safety: Mocks that track calls or collect data use `sync.Mutex`.
*   Synchronization: Use channel-based signaling (`writeReady chan struct{}` with `sync.Once`) instead of `time.Sleep` for waiting on async operations.

### Assertions
*   Use `t.Fatalf` for conditions that make continuing the test meaningless.
*   Use `t.Errorf` for non-blocking assertions where multiple can fail independently.
*   No third-party assertion libraries (e.g., testify). Standard library only.

## 8. Makefile & Build Conventions

### Targets
| Target | Description |
| :--- | :--- |
| `make all` | Runs `tidy`, `lint`, `test`, `build` in sequence |
| `make tidy` | `go mod tidy` + `go fmt ./...` |
| `make lint` | `golangci-lint run ./...` |
| `make test` | `go test -race -count=1 ./...` |
| `make build` | Compiles to `bin/eero-stats` with ldflags for version metadata |
| `make version` | Prints current Version, Commit, Build Date |
| `make dashboard` | Regenerates Grafana dashboard JSON via Python script |
| `make docker-up` | Creates data dirs, fixes ownership, runs `docker compose up -d` |
| `make docker-down` | Runs `docker compose down` |
| `make setup` | Sets git hooks path to `.githooks/` |
| `make clean` | Removes `bin/` directory |

### Conventions
*   All targets are declared `.PHONY`.
*   Build output goes to `bin/` (gitignored).
*   `make all` is the default target and runs the full CI-equivalent pipeline locally.

## 9. Docker Conventions

### Dockerfile
*   **Multi-stage build:** `golang:1.22-alpine` (builder) → `alpine:3.21` (runtime).
*   **Static binary:** `CGO_ENABLED=0 GOOS=linux` for Alpine compatibility.
*   **Strip symbols:** `-ldflags "-s -w"` to reduce binary size.
*   **No hardcoded USER:** User is controlled via `docker-compose.yml` `user:` directive for flexibility.
*   **Minimal runtime:** Only `ca-certificates` (HTTPS to Eero API) and `tzdata` (timezone support).

### Docker Compose
*   All containers use `restart: unless-stopped`.
*   Data persistence via bind mounts to `./data/<service>/`.
*   `eero-stats` depends on `influxdb` with `condition: service_healthy` to avoid startup race.
*   Environment variables sourced from `.env` via `env_file:`.

## 10. Python Conventions (Dashboard Script)

*   `scripts/build_dashboard.py` uses **only Python standard library** (`json`). No pip dependencies.
*   Panel types are generated via helper functions: `stat()`, `gauge()`, `timeseries()`, `table()`, `state_timeline()`, `piechart()`, `bar_chart()`, `bargauge()`.
*   Flux queries are defined as module-level string constants prefixed with `Q_` (e.g., `Q_ISP_STATUS`, `Q_MESH_QUALITY`).
*   The datasource UID is a module-level constant `DS = {"type": "influxdb", "uid": "P951FEA4DE68E13C5"}`.
*   Panel IDs are auto-incremented via `next_id()`.
*   Output path is hardcoded: `grafana/dashboards/eero.json`.

## 11. Linting & Code Quality

### golangci-lint Configuration (`.golangci.yml`)
*   **Timeout:** 5 minutes.
*   **Enabled linters:**
    *   Default: `errcheck`, `govet`, `staticcheck`, `unused`, `ineffassign`
    *   Extra: `gofmt`, `goimports`, `misspell`, `gosec`, `prealloc`, `gocritic`
*   **Settings:**
    *   `govet`: All checks enabled.
    *   `gosec`: `G104` excluded (duplicates `errcheck`).
    *   `gocritic`: `diagnostic` and `performance` tags enabled.
*   **Exclusions:** `gosec` is relaxed in `_test.go` files.

### Pre-Commit Hook (`.githooks/pre-commit`)
*   Runs `go vet ./...` followed by `make lint`.
*   Fails the commit if either check fails.
*   Bypass: `git commit --no-verify`.

## 12. InfluxDB Naming Conventions

### Measurements
*   Prefix: `eero_` for all measurements.
*   Format: `eero_<entity>_<data_type>` (e.g., `eero_client_timeseries`, `eero_node_metadata`, `eero_network_config`).

### Tags (Indexed Dimensions)
*   `snake_case` naming.
*   Used for values you'll filter or group by: `mac`, `device_name`, `serial`, `node_name`, `network_name`, `profile_name`.

### Fields (Metric Values)
*   `snake_case` naming.
*   Boolean fields: `connected`, `paused`, `is_guest`, `heartbeat_ok`, `wpa3_enabled`.
*   Numeric fields: `score_bars`, `signal` (parsed from string), `rx_rate_bps`, `speed_down_mbps`.
*   String fields: `status`, `state`, `os_version`, `dns_mode`.

## 13. Anti-Patterns (FORBIDDEN)

*   **NEVER** use `fmt.Println`, `fmt.Printf`, or `log.Printf` for operational logging. ALWAYS use `log/slog`.
*   **NEVER** bypass the batching InfluxDB client when writing metrics. All writes go through `MetricWriter.WritePoint()`.
*   **NEVER** swallow panics in background goroutines without logging them via `slog.Error`.
*   **NEVER** write blocking code in the main `context.Done()` wait loop.
*   **NEVER** use `time.Sleep` in tests for synchronization. Use channels or `sync` primitives.
*   **NEVER** import the concrete `eero-go` client in `internal/poller/` source files (only in `adapter.go` and test files for type references).
*   **NEVER** hand-edit `grafana/dashboards/eero.json`. Always regenerate via `make dashboard`.
*   **NEVER** use global mutable state or `init()` for dependency wiring.
*   **NEVER** use third-party test assertion libraries. Standard library `testing` only.
