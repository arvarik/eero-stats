# Coding Style & Conventions

## 1. Structural Patterns
### File & Module Organization
*   **Project Layout:** Standard Go layout (`cmd/`, `internal/`).
    *   `cmd/eero-stats/`: Main application entry point.
    *   `internal/auth/`: Eero API authentication logic.
    *   `internal/config/`: Env var loading and validation.
    *   `internal/db/`: InfluxDB client wrapper.
    *   `internal/poller/`: Tiered polling engine and metric writers.
*   **Naming Conventions:** Standard Go naming conventions (`camelCase` for private, `PascalCase` for public). Files are `snake_case` (e.g., `main.go`, `writers.go`).

### Error Handling
*   **Pattern:** Standard Go `if err != nil`.
*   **Centralized Logging:** Errors in background polls are logged via `slog.Error` or `slog.Warn`, rather than causing a fatal crash. Retries with exponential backoff are used where appropriate.

## 2. API & Component Design
### Go Practices
*   **Dependency Injection:** Interfaces and structs are initialized explicitly (e.g., `NewPoller(client EeroClient, influx MetricWriter, networkURL string)`).
*   **Logging:** Use standard library `log/slog` for structured, leveled logging (e.g., `slog.Info`, `slog.Error`). Avoid `fmt.Printf` for operational logs.

## 3. Visual Identity & UI/UX Design
*   **N/A:** Backend daemon project. Dashboard visuals are handled by Grafana provisioning JSONs.

## 4. Anti-Patterns (FORBIDDEN)
*   **NEVER** use `fmt.Println` or `log.Printf` for operational logging. ALWAYS use `log/slog` for structured logging.
*   **NEVER** bypass the batching InfluxDB client when writing metrics.
*   **NEVER** swallow panics in background goroutines without logging them.
*   **NEVER** write blocking code in the main `context.Done()` wait loop.
