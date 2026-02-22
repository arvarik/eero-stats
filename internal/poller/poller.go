package poller

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/arvarik/eero-stats/internal/db"

	"github.com/arvarik/eero-go/eero"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

type Poller struct {
	client     *eero.Client
	influx     *db.InfluxClient
	networkURL string
}

func NewPoller(client *eero.Client, influx *db.InfluxClient, networkURL string) *Poller {
	return &Poller{
		client:     client,
		influx:     influx,
		networkURL: networkURL,
	}
}

// Start begins the tiered polling daemon.
func (p *Poller) Start(ctx context.Context) {
	slog.Info("Starting Tiered Polling Daemon")

	// Fast Loop: Every 3 minutes for devices/nodes
	fastTicker := time.NewTicker(3 * time.Minute)
	defer fastTicker.Stop()

	// Slow Loop: Every 12 hours for ISP speed tests
	slowTicker := time.NewTicker(12 * time.Hour)
	defer slowTicker.Stop()

	// Trigger an immediate initial poll of both
	p.safePollFast(ctx)
	p.safePollSlow(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("Poller received cancellation signal, stopping loops")
			return
		case <-fastTicker.C:
			p.safePollFast(ctx)
		case <-slowTicker.C:
			p.safePollSlow(ctx)
		}
	}
}

// safePollFast wraps the device execution loop to swallow and recover panics
func (p *Poller) safePollFast(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Panic recovered in Fast Poll. Preventing container crash.", "panic", r)
		}
	}()
	p.pollFast(ctx)
}

// safePollSlow wraps the ISP execution loop to swallow and recover panics
func (p *Poller) safePollSlow(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Panic recovered in Slow Poll. Preventing container crash.", "panic", r)
		}
	}()
	p.pollSlow(ctx)
}

// --- Polling Loops ---

func (p *Poller) pollFast(ctx context.Context) {
	slog.Info("Running Fast Poll (Devices & Network Health)")

	// 1. Fetch Network Nodes
	var net *eero.NetworkDetails
	err := p.withRetry(ctx, func() error {
		var err error
		net, err = p.client.Network.Get(ctx, p.networkURL)
		return err
	})
	if err != nil {
		slog.Warn("Fast Poll Failed: Network.Get", "error", err)
	} else {
		p.writeNodes(net)
	}

	// 2. Fetch Connected Devices
	var devices []eero.Device
	err = p.withRetry(ctx, func() error {
		var err error
		devices, err = p.client.Device.List(ctx, p.networkURL)
		return err
	})
	if err != nil {
		slog.Warn("Fast Poll Failed: Device.List", "error", err)
	} else {
		p.writeDevices(devices)
	}
}

func (p *Poller) pollSlow(ctx context.Context) {
	slog.Info("Running Slow Poll (ISP Speed Test)")

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

	p.writeNetworkSpeed(net)
}

// --- Data Mapping & InfluxDB Writers ---

func (p *Poller) writeNetworkSpeed(net *eero.NetworkDetails) {
	tags := map[string]string{
		"network_name": net.Name,
	}
	fields := map[string]interface{}{
		"speed_down_mbps": net.Speed.Down.Value,
		"speed_up_mbps":   net.Speed.Up.Value,
	}

	if net.WanIP != "" {
		fields["wan_ip"] = net.WanIP
	}
	if net.GatewayIP != "" {
		fields["gateway_ip"] = net.GatewayIP
	}
	fields["ipv6_upstream"] = net.IPv6Upstream
	fields["dhcp_mode"] = net.DHCP.Mode
	fields["dns_caching_enabled"] = net.DNS.Caching
	fields["adblock_enabled"] = net.PremiumDNS.AdBlockSettings.Enabled

	pt := influxdb2.NewPoint("eero_network", tags, fields, time.Now())
	p.influx.WriteAPI.WritePoint(pt)
}

func (p *Poller) writeNodes(net *eero.NetworkDetails) {
	now := time.Now()
	for _, node := range net.Eeros.Data {
		tags := map[string]string{
			"serial": node.Serial,
			"model":  node.Model,
		}
		if node.Location != "" {
			tags["location"] = node.Location
		}

		fields := map[string]interface{}{
			"status": node.Status,
		}

		fields["connected_clients"] = node.ConnectedClientsCount
		fields["mesh_quality_bars"] = node.MeshQualityBars
		fields["wired"] = node.Wired
		fields["using_wan"] = node.UsingWan
		if node.OSVersion != "" {
			fields["os_version"] = node.OSVersion
		}

		pt := influxdb2.NewPoint("eero_nodes", tags, fields, now)
		p.influx.WriteAPI.WritePoint(pt)
	}
}

func (p *Poller) writeDevices(devices []eero.Device) {
	now := time.Now()
	for _, d := range devices {
		tags := map[string]string{
			"mac":         d.MAC,
			"device_type": d.DeviceType,
		}
		if d.Nickname != nil {
			tags["nickname"] = *d.Nickname
		}

		// Map 'is_guest' directly from struct
		isGuest := "false"
		if d.IsGuest {
			isGuest = "true"
		}
		tags["is_guest"] = isGuest

		fields := map[string]interface{}{
			"connected": d.Connected,
		}

		fields["score_bars"] = d.Connectivity.ScoreBars
		fields["signal"] = d.Connectivity.Signal
		fields["rx_bitrate"] = d.Connectivity.RxBitrate
		fields["channel"] = d.Channel

		if d.VlanID != nil {
			fields["vlan_id"] = *d.VlanID
		}

		if d.Usage != nil {
			fields["usage_download"] = d.Usage.Download
			fields["usage_upload"] = d.Usage.Upload
		}

		pt := influxdb2.NewPoint("eero_devices", tags, fields, now)
		p.influx.WriteAPI.WritePoint(pt)
	}
}

// --- Helpers ---

// withRetry implements an exponential backoff retry for network requests.
func (p *Poller) withRetry(ctx context.Context, op func() error) error {
	const maxRetries = 3
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		err = op()
		if err == nil {
			return nil
		}

		// Only sleep if we have more retries left
		if attempt < maxRetries-1 {
			// Exponential backoff: 2s, 4s
			backoff := time.Duration(math.Pow(2, float64(attempt+1))) * time.Second
			slog.Warn(fmt.Sprintf("API call failed (attempt %d/%d). Retrying in %v...", attempt+1, maxRetries, backoff), "error", err)

			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return fmt.Errorf("after %d attempts, last error: %w", maxRetries, err)
}

// Ensure the unused `write` import from standard snippet doesn't break build
var _ = write.Point{}
