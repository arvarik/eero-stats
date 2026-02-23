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
	"time"

	"github.com/arvarik/eero-stats/internal/db"

	"github.com/arvarik/eero-go/eero"
)

// Poller orchestrates the tiered data collection from the Eero API.
type Poller struct {
	client     *eero.Client
	influx     *db.InfluxClient
	networkURL string
}

// NewPoller creates a Poller that polls the given network and writes metrics
// to the provided InfluxDB client.
func NewPoller(client *eero.Client, influx *db.InfluxClient, networkURL string) *Poller {
	return &Poller{
		client:     client,
		influx:     influx,
		networkURL: networkURL,
	}
}

// Start begins the tiered polling daemon. It runs an immediate poll for all
// tiers on startup, then continues on their respective ticker intervals until
// the context is cancelled.
func (p *Poller) Start(ctx context.Context) {
	slog.Info("Starting Tiered Polling Daemon")

	fastTicker := time.NewTicker(3 * time.Minute)
	defer fastTicker.Stop()

	mediumTicker := time.NewTicker(90 * time.Minute)
	defer mediumTicker.Stop()

	slowTicker := time.NewTicker(12 * time.Hour)
	defer slowTicker.Stop()

	// Run all tiers immediately on startup before entering the ticker loop.
	p.safePollFast(ctx)
	p.safePollMedium(ctx)
	p.safePollSlow(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("Poller received cancellation signal, stopping loops")
			return
		case <-fastTicker.C:
			p.safePollFast(ctx)
		case <-mediumTicker.C:
			p.safePollMedium(ctx)
		case <-slowTicker.C:
			p.safePollSlow(ctx)
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
		var err error
		net, err = p.client.Network.Get(ctx, p.networkURL)
		return err
	})
	if err != nil {
		slog.Warn("Fast Poll Failed: Network.Get", "error", err)
	} else {
		p.writeNodeTimeSeries(net)
		p.writeNetworkHealth(net)
	}

	var devices []eero.Device
	err = p.withRetry(ctx, func() error {
		var err error
		devices, err = p.client.Device.List(ctx, p.networkURL)
		return err
	})
	if err != nil {
		slog.Warn("Fast Poll Failed: Device.List", "error", err)
	} else {
		p.writeClientDeviceTimeSeries(devices, net)
	}
}

// pollMedium collects slowly-changing metadata: node inventory, device details,
// and profile-to-device mappings.
func (p *Poller) pollMedium(ctx context.Context) {
	slog.Info("Running Medium Poll (Static Metadata)")

	var net *eero.NetworkDetails
	err := p.withRetry(ctx, func() error {
		var err error
		net, err = p.client.Network.Get(ctx, p.networkURL)
		return err
	})
	if err != nil {
		slog.Warn("Medium Poll Failed: Network.Get", "error", err)
	} else {
		p.writeNodeMetadata(net)
	}

	var devices []eero.Device
	err = p.withRetry(ctx, func() error {
		var err error
		devices, err = p.client.Device.List(ctx, p.networkURL)
		return err
	})
	if err != nil {
		slog.Warn("Medium Poll Failed: Device.List", "error", err)
	} else {
		p.writeClientMetadata(devices)
	}

	var profiles []eero.Profile
	err = p.withRetry(ctx, func() error {
		var err error
		profiles, err = p.client.Profile.List(ctx, p.networkURL)
		return err
	})
	if err != nil {
		slog.Warn("Medium Poll Failed: Profile.List", "error", err)
	} else {
		p.writeProfileMappings(profiles)
	}
}

// pollSlow collects infrequently-changing data: ISP speed test results and
// full network configuration snapshots.
func (p *Poller) pollSlow(ctx context.Context) {
	slog.Info("Running Slow Poll (Config & SLA)")

	var net *eero.NetworkDetails
	err := p.withRetry(ctx, func() error {
		var err error
		net, err = p.client.Network.Get(ctx, p.networkURL)
		return err
	})
	if err != nil {
		slog.Warn("Slow Poll Failed: Network.Get", "error", err)
		return
	}

	p.writeISPSpeeds(net)
	p.writeNetworkConfig(net)
}
