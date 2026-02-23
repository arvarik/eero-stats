package poller

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
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

	fastTicker := time.NewTicker(3 * time.Minute)
	defer fastTicker.Stop()

	mediumTicker := time.NewTicker(90 * time.Minute)
	defer mediumTicker.Stop()

	slowTicker := time.NewTicker(12 * time.Hour)
	defer slowTicker.Stop()

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

// Polling Loop Wrappers

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

// --- Polling Logic ---

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

// --- Data Writers ---

// FAST POLL
func (p *Poller) writeClientDeviceTimeSeries(devices []eero.Device, net *eero.NetworkDetails) {
	now := time.Now()

	// Create map for node locations out of net for resolving node names
	nodeMap := make(map[string]string)
	if net != nil {
		for _, n := range net.Eeros.Data {
			nodeMap[n.Location] = fmt.Sprintf("%s - %s", n.Location, n.Model)
		}
	}

	for _, d := range devices {
		deviceName := d.MAC
		if d.Nickname != nil && *d.Nickname != "" {
			deviceName = *d.Nickname
		} else if d.Hostname != nil && *d.Hostname != "" {
			deviceName = *d.Hostname
		}

		nodeName := d.Source.Location
		if resolvedName, ok := nodeMap[d.Source.Location]; ok {
			nodeName = resolvedName
		}

		tags := map[string]string{
			"mac":             d.MAC,
			"device_name":     deviceName,
			"source_location": d.Source.Location,
			"node_name":       nodeName,
			"connection_type": d.ConnectionType,
			"frequency":       d.Interface.Frequency,
			"frequency_unit":  d.Interface.FrequencyUnit,
		}

		fields := map[string]interface{}{
			"connected":   d.Connected,
			"score_bars":  d.Connectivity.ScoreBars,
			"score":       d.Connectivity.Score,
			"paused":      d.Paused,
			"is_guest":    d.IsGuest,
			"blacklisted": d.Blacklisted,
			"channel":     d.Channel,
		}

		if strings.HasSuffix(d.Connectivity.Signal, " dBm") {
			s := strings.TrimSuffix(d.Connectivity.Signal, " dBm")
			val, err := strconv.Atoi(s)
			if err == nil {
				fields["signal"] = val
			}
		}
		if d.Connectivity.SignalAvg != nil {
			if strings.HasSuffix(*d.Connectivity.SignalAvg, " dBm") {
				s := strings.TrimSuffix(*d.Connectivity.SignalAvg, " dBm")
				val, err := strconv.Atoi(s)
				if err == nil {
					fields["signal_avg"] = val
				}
			}
		}

		if d.Connectivity.RxRateInfo.RateBps != nil {
			fields["rx_rate_bps"] = *d.Connectivity.RxRateInfo.RateBps
		}
		if d.Connectivity.TxRateInfo.RateBps != nil {
			fields["tx_rate_bps"] = *d.Connectivity.TxRateInfo.RateBps
		}
		if d.Connectivity.RxRateInfo.ChannelWidth != nil {
			fields["rx_channel_width"] = *d.Connectivity.RxRateInfo.ChannelWidth
		}
		if d.Connectivity.RxRateInfo.MCS != nil {
			fields["rx_mcs"] = *d.Connectivity.RxRateInfo.MCS
		}

		pt := influxdb2.NewPoint("eero_client_timeseries", tags, fields, now)
		p.influx.WriteAPI.WritePoint(pt)
	}
}

