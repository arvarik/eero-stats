package poller

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/arvarik/eero-go/eero"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

// resolveDeviceName returns the best human-readable name for a device,
// preferring nickname > hostname > MAC address.
func resolveDeviceName(d *eero.Device) string {
	if d.Nickname != nil && *d.Nickname != "" {
		return *d.Nickname
	}
	if d.Hostname != nil && *d.Hostname != "" {
		return *d.Hostname
	}
	return d.MAC
}

// ---------------------------------------------------------------------------
// Fast Poll Writers (every 3 minutes)
// ---------------------------------------------------------------------------

// writeClientDeviceTimeSeries writes per-device connectivity metrics such as
// signal strength, link rates, and connection state to InfluxDB.
func (p *Poller) writeClientDeviceTimeSeries(devices []eero.Device, net *eero.NetworkDetails) {
	now := time.Now()

	// Build a lookup map from node location to a friendly "Location - Model" name.
	nodeMap := make(map[string]string)
	if net != nil {
		for i := range net.Eeros.Data {
			nodeMap[net.Eeros.Data[i].Location] = fmt.Sprintf("%s - %s", net.Eeros.Data[i].Location, net.Eeros.Data[i].Model)
		}
	}

	for i := range devices {
		d := &devices[i]
		deviceName := resolveDeviceName(d)

		nodeName := d.Source.Location
		if resolvedName, ok := nodeMap[d.Source.Location]; ok {
			nodeName = resolvedName
		}

		pt := influxdb2.NewPointWithMeasurement("eero_client_timeseries").
			AddTag("mac", d.MAC).
			AddTag("device_name", deviceName).
			AddTag("source_location", d.Source.Location).
			AddTag("node_name", nodeName).
			AddTag("connection_type", d.ConnectionType).
			AddTag("frequency", d.Interface.Frequency).
			AddTag("frequency_unit", d.Interface.FrequencyUnit).
			AddField("connected", d.Connected).
			AddField("score_bars", d.Connectivity.ScoreBars).
			AddField("score", d.Connectivity.Score).
			AddField("paused", d.Paused).
			AddField("is_guest", d.IsGuest).
			AddField("blacklisted", d.Blacklisted).
			AddField("channel", d.Channel).
			SetTime(now)

		// Parse signal strength from the "NN dBm" string format.
		if val, err := parseSignalDBm(d.Connectivity.Signal); err == nil {
			pt.AddField("signal", val)
		}
		if d.Connectivity.SignalAvg != nil {
			if val, err := parseSignalDBm(*d.Connectivity.SignalAvg); err == nil {
				pt.AddField("signal_avg", val)
			}
		}

		if d.Connectivity.RxRateInfo.RateBps != nil {
			pt.AddField("rx_rate_bps", *d.Connectivity.RxRateInfo.RateBps)
		}
		if d.Connectivity.TxRateInfo.RateBps != nil {
			pt.AddField("tx_rate_bps", *d.Connectivity.TxRateInfo.RateBps)
		}
		if d.Connectivity.RxRateInfo.ChannelWidth != nil {
			pt.AddField("rx_channel_width", *d.Connectivity.RxRateInfo.ChannelWidth)
		}
		if d.Connectivity.RxRateInfo.MCS != nil {
			pt.AddField("rx_mcs", *d.Connectivity.RxRateInfo.MCS)
		}

		p.influx.WritePoint(pt)
	}
}

// writeNodeTimeSeries writes per-node health metrics (client count, mesh quality,
// heartbeat, power source, etc.) to InfluxDB.
func (p *Poller) writeNodeTimeSeries(net *eero.NetworkDetails) {
	now := time.Now()
	for i := range net.Eeros.Data {
		node := &net.Eeros.Data[i]
		nodeName := fmt.Sprintf("%s - %s", node.Location, node.Model)
		pt := influxdb2.NewPointWithMeasurement("eero_node_timeseries").
			AddTag("serial", node.Serial).
			AddTag("location", node.Location).
			AddTag("model", node.Model).
			AddTag("node_name", nodeName).
			AddField("connected_clients_count", node.ConnectedClientsCount).
			AddField("mesh_quality_bars", node.MeshQualityBars).
			AddField("heartbeat_ok", node.HeartbeatOK).
			AddField("status", node.Status).
			AddField("state", node.State).
			AddField("using_wan", node.UsingWan).
			AddField("power_source", node.PowerInfo.PowerSource).
			AddField("connection_type", node.ConnectionType).
			AddField("led_on", node.LedOn).
			SetTime(now)

		p.influx.WritePoint(pt)
	}
}

