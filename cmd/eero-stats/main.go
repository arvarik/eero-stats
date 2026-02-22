package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"eero-stats/internal/auth"
	"eero-stats/internal/config"
	"eero-stats/internal/db"
	"eero-stats/internal/poller"
)

func main() {
	// Configure structured logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	slog.Info("eero-stats daemon starting up")

	// 1. Initialize context with graceful shutdown hooked to SIGTERM (Docker standard)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		slog.Info("Received termination signal, shutting down gracefully", "signal", sig)
		cancel()
	}()

	// 2. Load Environment Config
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// 3. Authenticate with Eero
	eeroClient, err := auth.Init(ctx, cfg)
	if err != nil {
		slog.Error("Failed to authenticate with Eero", "error", err)
		os.Exit(1)
	}

	// Fetch account to get the primary network URL
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

	// 4. Spin up NVMe-Optimized InfluxDB Client
	influxClient := db.NewInfluxClient(cfg)
	defer luxClientShutdown(ctx, influxClient)

	// 5. Start the Polling Daemon
	var wg sync.WaitGroup
	daemon := poller.NewPoller(eeroClient, influxClient, networkURL)

	wg.Add(1)
	go func() {
		defer wg.Done()
		daemon.Start(ctx)
	}()

	// Block until graceful shutdown via context cancellation
	slog.Info("Application initialized and polling started")
	<-ctx.Done()

	slog.Info("Context cancelled, waiting for daemon graceful loop termination...")
	wg.Wait()

	slog.Info("Main daemon loop exiting")
}

func luxClientShutdown(ctx context.Context, client *db.InfluxClient) {
	// Add a timeout for the shutdown if needed
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client.Shutdown(shutdownCtx)
}
