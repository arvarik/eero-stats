// Package poller implements a tiered polling daemon that periodically fetches
// data from the Eero API and writes it to InfluxDB. Three polling tiers run at
// different intervals to balance data freshness against API rate limits:
//
//   - Fast  (3 min):  Device connectivity, node health, network status
//   - Medium (90 min): Node/device metadata, profile mappings
//   - Slow  (12 hr):  ISP speed tests, network configuration snapshots
package poller

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/arvarik/eero-go/eero"
)

// Poller orchestrates the tiered data collection from the Eero API.
type Poller struct {
	client     EeroClient
	influx     MetricWriter
	networkURL string
}

// NewPoller creates a Poller that polls the given network and writes metrics
// to the provided MetricWriter.
func NewPoller(client EeroClient, influx MetricWriter, networkURL string) *Poller {
	return &Poller{
		client:     client,
		influx:     influx,
		networkURL: networkURL,
	}
}

// Start begins the tiered polling daemon. Each tier runs in its own goroutine
// with an immediate initial poll followed by periodic ticks. This prevents a
// slow poll in one tier from blocking others.
func (p *Poller) Start(ctx context.Context) {
	slog.Info("Starting Tiered Polling Daemon")

	var wg sync.WaitGroup

	// Launch separate goroutines for each polling tier.
	// This prevents long-running polls in one tier from blocking others.
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.runTier(ctx, "Fast", 3*time.Minute, p.safePollFast)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		p.runTier(ctx, "Medium", 90*time.Minute, p.safePollMedium)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		p.runTier(ctx, "Slow", 12*time.Hour, p.safePollSlow)
	}()

	// Wait for all tiers to shut down.
	wg.Wait()
	slog.Info("All polling loops stopped")
}

// runTier executes the polling function immediately, then periodically at the
// given interval until the context is cancelled.
func (p *Poller) runTier(ctx context.Context, name string, interval time.Duration, pollFunc func(context.Context)) {
	slog.Info("Starting poll loop", "tier", name, "interval", interval)

	// Run immediate poll.
	pollFunc(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Poll loop stopping", "tier", name)
			return
		case <-ticker.C:
			pollFunc(ctx)
		}
	}
}

// ---------------------------------------------------------------------------
// Panic-safe wrappers
// ---------------------------------------------------------------------------

func (p *Poller) safePollFast(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Panic recovered in Fast Poll", "panic", r)
		}
	}()
	p.pollFast(ctx)
}

func (p *Poller) safePollMedium(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Panic recovered in Medium Poll", "panic", r)
		}
	}()
	p.pollMedium(ctx)
}

func (p *Poller) safePollSlow(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Panic recovered in Slow Poll", "panic", r)
		}
	}()
	p.pollSlow(ctx)
}

// ---------------------------------------------------------------------------
// Tier implementations
// ---------------------------------------------------------------------------

// pollFast collects high-frequency time-series data: device connectivity
// metrics, node health, and overall network status.
func (p *Poller) pollFast(ctx context.Context) {
	slog.Info("Running Fast Poll (Time-Series Metrics)")

	var net *eero.NetworkDetails
	err := p.withRetry(ctx, func() error {
		var retryErr error
		net, retryErr = p.client.GetNetwork(ctx, p.networkURL)
		return retryErr
	})
	if err != nil {
		slog.Warn("Fast Poll Failed: GetNetwork", "error", err)
	} else {
		p.writeNodeTimeSeries(net)
		p.writeNetworkHealth(net)
	}

	var devices []eero.Device
	err = p.withRetry(ctx, func() error {
		var retryErr error
		devices, retryErr = p.client.ListDevices(ctx, p.networkURL)
		return retryErr
	})
	if err != nil {
		slog.Warn("Fast Poll Failed: ListDevices", "error", err)
	} else {
		p.writeClientDeviceTimeSeries(devices, net)
	}
}

// pollMedium collects slowly-changing metadata: node inventory, device details,
// and profile-to-device mappings.
func (p *Poller) pollMedium(ctx context.Context) {
	slog.Info("Running Medium Poll (Static Metadata)")

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		var net *eero.NetworkDetails
		err := p.withRetry(ctx, func() error {
			var retryErr error
			net, retryErr = p.client.GetNetwork(ctx, p.networkURL)
			return retryErr
		})
		if err != nil {
			slog.Warn("Medium Poll Failed: GetNetwork", "error", err)
		} else {
			p.writeNodeMetadata(net)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		var devices []eero.Device
		err := p.withRetry(ctx, func() error {
			var retryErr error
			devices, retryErr = p.client.ListDevices(ctx, p.networkURL)
			return retryErr
		})
		if err != nil {
			slog.Warn("Medium Poll Failed: ListDevices", "error", err)
		} else {
			p.writeClientMetadata(devices)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		var profiles []eero.Profile
		err := p.withRetry(ctx, func() error {
			var retryErr error
			profiles, retryErr = p.client.ListProfiles(ctx, p.networkURL)
			return retryErr
		})
		if err != nil {
			slog.Warn("Medium Poll Failed: ListProfiles", "error", err)
		} else {
			p.writeProfileMappings(profiles)
		}
	}()

	wg.Wait()
}

// pollSlow collects infrequently-changing data: ISP speed test results and
// full network configuration snapshots.
func (p *Poller) pollSlow(ctx context.Context) {
	slog.Info("Running Slow Poll (Config & SLA)")

	var net *eero.NetworkDetails
	err := p.withRetry(ctx, func() error {
		var retryErr error
		net, retryErr = p.client.GetNetwork(ctx, p.networkURL)
		return retryErr
	})
	if err != nil {
		slog.Warn("Slow Poll Failed: GetNetwork", "error", err)
		return
	}

	p.writeISPSpeeds(net)
	p.writeNetworkConfig(net)
}