// writeNetworkHealth writes a single point reflecting the overall network health
// (ISP up status, internet status, and eero mesh status).
func (p *Poller) writeNetworkHealth(net *eero.NetworkDetails) {
	pt := influxdb2.NewPointWithMeasurement("eero_network_health").
		AddTag("network_name", net.Name).
		AddField("isp_up", net.Health.Internet.ISPUp).
		AddField("internet_status", net.Health.Internet.Status).
		AddField("eero_network_status", net.Status).
		SetTime(time.Now())

	p.influx.WritePoint(pt)
}

// ---------------------------------------------------------------------------
// Medium Poll Writers (every 90 minutes)
// ---------------------------------------------------------------------------

// writeNodeMetadata writes slowly-changing node inventory data (IP, MAC, firmware
// version, ethernet addresses, etc.) to InfluxDB.
func (p *Poller) writeNodeMetadata(net *eero.NetworkDetails) {
	now := time.Now()
	for i := range net.Eeros.Data {
		node := &net.Eeros.Data[i]
		nodeName := fmt.Sprintf("%s - %s", node.Location, node.Model)
		pt := influxdb2.NewPointWithMeasurement("eero_node_metadata").
			AddTag("serial", node.Serial).
			AddTag("node_name", nodeName).
			AddField("ip_address", node.IPAddress).
			AddField("mac_address", node.MACAddress).
			AddField("os_version", node.OSVersion).
			AddField("model_number", node.ModelNumber).
			AddField("update_available", node.UpdateAvailable).
			AddField("wired", node.Wired).
			AddField("gateway", node.Gateway).
			AddField("is_primary_node", node.IsPrimaryNode).
			AddField("led_on", node.LedOn).
			AddField("last_heartbeat", node.LastHeartbeat.UnixMilli()).
			AddField("joined", node.Joined.Time.Format(time.RFC3339)).
			AddField("ethernet_addresses", strings.Join(node.EthernetAddresses, ",")).
			AddField("wifi_bssids", strings.Join(node.WifiBSSIDs, ",")).
			AddField("bands", strings.Join(node.Bands, ",")).
			SetTime(now)

		p.influx.WritePoint(pt)
	}
}

// writeClientMetadata writes slowly-changing device metadata (device type, IPs,
// VLAN, manufacturer, etc.) to InfluxDB.
func (p *Poller) writeClientMetadata(devices []eero.Device) {
	now := time.Now()
	for i := range devices {
		d := &devices[i]
		deviceName := resolveDeviceName(d)

		pt := influxdb2.NewPointWithMeasurement("eero_client_metadata").
			AddTag("mac", d.MAC).
			AddTag("device_name", deviceName).
			AddField("device_type", d.DeviceType).
			AddField("ipv4", d.IPv4).
			AddField("is_proxied_node", d.IsProxiedNode).
			AddField("is_private_mac", d.IsPrivate).
			AddField("is_guest", d.IsGuest).
			AddField("blacklisted", d.Blacklisted).
			AddField("paused", d.Paused).
			AddField("auth", d.Auth).
			AddField("ssid", d.SSID).
			AddField("subnet_kind", d.SubnetKind).
			AddField("vlan_name", d.VlanName).
			AddField("first_active", d.FirstActive.Time.Format(time.RFC3339)).
			AddField("last_active", d.LastActive.Time.Format(time.RFC3339)).
			SetTime(now)

		if d.Manufacturer != nil {
			pt.AddField("manufacturer", *d.Manufacturer)
		}
		if d.IP != nil {
			pt.AddField("ip", *d.IP)
		}
		if len(d.IPv6Addresses) > 0 {
			pt.AddField("ipv6", d.IPv6Addresses[0].Address)
		}
		if d.VlanID != nil {
			pt.AddField("vlan_id", *d.VlanID)
		}

		p.influx.WritePoint(pt)
	}
}

