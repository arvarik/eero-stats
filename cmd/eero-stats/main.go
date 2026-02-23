// Binary eero-stats is a daemon that polls the Eero mesh network API on a
// tiered schedule and writes the collected metrics to InfluxDB for visualization
// in Grafana. It handles graceful shutdown via SIGTERM/SIGINT for clean
// container lifecycle management.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/arvarik/eero-stats/internal/auth"
	"github.com/arvarik/eero-stats/internal/config"
	"github.com/arvarik/eero-stats/internal/db"
	"github.com/arvarik/eero-stats/internal/poller"
	"github.com/arvarik/eero-stats/internal/version"
)

func main() {
	// Configure structured logger with human-readable text output.
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	slog.Info("eero-stats daemon starting up",
		"version", version.Version,
		"commit", version.Commit,
		"built", version.BuildDate,
	)

	// Initialize context with graceful shutdown hooked to SIGTERM (Docker)
	// and SIGINT (Ctrl+C).
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		slog.Info("Received termination signal, shutting down gracefully", "signal", sig)
		signal.Stop(sigCh) // Deregister to allow force-kill on second signal.
		cancel()
	}()

	// Load configuration from environment variables (and optional .env file).
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Authenticate with the Eero cloud API (restores cached session or
	// performs interactive 2FA login via stdin).
	eeroClient, err := auth.Init(ctx, cfg)
	if err != nil {
		slog.Error("Failed to authenticate with Eero", "error", err)
		os.Exit(1)
	}

	// Fetch the account to discover the primary network URL.
	acct, err := eeroClient.Account.Get(ctx)
	if err != nil {
		slog.Error("Failed to fetch Eero account details", "error", err)
		os.Exit(1)
	}
	if acct.Networks.Count == 0 {
		slog.Error("No networks found on this account")
		os.Exit(1)
	}
	networkURL := acct.Networks.Data[0].URL

	// Initialize the InfluxDB client with NVMe-optimized batching settings.
	influxClient := db.NewInfluxClient(cfg)
	defer influxClientShutdown(influxClient)

	// Start the polling daemon in a goroutine so the main goroutine can
	// block on context cancellation for orderly shutdown.
	var wg sync.WaitGroup
	daemon := poller.NewPoller(eeroClient, influxClient, networkURL)

	wg.Add(1)
	go func() {
		defer wg.Done()
		daemon.Start(ctx)
	}()

	slog.Info("Application initialized and polling started")
	<-ctx.Done()

	slog.Info("Context cancelled, waiting for daemon graceful loop termination...")
	wg.Wait()

	slog.Info("Main daemon loop exiting")
}

// influxClientShutdown gracefully flushes buffered writes and closes the
// InfluxDB connection with a 15-second timeout.
func influxClientShutdown(client *db.InfluxClient) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client.Shutdown(shutdownCtx)
}
