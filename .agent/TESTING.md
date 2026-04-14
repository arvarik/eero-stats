# Testing & Verification Strategy

## 1. Local Development Setup
**Prerequisites:** Docker, Docker Compose, Go 1.22+, GNU Make.

**Startup Command:**
```bash
make docker-up
```
This starts the daemon, InfluxDB, and Grafana.

**Initial Authentication (2FA):**
```bash
docker attach eero-stats
# Enter OTP, then detach with CTRL-P, CTRL-Q
```

**Dashboard:**
Access Grafana at `http://localhost:<GRAFANA_PORT>` (default `3000`).

## 2. Test Execution Commands
*   **Format & Tidy:** `make tidy` (runs `go mod tidy` and `go fmt ./...`)
*   **Linting:** `make lint` (runs `golangci-lint run ./...`)
*   **Unit Tests:** `make test` (runs `go test -race -count=1 ./...`)
*   **Dashboard Generation:** `make dashboard` (runs `python3 scripts/build_dashboard.py`)
*   **Build Binary:** `make build` (builds to `bin/eero-stats`)
*   **Clean:** `make clean`

## 3. Execution Evidence Rules
Before marking a PR or feature branch as complete, the agent MUST run the relevant test commands and paste the failing/passing output into the PR description or final validation step.

**Mandatory Checks:**
1. `make test` must pass.
2. `make lint` must pass.
3. `make tidy` must not introduce uncommitted changes.

## 4. Acceptance Criteria Templates
*   (Empty scenario tables ready for the first feature)

### Feature: [Feature Name]
| Scenario ID | Description | Preconditions | Action | Expected Result | Evidence (Log/Output) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `SCN-01` | | | | | |

## 5. Regression Scenarios
| Scenario ID | Feature | Description | Last Verified |
| :--- | :--- | :--- | :--- |
| `REG-01` | Authentication | Re-authenticate when session token expires | [Date] |