func (p *Poller) writeNodeTimeSeries(net *eero.NetworkDetails) {
	now := time.Now()
	for _, node := range net.Eeros.Data {
		nodeName := fmt.Sprintf("%s - %s", node.Location, node.Model)
		tags := map[string]string{
			"serial":    node.Serial,
			"location":  node.Location,
			"model":     node.Model,
			"node_name": nodeName,
		}

		fields := map[string]interface{}{
			"connected_clients_count": node.ConnectedClientsCount,
			"mesh_quality_bars":       node.MeshQualityBars,
			"heartbeat_ok":            node.HeartbeatOK,
			"status":                  node.Status,
			"state":                   node.State,
			"using_wan":               node.UsingWan,
			"power_source":            node.PowerInfo.PowerSource,
			"connection_type":         node.ConnectionType,
			"led_on":                  node.LedOn,
		}

		pt := influxdb2.NewPoint("eero_node_timeseries", tags, fields, now)
		p.influx.WriteAPI.WritePoint(pt)
	}
}

func (p *Poller) writeNetworkHealth(net *eero.NetworkDetails) {
	tags := map[string]string{
		"network_name": net.Name,
	}
	fields := map[string]interface{}{
		"isp_up":              net.Health.Internet.ISPUp,
		"internet_status":     net.Health.Internet.Status,
		"eero_network_status": net.Status,
	}

	pt := influxdb2.NewPoint("eero_network_health", tags, fields, time.Now())
	p.influx.WriteAPI.WritePoint(pt)
}

// MEDIUM POLL
func (p *Poller) writeNodeMetadata(net *eero.NetworkDetails) {
	now := time.Now()
	for _, node := range net.Eeros.Data {
		nodeName := fmt.Sprintf("%s - %s", node.Location, node.Model)
		tags := map[string]string{
			"serial":    node.Serial,
			"node_name": nodeName,
		}

		fields := map[string]interface{}{
			"ip_address":         node.IPAddress,
			"mac_address":        node.MACAddress,
			"os_version":         node.OSVersion,
			"model_number":       node.ModelNumber,
			"update_available":   node.UpdateAvailable,
			"wired":              node.Wired,
			"gateway":            node.Gateway,
			"is_primary_node":    node.IsPrimaryNode,
			"led_on":             node.LedOn,
			"last_heartbeat":     node.LastHeartbeat.UnixMilli(),
			"joined":             node.Joined.Time.Format(time.RFC3339),
			"ethernet_addresses": strings.Join(node.EthernetAddresses, ","),
			"wifi_bssids":        strings.Join(node.WifiBSSIDs, ","),
			"bands":              strings.Join(node.Bands, ","),
		}

		pt := influxdb2.NewPoint("eero_node_metadata", tags, fields, now)
		p.influx.WriteAPI.WritePoint(pt)
	}
}

func (p *Poller) writeClientMetadata(devices []eero.Device) {
	now := time.Now()
	for _, d := range devices {
		deviceName := d.MAC
		if d.Nickname != nil && *d.Nickname != "" {
			deviceName = *d.Nickname
		} else if d.Hostname != nil && *d.Hostname != "" {
			deviceName = *d.Hostname
		}

		tags := map[string]string{
			"mac":         d.MAC,
			"device_name": deviceName,
		}

		fields := map[string]interface{}{
			"device_type":     d.DeviceType,
			"ipv4":            d.IPv4,
			"is_proxied_node": d.IsProxiedNode,
			"is_private_mac":  d.IsPrivate,
			"is_guest":        d.IsGuest,
			"blacklisted":     d.Blacklisted,
			"paused":          d.Paused,
			"auth":            d.Auth,
			"ssid":            d.SSID,
			"subnet_kind":     d.SubnetKind,
			"vlan_name":       d.VlanName,
			"first_active":    d.FirstActive.Time.Format(time.RFC3339),
			"last_active":     d.LastActive.Time.Format(time.RFC3339),
		}

		if d.Manufacturer != nil {
			fields["manufacturer"] = *d.Manufacturer
		}
		if d.IP != nil {
			fields["ip"] = *d.IP
		}
		if len(d.IPv6Addresses) > 0 {
			fields["ipv6"] = d.IPv6Addresses[0].Address
		}
		if d.VlanID != nil {
			fields["vlan_id"] = *d.VlanID
		}

		pt := influxdb2.NewPoint("eero_client_metadata", tags, fields, now)
		p.influx.WriteAPI.WritePoint(pt)
	}
}

