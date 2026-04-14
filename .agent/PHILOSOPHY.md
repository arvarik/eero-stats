# Product Philosophy & Vision

## 1. Core Purpose

`eero-stats` is a daemon that extracts real-time metrics from the Eero Mesh Network cloud API and writes them to InfluxDB v2 for visualization in Grafana. It fills the gap of advanced network telemetry and historical data analysis that the consumer Eero app lacks — providing deep visibility into signal quality, device roaming, node backhaul health, ISP speed trends, and network configuration drift over time.

## 2. Target Persona

*   **Primary:** Homelab enthusiasts and NAS operators (TrueNAS SCALE, Unraid, Proxmox) who want deep visibility into their home network performance without relying solely on Eero's cloud dashboard.
*   **Secondary:** Network administrators managing small-to-medium deployments who need historical trend data, anomaly detection (e.g., double NAT, firmware staleness, disconnected devices), and per-device signal analysis.
*   **Deployment Environment:** Containerized (Docker Compose) on always-on NAS hardware with NVMe or flash storage. The daemon is designed to run 24/7 with minimal resource consumption and zero interactive maintenance after initial 2FA setup.

## 3. Core Beliefs & Principles

1.  **Minimal Hardware Wear:** NAS systems often run on NVMe or USB flash drives with limited write endurance. Aggressive memory batching (`BatchSize=100`, `FlushInterval=60s`) is non-negotiable to prevent premature hardware failure from continuous metric writes. No unbatched writes are ever permitted.

2.  **API Etiquette:** The Eero cloud API is undocumented and rate-limited. Tiered polling (Fast: 3min / Medium: 90min / Slow: 12hr) balances data freshness with API safety. All API calls are wrapped in exponential backoff retry logic (3 attempts, 2^n seconds). Polling failures are logged and recovered, never fatal.

3.  **Resilience:** The daemon must survive network partitions, API rate limits, and unexpected panics without crashing. Each polling tier runs in its own goroutine with panic recovery. Context-aware shutdown ensures buffered data is flushed to disk before the process exits.

4.  **Zero-Configuration Dashboards:** The Grafana dashboard is fully provisioned and ready to use immediately upon container start, requiring no manual metric mapping, datasource configuration, or panel creation by the user. Dashboard-as-code (Python generator) ensures reproducibility.

5.  **Testability-First Design:** All external dependencies (Eero API, InfluxDB) are abstracted behind Go interfaces (`EeroClient`, `MetricWriter`). The concrete `eero.Client` is adapted at the application boundary via `EeroClientAdapter`. This enables comprehensive unit testing with mocks — no network or database access required in CI.

6.  **Minimal Runtime Footprint:** The production Docker image is built on `alpine:3.21` with only `ca-certificates` and `tzdata` installed. No runtime dependencies on Go toolchain, build tools, or unnecessary system packages. The binary is statically compiled with `CGO_ENABLED=0`.

7.  **Structured Observability:** All operational logging uses `log/slog` with leveled, structured output (key-value pairs). No `fmt.Println`, `log.Printf`, or unstructured log statements. This ensures logs are machine-parseable and filterable in container orchestration platforms.

8.  **Dashboard-as-Code:** The Grafana dashboard JSON is never hand-edited. It is generated programmatically by `scripts/build_dashboard.py`, ensuring consistency, version control, and reproducibility. The Python script is the single source of truth for all dashboard panels, queries, and layout.

## 4. Design Decisions & Rationale

| Decision | Rationale |
| :--- | :--- |
| Tiered polling over single interval | Different data types change at different rates. Signal strength changes in minutes; firmware versions change in weeks. Tiered polling minimizes API calls while keeping high-frequency data fresh. |
| Interface-based DI over global state | Enables mock injection for testing without network access. Keeps `eero-go` dependency at the boundary. |
| Adapter pattern for `eero.Client` | The `eero-go` client has a service-based API (`client.Network.Get()`, `client.Device.List()`). The adapter flattens this into the `EeroClient` interface for cleaner poller code. |
| Non-blocking InfluxDB write API | Metric writes must never block the polling loop. The async write API with background error channel keeps the poller responsive. |
| Python dashboard generator over JSON editing | A 55 KB JSON dashboard is unmaintainable by hand. The Python script uses helper functions (`stat()`, `timeseries()`, `table()`) to compose panels declaratively, making changes safe and reviewable. |
| Multi-stage Docker build | Produces a ~15 MB final image (Alpine + static binary) instead of ~1 GB (full Go toolchain). Critical for NAS deployments with limited storage. |
| `context.Context` everywhere | Enables clean cancellation propagation from SIGTERM through all polling goroutines and API calls. The retry helper also respects context cancellation during backoff waits. |
| Session cache on disk | Avoids re-authentication on every container restart. The Eero session token is long-lived (~30 days). Persisted via Docker volume mount. |

## 5. What This Is NOT (Anti-Goals)

*   It is **NOT** a tool to modify network settings or control Eero devices. This is **read-only telemetry** — no write operations to the Eero API.
*   It is **NOT** designed for direct, real-time streaming. Metrics are batched and delayed slightly (up to 60 seconds) to optimize writes.
*   It is **NOT** a Prometheus exporter. Metrics are written to InfluxDB using the native line protocol client, not exposed as a scrape target.
*   It is **NOT** a real-time alerting system. While the Grafana dashboard includes an "Alerts & Anomalies" section, it relies on Grafana's built-in alerting, not application-level alert logic.
*   It is **NOT** designed for multi-network support. The daemon discovers and monitors the **first** network on the authenticated Eero account.
