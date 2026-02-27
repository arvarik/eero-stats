.PHONY: all tidy lint test build version dashboard docker-up docker-down setup clean

# Build metadata
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS  := -s -w \
	-X github.com/arvarik/eero-stats/internal/version.Version=$(VERSION) \
	-X github.com/arvarik/eero-stats/internal/version.Commit=$(COMMIT) \
	-X github.com/arvarik/eero-stats/internal/version.BuildDate=$(BUILD_DATE)

# The primary binary output location
BIN_DIR := bin
BINARY_NAME := eero-stats

all: tidy lint test build

tidy:
	@echo "=> Running go mod tidy and formatting code..."
	go mod tidy
	go fmt ./...

lint:
	@echo "=> Running golangci-lint..."
	golangci-lint run ./...

test:
	@echo "=> Running tests..."
	go test -race -count=1 ./...

build:
	@echo "=> Building eero-stats binary..."
	mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/eero-stats

version:
	@echo "Version:    $(VERSION)"
	@echo "Commit:     $(COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"

dashboard:
	@echo "=> Regenerating Grafana dashboard JSON..."
	python3 scripts/build_dashboard.py

docker-up:
	@echo "=> Starting Docker Compose environment..."
	@mkdir -p data/app data/grafana
	@if [ "$$(stat -c '%u' data/app 2>/dev/null)" != "1000" ]; then \
		echo "=> Fixing data/app ownership for container user..."; \
		sudo chown -R 1000:1000 data/app; \
	fi
	@if [ "$$(stat -c '%u' data/grafana 2>/dev/null)" != "472" ]; then \
		echo "=> Fixing data/grafana ownership for container user..."; \
		sudo chown -R 472:0 data/grafana; \
	fi
	docker compose up -d

docker-down:
	@echo "=> Stopping Docker Compose environment..."
	docker compose down

setup:
	@echo "=> Configuring local git hooks..."
	git config core.hooksPath .githooks
	chmod +x .githooks/*
	@echo "✅ Pre-commit hooks installed."

clean:
	@echo "=> Cleaning up build artifacts..."
	rm -rf $(BIN_DIR)