func (p *Poller) writeProfileMappings(profiles []eero.Profile) {
	now := time.Now()
	for _, prof := range profiles {
		tags := map[string]string{
			"profile_name": prof.Name,
		}

		var macs []string
		for _, dev := range prof.Devices {
			macs = append(macs, dev.MAC)
		}

		fields := map[string]interface{}{
			"devices":            strings.Join(macs, ","),
			"paused":             prof.Paused,
			"block_apps":         prof.BlockApps,
			"safe_search_active": prof.SafeSearchActive,
		}

		pt := influxdb2.NewPoint("eero_profile_mappings", tags, fields, now)
		p.influx.WriteAPI.WritePoint(pt)
	}
}

// SLOW POLL
func (p *Poller) writeISPSpeeds(net *eero.NetworkDetails) {
	tags := map[string]string{
		"network_name": net.Name,
	}
	fields := map[string]interface{}{
		"speed_down_mbps": net.Speed.Down.Value,
		"speed_up_mbps":   net.Speed.Up.Value,
	}

	pt := influxdb2.NewPoint("eero_isp_speed", tags, fields, time.Now())
	p.influx.WriteAPI.WritePoint(pt)
}

func (p *Poller) writeNetworkConfig(net *eero.NetworkDetails) {
	tags := map[string]string{
		"network_name": net.Name,
	}
	dhcpRouter := ""
	if net.Lease.DHCP != nil {
		dhcpRouter = net.Lease.DHCP.Router
	}

	fields := map[string]interface{}{
		"premium_status":        net.PremiumStatus,
		"premium_tier":          net.PremiumDetails.Tier,
		"dns_policies_enabled":  net.PremiumDNS.DNSPoliciesEnabled,
		"ad_block_enabled":      net.PremiumDNS.AdBlockSettings.Enabled,
		"block_malware_enabled": net.PremiumDNS.DNSPolicies.BlockMalware,
		"dhcp_mode":             net.DHCP.Mode,
		"dhcp_router":           dhcpRouter,
		"dns_mode":              net.DNS.Mode,
		"dns_caching":           net.DNS.Caching,
		"dns_parent_ips":        strings.Join(net.DNS.Parent.IPs, ","),
		"geoip_country":         net.GeoIP.CountryName,
		"geoip_region":          net.GeoIP.Region,
		"geoip_region_name":     net.GeoIP.RegionName,
		"geoip_city":            net.GeoIP.City,
		"geoip_timezone":        net.GeoIP.Timezone,
		"geoip_isp":             net.GeoIP.ISP,
		"geoip_org":             net.GeoIP.Org,
		"geoip_asn":             net.GeoIP.ASN,
		"wan_type":              net.WanType,
		"wireless_mode":         net.WirelessMode,
		"mlo_mode":              net.MloMode,
		"band_steering":         net.BandSteering,
		"wpa3_enabled":          net.Wpa3,
		"upnp_enabled":          net.UpnpEnabled,
		"ipv6_upstream":         net.IPv6Upstream,
		"thread_enabled":        net.ThreadEnabled,
		"sqm_enabled":           net.SQMEnabled,
		"double_nat":            net.IPSettings.DoubleNAT,
		"public_ip":             net.IPSettings.PublicIP,
		"guest_network_enabled": net.GuestNetwork.Enabled,
		"guest_network_name":    net.GuestNetwork.Name,
		"firmware_has_update":   net.Updates.HasUpdate,
		"firmware_target":       net.Updates.TargetFirmware,
		"firmware_update_req":   net.Updates.UpdateRequired,
	}

	pt := influxdb2.NewPoint("eero_network_config", tags, fields, time.Now())
	p.influx.WriteAPI.WritePoint(pt)
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

		if attempt < maxRetries-1 {
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

// Ensure write import not stripped
var _ = write.Point{}
