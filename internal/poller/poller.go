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
		p.runTier(ctx, "Fast", 3*time.Minute, func(ctx context.Context) {
			p.safePoll(ctx, "Fast", p.pollFast)
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		p.runTier(ctx, "Medium", 90*time.Minute, func(ctx context.Context) {
			p.safePoll(ctx, "Medium", p.pollMedium)
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		p.runTier(ctx, "Slow", 12*time.Hour, func(ctx context.Context) {
			p.safePoll(ctx, "Slow", p.pollSlow)
		})
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
// Polling Helpers
// ---------------------------------------------------------------------------

// safePoll wraps a polling function with panic recovery.
func (p *Poller) safePoll(ctx context.Context, tier string, pollFunc func(context.Context)) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Panic recovered in Poll", "tier", tier, "panic", r)
		}
	}()
	pollFunc(ctx)
}

// pollWithRetry executes an API operation with retry logic and structured logging.
func (p *Poller) pollWithRetry(ctx context.Context, tier, name string, op func() error, onSuccess func()) error {
	err := p.withRetry(ctx, op)
	if err != nil {
		slog.Warn(tier+" Poll Failed: "+name, "error", err)
		return err
	}
	if onSuccess != nil {
		onSuccess()
	}
	return nil
}

// pollAsync executes pollWithRetry in a goroutine and tracks it with a WaitGroup.
func (p *Poller) pollAsync(ctx context.Context, wg *sync.WaitGroup, tier, name string, op func() error, onSuccess func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = p.pollWithRetry(ctx, tier, name, op, onSuccess)
	}()
}


// ---------------------------------------------------------------------------
// Tier implementations
// ---------------------------------------------------------------------------

// pollFast collects high-frequency time-series data: device connectivity
// metrics, node health, and overall network status.
func (p *Poller) pollFast(ctx context.Context) {
	slog.Info("Running Fast Poll (Time-Series Metrics)")

	var wg sync.WaitGroup
	var net *eero.NetworkDetails
	var devices []eero.Device
	var devErr error

	p.pollAsync(ctx, &wg, "Fast", "GetNetwork", func() error {
		var err error
		net, err = p.client.GetNetwork(ctx, p.networkURL)
		return err
	}, func() {
		p.writeNodeTimeSeries(net)
		p.writeNetworkHealth(net)
	})

	p.pollAsync(ctx, &wg, "Fast", "ListDevices", func() error {
		var err error
		devices, err = p.client.ListDevices(ctx, p.networkURL)
		devErr = err
		return err
	}, nil)

	wg.Wait()

	if devErr == nil {
		p.writeClientDeviceTimeSeries(devices, net)
	}
}

// pollMedium collects slowly-changing metadata: node inventory, device details,
// and profile-to-device mappings.
func (p *Poller) pollMedium(ctx context.Context) {
	slog.Info("Running Medium Poll (Static Metadata)")

	var wg sync.WaitGroup

	p.pollAsync(ctx, &wg, "Medium", "GetNetwork", func() error {
		var net *eero.NetworkDetails
		var err error
		net, err = p.client.GetNetwork(ctx, p.networkURL)
		if err == nil {
			p.writeNodeMetadata(net)
		}
		return err
	}, nil)

	p.pollAsync(ctx, &wg, "Medium", "ListDevices", func() error {
		var devices []eero.Device
		var err error
		devices, err = p.client.ListDevices(ctx, p.networkURL)
		if err == nil {
			p.writeClientMetadata(devices)
		}
		return err
	}, nil)

	p.pollAsync(ctx, &wg, "Medium", "ListProfiles", func() error {
		var profiles []eero.Profile
		var err error
		profiles, err = p.client.ListProfiles(ctx, p.networkURL)
		if err == nil {
			p.writeProfileMappings(profiles)
		}
		return err
	}, nil)

	wg.Wait()
}

// pollSlow collects infrequently-changing data: ISP speed test results and
// full network configuration snapshots.
func (p *Poller) pollSlow(ctx context.Context) {
	slog.Info("Running Slow Poll (Config & SLA)")

	var net *eero.NetworkDetails
	_ = p.pollWithRetry(ctx, "Slow", "GetNetwork", func() error {
		var err error
		net, err = p.client.GetNetwork(ctx, p.networkURL)
		return err
	}, func() {
		p.writeISPSpeeds(net)
		p.writeNetworkConfig(net)
	})
}