// writeProfileMappings writes eero profile-to-device associations and profile
// settings (pause state, app blocking, safe search) to InfluxDB.
func (p *Poller) writeProfileMappings(profiles []eero.Profile) {
	now := time.Now()
	for _, prof := range profiles {
		macs := make([]string, 0, len(prof.Devices))
		for j := range prof.Devices {
			macs = append(macs, prof.Devices[j].MAC)
		}

		pt := influxdb2.NewPointWithMeasurement("eero_profile_mappings").
			AddTag("profile_name", prof.Name).
			AddField("devices", strings.Join(macs, ",")).
			AddField("paused", prof.Paused).
			AddField("block_apps", prof.BlockApps).
			AddField("safe_search_active", prof.SafeSearchActive).
			SetTime(now)

		p.influx.WritePoint(pt)
	}
}

// ---------------------------------------------------------------------------
// Slow Poll Writers (every 12 hours)
// ---------------------------------------------------------------------------

// writeISPSpeeds writes the eero-reported ISP speed test results (download/upload)
// to InfluxDB. These are the speeds eero measures, not real-time throughput.
func (p *Poller) writeISPSpeeds(net *eero.NetworkDetails) {
	pt := influxdb2.NewPointWithMeasurement("eero_isp_speed").
		AddTag("network_name", net.Name).
		AddField("speed_down_mbps", net.Speed.Down.Value).
		AddField("speed_up_mbps", net.Speed.Up.Value).
		SetTime(time.Now())

	p.influx.WritePoint(pt)
}

// parseSignalDBm parses signal strength from the "NN dBm" string format.
func parseSignalDBm(s string) (int, error) {
	if !strings.HasSuffix(s, " dBm") {
		return 0, fmt.Errorf("invalid signal format")
	}
	valStr := strings.TrimSuffix(s, " dBm")
	return strconv.Atoi(valStr)
}

// writeNetworkConfig writes a comprehensive snapshot of the network configuration
// including DNS/DHCP settings, security features, GeoIP data, and firmware status.
func (p *Poller) writeNetworkConfig(net *eero.NetworkDetails) {
	dhcpRouter := ""
	if net.Lease.DHCP != nil {
		dhcpRouter = net.Lease.DHCP.Router
	}

	pt := influxdb2.NewPointWithMeasurement("eero_network_config").
		AddTag("network_name", net.Name).
		AddField("premium_status", net.PremiumStatus).
		AddField("premium_tier", net.PremiumDetails.Tier).
		AddField("dns_policies_enabled", net.PremiumDNS.DNSPoliciesEnabled).
		AddField("ad_block_enabled", net.PremiumDNS.AdBlockSettings.Enabled).
		AddField("block_malware_enabled", net.PremiumDNS.DNSPolicies.BlockMalware).
		AddField("dhcp_mode", net.DHCP.Mode).
		AddField("dhcp_router", dhcpRouter).
		AddField("dns_mode", net.DNS.Mode).
		AddField("dns_caching", net.DNS.Caching).
		AddField("dns_parent_ips", strings.Join(net.DNS.Parent.IPs, ",")).
		AddField("geoip_country", net.GeoIP.CountryName).
		AddField("geoip_region", net.GeoIP.Region).
		AddField("geoip_region_name", net.GeoIP.RegionName).
		AddField("geoip_city", net.GeoIP.City).
		AddField("geoip_timezone", net.GeoIP.Timezone).
		AddField("geoip_isp", net.GeoIP.ISP).
		AddField("geoip_org", net.GeoIP.Org).
		AddField("geoip_asn", net.GeoIP.ASN).
		AddField("wan_type", net.WanType).
		AddField("wireless_mode", net.WirelessMode).
		AddField("mlo_mode", net.MloMode).
		AddField("band_steering", net.BandSteering).
		AddField("wpa3_enabled", net.Wpa3).
		AddField("upnp_enabled", net.UpnpEnabled).
		AddField("ipv6_upstream", net.IPv6Upstream).
		AddField("thread_enabled", net.ThreadEnabled).
		AddField("sqm_enabled", net.SQMEnabled).
		AddField("double_nat", net.IPSettings.DoubleNAT).
		AddField("public_ip", net.IPSettings.PublicIP).
		AddField("guest_network_enabled", net.GuestNetwork.Enabled).
		AddField("guest_network_name", net.GuestNetwork.Name).
		AddField("firmware_has_update", net.Updates.HasUpdate).
		AddField("firmware_target", net.Updates.TargetFirmware).
		AddField("firmware_update_req", net.Updates.UpdateRequired).
		SetTime(time.Now())

	p.influx.WritePoint(pt)
}
