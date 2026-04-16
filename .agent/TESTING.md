# Testing & Verification Strategy

## 1. Local Development Setup

### Prerequisites
*   **Go:** 1.22+ (matches `go.mod` and CI)
*   **Docker & Docker Compose:** Required for full-stack integration
*   **GNU Make:** Build automation
*   **Python 3:** For Grafana dashboard generation (`scripts/build_dashboard.py`, stdlib only)
*   **golangci-lint:** For linting (`make lint`). [Install instructions](https://golangci-lint.run/usage/install/)

### First-Time Setup
```bash
make setup    # Configures git hooks for pre-commit linting (go vet + golangci-lint)
cp .env.example .env
# Edit .env with your Eero login and any custom ports
```

### Starting the Full Stack
```bash
make docker-up    # Creates data dirs, fixes ownership, starts daemon + InfluxDB + Grafana
```

### Initial Authentication (First Boot Only)
```bash
docker attach eero-stats
# Enter the OTP verification code sent to your email/phone
# Detach with CTRL-P, CTRL-Q (do NOT use CTRL-C — it will kill the daemon)
```

### Accessing the Dashboard
Open Grafana at `http://localhost:<GRAFANA_PORT>` (default `3000`). Login: `admin` / your `GF_ADMIN_PASSWORD`.

## 2. Test Execution Commands

| Command | Description | What It Runs |
| :--- | :--- | :--- |
| `make all` | Full CI-equivalent pipeline | `tidy` → `lint` → `test` → `build` |
| `make tidy` | Format and tidy dependencies | `go mod tidy` + `go fmt ./...` |
| `make lint` | Static analysis | `golangci-lint run ./...` (5min timeout) |
| `make test` | Unit tests with race detector | `go test -race -count=1 ./...` |
| `make build` | Compile binary | `go build` with ldflags → `bin/eero-stats` |
| `make version` | Print build metadata | Displays Version, Commit, Build Date |
| `make dashboard` | Regenerate Grafana dashboard | `python3 scripts/build_dashboard.py` → `grafana/dashboards/eero.json` |
| `make clean` | Remove build artifacts | `rm -rf bin/` |

### Race Detector Rationale
The `-race` flag is mandatory because the daemon is goroutine-heavy (3 concurrent polling tiers + async InfluxDB writes + signal handler goroutine). Race conditions would be catastrophic in a long-running daemon.

## 3. Test Inventory

### `internal/config/config_test.go`
| Test Function | Type | Description |
| :--- | :--- | :--- |
| `TestLoad` | Table-driven | Validates config loading: missing `EERO_LOGIN`, missing `INFLUX_URL`, missing `INFLUX_TOKEN`, missing `INFLUX_ORG`, missing `INFLUX_BUCKET`, and all-present success case. Uses env var save/restore for isolation. |

### `internal/poller/retry_test.go`
| Test Function | Type | Description |
| :--- | :--- | :--- |
| `TestWithRetry_SuccessOnFirstAttempt` | Unit | Verifies single call on success. |
| `TestWithRetry_SuccessAfterRetry` | Unit | Verifies recovery after 2 failures (3rd attempt succeeds). |
| `TestWithRetry_AllRetriesExhausted` | Unit | Verifies error wrapping after 3 failures. Checks `errors.Is` on the wrapped error. |
| `TestWithRetry_ContextCancellation` | Unit | Verifies that cancelling context during backoff returns `context.Canceled`. |

### `internal/poller/writers_test.go`
| Test Function | Type | Description |
| :--- | :--- | :--- |
| `TestResolveDeviceName` | Table-driven | Tests the device name resolution chain: nickname → hostname → MAC address. Covers nil, empty string, and present-value cases for both nickname and hostname. |

### `internal/poller/poller_test.go`
| Test Function | Type | Description |
| :--- | :--- | :--- |
| `TestPoller_Start` | Integration | Full poller lifecycle: starts with mocks, waits for at least one metric write (channel-based sync, not `time.Sleep`), cancels context, verifies clean shutdown, asserts all 3 API methods were called and points were written. |

### Untested Packages
| Package | Reason |
| :--- | :--- |
| `internal/auth` | Requires stdin interaction for 2FA and a real Eero API session. Would need stdin mocking to test. |
| `internal/db` | Thin wrapper around InfluxDB client. Write path tested via `MockMetricWriter` in poller tests. |
| `internal/version` | Build-time constants only. No logic to test. |

## 4. Mock Patterns

### `MockEeroClient` (`poller_test.go`)
```go
type MockEeroClient struct {
    GetNetworkFunc   func(ctx context.Context, url string) (*eero.NetworkDetails, error)
    ListDevicesFunc  func(ctx context.Context, url string) ([]eero.Device, error)
    ListProfilesFunc func(ctx context.Context, url string) ([]eero.Profile, error)
}
```
*   Each interface method delegates to the corresponding `Func` field.
*   If the `Func` field is `nil`, returns a sensible zero-value default (empty struct, empty slice).
*   Test-specific behavior is injected per-test by setting the function fields.
*   **Thread-safety:** Call tracking uses `sync.Mutex` around shared boolean flags.

### `MockMetricWriter` (`poller_test.go`)
```go
type MockMetricWriter struct {
    mu         sync.Mutex
    points     []*write.Point
    writeReady chan struct{}
    once       sync.Once
}
```
*   Collects all written points in a thread-safe slice.
*   `writeReady` channel + `sync.Once` signals when the first metric write occurs.
*   **Used for synchronization:** Test code does `<-mockWriter.writeReady` instead of `time.Sleep` to avoid CI flakiness.
*   `pointCount()` method provides thread-safe read access.

## 5. CI Pipeline Integration

### GitHub Actions (`ci.yml`)
**Triggers:** Push to `main`, PRs targeting `main`.

**Steps:**
1.  Checkout (`actions/checkout@v4`)
2.  Set up Go 1.22 (`actions/setup-go@v5`)
3.  `go mod tidy`
4.  `go build ./cmd/... ./internal/...`
5.  `go vet ./cmd/... ./internal/...`
6.  `go test -race -count=1 ./...`
7.  `golangci-lint` (latest, 5min timeout, `continue-on-error: true`)

> **Note:** Lint failures currently do NOT block PR merges (`continue-on-error: true`). This is tracked as tech debt (`TD-08` in STATUS.md).

### Pre-Commit Hook
*   Runs `go vet ./...` + `make lint` before every commit.
*   Setup: `make setup` (runs `git config core.hooksPath .githooks`).
*   Bypass: `git commit --no-verify`.

## 6. golangci-lint Configuration

Configured in `.golangci.yml`:

**Enabled Linters:**
| Linter | Category |
| :--- | :--- |
| `errcheck` | Error handling |
| `govet` | Correctness (all checks enabled) |
| `staticcheck` | Static analysis |
| `unused` | Dead code |
| `ineffassign` | Unused assignments |
| `gofmt` | Formatting |
| `goimports` | Import ordering |
| `misspell` | Typos |
| `gosec` | Security (G104 excluded, relaxed in tests) |
| `prealloc` | Performance (slice preallocation) |
| `gocritic` | Diagnostic + performance tags |

## 7. Execution Evidence Rules

Before marking a PR or feature branch as complete, the agent MUST:
1.  Run `make test` and paste the passing output.
2.  Run `make lint` and paste the passing output.
3.  Run `make tidy` and verify no uncommitted changes result.
4.  If dashboard changes were made: run `make dashboard` and commit the regenerated JSON.

---

## Backend Route Coverage Matrix

_Populated by the SDET during the Trap phase. One row per API endpoint. All cells must show PASS with execution evidence or FAIL with reproduction steps._

| Endpoint | Method | 200 OK | 400 Bad Req | 401/403 Auth | 404 Not Found | Idempotent | Edge Cases |
|----------|--------|--------|-------------|--------------|---------------|------------|------------|

_N/A — eero-stats is a polling daemon, not an HTTP server. No API routes are registered. The Go backend polls the Eero cloud API on a timer and writes metrics to InfluxDB. There are no `http.HandleFunc`, chi, mux, or other HTTP route registrations in the codebase._

---

## Frontend Component State Matrix

_Populated by the SDET during the Trap phase. Every interactive component must be tested across all visual states._

| Component | Empty | Loading | Success | Error | Partial |
|-----------|-------|---------|---------|-------|---------|
| Executive Summary (ISP Status, Mesh Status, Double NAT, Firmware, Speed Gauges) | | | | | |
| ISP & Connectivity (Location & ISP, Security & Wireless, DNS/DHCP & Services) | | | | | |
| Eero Node Telemetry (Connected Clients, Node Uptime, Mesh Quality, HW Details) | | | | | |
| Node Deep-Dive (Clients History, Mesh Quality History, Connected Devices, Power) | | | | | |
| Client Device Health (Band Distribution, Band Steering, Peak Hours, Signal Strength) | | | | | |
| Device Deep-Dive (AP Roaming Events, Device Signal Strength, Device Metadata) | | | | | |
| Alerts & Anomalies (Disconnected Devices, Paused Devices, Blocked Devices) | | | | | |

_Note: The frontend is a Grafana dashboard generated by `scripts/build_dashboard.py`. Components above correspond to the 7 dashboard sections, each containing stat, gauge, timeseries, table, state-timeline, piechart, barchart, and bargauge panels._

---

## ML / AI Evaluation Thresholds

N/A — ML/AI topology is not active for this project.

## 8. Regression Scenarios

| ID | Feature | Description | Verification |
| :--- | :--- | :--- | :--- |
| `REG-01` | Authentication | Session token restore from `.eero_session.json` on daemon restart | Stop/start container, verify no 2FA prompt if session valid |
| `REG-02` | Authentication | Re-authenticate when session token expires (~30 days) | Delete session file, restart, verify 2FA flow works |
| `REG-03` | Graceful Shutdown | Buffered metrics flushed to InfluxDB on SIGTERM | Send `docker kill --signal=SIGTERM`, check logs for flush confirmation |
| `REG-04` | Panic Recovery | Panic in one polling tier does not crash daemon | Verify via unit test (`TestPoller_Start`) or inject panic in mock |
| `REG-05` | Retry Logic | API failures recover after transient errors | Verify via unit tests (`TestWithRetry_SuccessAfterRetry`) |
| `REG-06` | Context Cancel | Retry aborts cleanly on shutdown signal | Verify via unit test (`TestWithRetry_ContextCancellation`) |
| `REG-07` | Config Validation | Missing required env vars produce clear error messages | Verify via unit test (`TestLoad`) |
| `REG-08` | Device Name Resolution | Nickname → hostname → MAC fallback chain | Verify via unit test (`TestResolveDeviceName`) |
| `REG-09` | Dashboard Generation | `make dashboard` produces valid JSON | Run `make dashboard`, verify Grafana loads without errors |
| `REG-10` | Docker Compose | Full stack starts cleanly with default `.env.example` values | `cp .env.example .env && make docker-up`, verify all 3 containers healthy |

## 9. Adding New Tests

### Unit Test for a New Writer
1.  Add test cases to `writers_test.go` following the table-driven pattern.
2.  Use `MockMetricWriter` to capture written points.
3.  Assert on `pointCount()` and inspect point tags/fields if needed.

### Unit Test for a New Retry Scenario
1.  Add test function to `retry_test.go`.
2.  Use `newTestPoller()` — the retry helper only needs the `Poller` struct, no client/influx.
3.  Inject behavior via the `op func() error` closure.

### Integration Test for New Polling Behavior
1.  Add or modify test in `poller_test.go`.
2.  Configure `MockEeroClient` function fields with test-specific return data.
3.  Use `MockMetricWriter.writeReady` channel for synchronization (never `time.Sleep`).
4.  Always test both the happy path and context cancellation.

## 10. Acceptance Criteria Template

When implementing new features, copy this template:

### Feature: [Feature Name]
| Scenario ID | Description | Preconditions | Action | Expected Result | Evidence |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `SCN-01` | | | | | |
| `SCN-02` | | | | | |
