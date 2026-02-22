.PHONY: all tidy lint build docker-up docker-down clean

# The primary binary output location
BIN_DIR := bin
BINARY_NAME := eero-stats

all: tidy lint build

tidy:
	@echo "=> Running go mod tidy and formatting code..."
	go mod tidy
	go fmt ./...

lint:
	@echo "=> Running golangci-lint..."
	# Assuming golangci-lint is installed locally or in PATH
	golangci-lint run ./...

build:
	@echo "=> Building eero-stats binary..."
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/eero-stats

docker-up:
	@echo "=> Starting Docker Compose environment..."
	@mkdir -p data/app data/grafana
	@if [ "$$(stat -c '%u' data/app 2>/dev/null)" != "100" ]; then \
		echo "=> Fixing data/app ownership for container user..."; \
		sudo chown -R 100:101 data/app; \
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
