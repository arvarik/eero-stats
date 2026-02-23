import json

DS = {"type": "influxdb", "uid": "P951FEA4DE68E13C5"}

# ─────────────────────────────────────────────────────────────────────────────
# Helpers
# ─────────────────────────────────────────────────────────────────────────────

def row(title, y, panel_id, collapsed=False):
    return {"id": panel_id, "type": "row", "title": title,
            "gridPos": {"h": 1, "w": 24, "x": 0, "y": y},
            "collapsed": collapsed, "panels": []}

def stat(title, query, panel_id, x, y, w=6, h=4, mappings=None, thresholds=None, desc=""):
    p = {
        "id": panel_id, "type": "stat", "title": title,
        "description": desc,
        "datasource": DS,
        "fieldConfig": {
            "defaults": {
                "color": {"mode": "thresholds"},
                "mappings": mappings or [],
                "thresholds": thresholds or {"mode": "absolute", "steps": [{"color": "green", "value": None}]},
            },
            "overrides": []
        },
        "gridPos": {"h": h, "w": w, "x": x, "y": y},
        "options": {"colorMode": "background", "graphMode": "none", "justifyMode": "auto",
                    "orientation": "auto", "reduceOptions": {"calc": "lastNotNull", "fields": "", "values": False},
                    "textMode": "auto"},
        "pluginVersion": "10.0.0",
        "targets": [{"datasource": DS, "refId": "A", "query": query}],
    }
    return p

def gauge(title, query, panel_id, x, y, w=6, h=4, unit="", min_val=0, max_val=1000, desc=""):
    return {
        "id": panel_id, "type": "gauge", "title": title,
        "description": desc,
        "datasource": DS,
        "fieldConfig": {
            "defaults": {
                "color": {"mode": "thresholds"},
                "thresholds": {"mode": "absolute",
                               "steps": [{"color": "red", "value": None},
                                         {"color": "yellow", "value": min_val + (max_val-min_val)*0.4},
                                         {"color": "green", "value": min_val + (max_val-min_val)*0.7}]},
                "unit": unit, "min": min_val, "max": max_val,
            }, "overrides": []
        },
        "gridPos": {"h": h, "w": w, "x": x, "y": y},
        "options": {"reduceOptions": {"calc": "lastNotNull", "fields": "", "values": False}},
        "pluginVersion": "10.0.0",
        "targets": [{"datasource": DS, "refId": "A", "query": query}],
    }

def timeseries(title, query, panel_id, x, y, w=24, h=8, unit="", display_name="", desc=""):
    return {
        "id": panel_id, "type": "timeseries", "title": title,
        "description": desc,
        "datasource": DS,
        "fieldConfig": {
            "defaults": {
                "color": {"mode": "palette-classic"},
                "custom": {"drawStyle": "line", "lineInterpolation": "smooth",
                           "lineWidth": 2, "fillOpacity": 15, "gradientMode": "opacity",
                           "showPoints": "auto", "spanNulls": False,
                           "axisLabel": unit,
                           "thresholdsStyle": {"mode": "off"}},
                "displayName": display_name,
                "mappings": [], "thresholds": {"mode": "absolute",
                                               "steps": [{"color": "green", "value": None}]},
                "unit": unit,
            }, "overrides": []
        },
        "gridPos": {"h": h, "w": w, "x": x, "y": y},
        "options": {"legend": {"calcs": [], "displayMode": "list", "placement": "bottom", "showLegend": True},
                    "tooltip": {"mode": "multi", "sort": "none"}},
        "pluginVersion": "10.0.0",
        "targets": [{"datasource": DS, "refId": "A", "query": query}],
    }

def table(title, query, panel_id, x, y, w=24, h=7, desc=""):
    return {
        "id": panel_id, "type": "table", "title": title,
        "description": desc,
        "datasource": DS,
        "fieldConfig": {"defaults": {}, "overrides": []},
        "gridPos": {"h": h, "w": w, "x": x, "y": y},
        "options": {"showHeader": True, "sortBy": []},
        "pluginVersion": "10.0.0",
        "targets": [{"datasource": DS, "refId": "A", "query": query}],
    }

def state_timeline(title, query, panel_id, x, y, w=24, h=10, desc=""):
    return {
        "id": panel_id, "type": "state-timeline", "title": title,
        "description": desc,
        "datasource": DS,
        "fieldConfig": {
            "defaults": {"color": {"mode": "palette-classic"},
                         "custom": {"fillOpacity": 70, "lineWidth": 0},
                         "mappings": [], "thresholds": {"mode": "absolute", "steps": [{"color": "green", "value": None}]}},
            "overrides": []
        },
        "gridPos": {"h": h, "w": w, "x": x, "y": y},
        "options": {"alignValue": "center",
                    "legend": {"displayMode": "list", "placement": "bottom", "showLegend": True},
                    "mergeValues": False, "rowHeight": 0.6, "showValue": "auto",
                    "tooltip": {"mode": "single", "sort": "none"}},
        "pluginVersion": "10.0.0",
        "targets": [{"datasource": DS, "refId": "A", "query": query}],
    }

def piechart(title, query, panel_id, x, y, w=8, h=8, display_name="", desc=""):
    return {
        "id": panel_id, "type": "piechart", "title": title,
        "description": desc,
        "datasource": DS,
        "fieldConfig": {
            "defaults": {"color": {"mode": "palette-classic"},
                         "displayName": display_name,
                         "custom": {}, "mappings": [],
                         "thresholds": {"mode": "absolute", "steps": [{"color": "green", "value": None}]}},
            "overrides": []
        },
        "gridPos": {"h": h, "w": w, "x": x, "y": y},
        "options": {"displayLabels": ["name", "percent"],
                    "legend": {"displayMode": "list", "placement": "bottom", "showLegend": True},
                    "pieType": "donut", "tooltip": {"mode": "single", "sort": "none"}},
        "pluginVersion": "10.0.0",
        "targets": [{"datasource": DS, "refId": "A", "query": query}],
    }

def bar_chart(title, query, panel_id, x, y, w=24, h=8, desc=""):
    return {
        "id": panel_id, "type": "barchart", "title": title,
        "description": desc,
        "datasource": DS,
        "fieldConfig": {
            "defaults": {"color": {"mode": "palette-classic"},
                         "custom": {"fillOpacity": 80, "gradientMode": "none"},
                         "mappings": [], "thresholds": {"mode": "absolute", "steps": [{"color": "green", "value": None}]}},
            "overrides": []
        },
        "gridPos": {"h": h, "w": w, "x": x, "y": y},
        "options": {"barWidth": 0.97, "groupWidth": 0.7, "legend": {"displayMode": "list", "placement": "bottom", "showLegend": True},
                    "orientation": "auto", "tooltip": {"mode": "multi", "sort": "none"},
                    "xTickLabelRotation": 0, "xTickLabelSpacing": 100},
        "pluginVersion": "10.0.0",
        "targets": [{"datasource": DS, "refId": "A", "query": query}],
    }

def bargauge(title, query, panel_id, x, y, w=12, h=6, display_name="", desc=""):
    return {
        "id": panel_id, "type": "bargauge", "title": title,
        "description": desc,
        "datasource": DS,
        "fieldConfig": {
            "defaults": {"color": {"mode": "thresholds"},
                         "displayName": display_name,
                         "thresholds": {"mode": "absolute",
                                        "steps": [{"color": "red", "value": None},
                                                  {"color": "yellow", "value": 3},
                                                  {"color": "green", "value": 4}]},
                         "min": 0, "max": 5},
            "overrides": []
        },
        "gridPos": {"h": h, "w": w, "x": x, "y": y},
        "options": {"displayMode": "lcd", "orientation": "horizontal",
                    "reduceOptions": {"calc": "lastNotNull", "fields": "", "values": False},
                    "showUnfilled": True},
        "pluginVersion": "10.0.0",
        "targets": [{"datasource": DS, "refId": "A", "query": query}],
    }

# ─────────────────────────────────────────────────────────────────────────────
# Flux Queries
# ─────────────────────────────────────────────────────────────────────────────

Q_ISP_STATUS = (
    'from(bucket: "eero")\n'
    '  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_network_health" and r._field == "isp_up")\n'
    '  |> last()\n'
    '  |> map(fn: (r) => ({r with _value: if r._value == true then 1 else 0}))'
)

Q_MESH_STATUS = (
    'from(bucket: "eero")\n'
    '  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_network_health" and r._field == "eero_network_status")\n'
    '  |> last()\n'
    '  |> map(fn: (r) => ({r with _value: if r._value == "connected" then 1 else 0}))'
)

Q_DOUBLE_NAT = (
    'from(bucket: "eero")\n'
    '  |> range(start: -25h)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_network_config" and r._field == "double_nat")\n'
    '  |> last()\n'
    '  |> map(fn: (r) => ({r with _value: if r._value == true then 1 else 0}))'
)

Q_FIRMWARE_ALERT = (
    'from(bucket: "eero")\n'
    '  |> range(start: -25h)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_network_config" and r._field == "firmware_has_update")\n'
    '  |> last()\n'
    '  |> map(fn: (r) => ({r with _value: if r._value == true then 1 else 0}))'
)

Q_ISP_DL = (
    'from(bucket: "eero")\n'
    '  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_isp_speed" and r._field == "speed_down_mbps")\n'
    '  |> last()'
)

Q_ISP_UL = Q_ISP_DL.replace("speed_down_mbps", "speed_up_mbps")

Q_NET_CONFIG_GEO = (
    'from(bucket: "eero")\n'
    '  |> range(start: -25h)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_network_config")\n'
    '  |> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")\n'
    '  |> keep(columns: ["_time", "geoip_city", "geoip_region_name", "geoip_timezone",\n'
    '      "geoip_isp", "geoip_org", "public_ip", "wan_type"])\n'
    '  |> last(column: "_time")'
)

Q_NET_CONFIG_SECURITY = (
    'from(bucket: "eero")\n'
    '  |> range(start: -25h)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_network_config")\n'
    '  |> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")\n'
    '  |> keep(columns: ["_time", "wpa3_enabled", "band_steering", "upnp_enabled",\n'
    '      "double_nat", "wireless_mode", "mlo_mode"])\n'
    '  |> last(column: "_time")'
)

Q_NET_CONFIG_DNS = (
    'from(bucket: "eero")\n'
    '  |> range(start: -25h)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_network_config")\n'
    '  |> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")\n'
    '  |> keep(columns: ["_time", "dns_mode", "dhcp_mode",\n'
    '      "guest_network_name", "premium_status", "premium_tier",\n'
    '      "dns_policies_enabled", "ad_block_enabled"])\n'
    '  |> last(column: "_time")'
)

Q_NODE_CLIENTS = (
    'from(bucket: "eero")\n'
    '  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_node_timeseries" and r._field == "connected_clients_count")\n'
    '  |> filter(fn: (r) => r.node_name =~ /^${Node:regex}$/)\n'
    '  |> group(columns: ["node_name"])\n'
    '  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)\n'
    '  |> yield(name: "mean")'
)

Q_NODE_UPTIME = (
    'from(bucket: "eero")\n'
    '  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_node_timeseries" and r._field == "status")\n'
    '  |> filter(fn: (r) => r.node_name =~ /^${Node:regex}$/)\n'
    '  |> keep(columns: ["_time", "_value", "node_name"])\n'
    '  |> group(columns: ["node_name"])\n'
    '  |> aggregateWindow(every: v.windowPeriod, fn: last, createEmpty: false)\n'
    '  |> yield(name: "last")'
)

Q_MESH_QUALITY = (
    'from(bucket: "eero")\n'
    '  |> range(start: -15m)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_node_timeseries" and r._field == "mesh_quality_bars")\n'
    '  |> filter(fn: (r) => r.node_name =~ /^${Node:regex}$/)\n'
    '  |> last()'
)

Q_HW_DETAILS = (
    'from(bucket: "eero")\n'
    '  |> range(start: -2h)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_node_metadata")\n'
    '  |> pivot(rowKey:["serial"], columnKey: ["_field"], valueColumn: "_value")\n'
    '  |> keep(columns: ["node_name", "serial", "model_number", "ip_address", "os_version",\n'
    '      "wired", "is_primary_node", "update_available", "ethernet_addresses", "joined"])\n'
    '  |> group()'
)

Q_FIRMWARE_STATUS = (
    'from(bucket: "eero")\n'
    '  |> range(start: -2h)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_node_metadata")\n'
    '  |> pivot(rowKey:["serial"], columnKey: ["_field"], valueColumn: "_value")\n'
    '  |> keep(columns: ["node_name", "os_version", "update_available"])\n'
    '  |> group()'
)

# Node Deep Dive
Q_NODE_DIVE_CLIENTS_TS = (
    'from(bucket: "eero")\n'
    '  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_node_timeseries" and r._field == "connected_clients_count")\n'
    '  |> filter(fn: (r) => r.node_name =~ /^${Node:regex}$/)\n'
    '  |> group(columns: ["node_name"])\n'
    '  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)'
)

Q_NODE_DIVE_MESH_TS = (
    'from(bucket: "eero")\n'
    '  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_node_timeseries" and r._field == "mesh_quality_bars")\n'
    '  |> filter(fn: (r) => r.node_name =~ /^${Node:regex}$/)\n'
    '  |> group(columns: ["node_name"])\n'
    '  |> aggregateWindow(every: v.windowPeriod, fn: last, createEmpty: false)'
)

Q_NODE_DIVE_DEVICES_TABLE = (
    'from(bucket: "eero")\n'
    '  |> range(start: -5m)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_client_timeseries" and r._field == "connected" and r._value == true)\n'
    '  |> filter(fn: (r) => r.node_name =~ /^${Node:regex}$/)\n'
    '  |> group(columns: ["device_name"])\n'
    '  |> last()\n'
    '  |> map(fn: (r) => ({_time: r._time, device: r.device_name, mac: r.mac,\n'
    '      connected_to: r.node_name,\n'
    '      band: if r.frequency == "" or r.frequency == "wired" then "Wired" else "Frequency " + r.frequency + " " + r.frequency_unit\n'
    '     }))\n'
    '  |> group()'
)

Q_NODE_DIVE_POWER = (
    'from(bucket: "eero")\n'
    '  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_node_timeseries" and\n'
    '      (r._field == "power_source" or r._field == "connection_type"))\n'
    '  |> filter(fn: (r) => r.node_name =~ /^${Node:regex}$/)\n'
    '  |> keep(columns: ["_time", "_value", "node_name", "_field"])\n'
    '  |> group(columns: ["node_name", "_field"])\n'
    '  |> aggregateWindow(every: v.windowPeriod, fn: last, createEmpty: false)'
)

# Client health
Q_BAND_DIST = (
    'from(bucket: "eero")\n'
    '  |> range(start: -5m)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_client_timeseries" and r._field == "connected" and r._value == true)\n'
    '  |> group(columns: ["frequency", "frequency_unit"])\n'
    '  |> count()\n'
    '  |> map(fn: (r) => ({\n'
    '       r with frequency:\n'
    '         if r.frequency == "" or r.frequency == "wired" then "Wired"\n'
    '         else "Frequency " + r.frequency + " " + r.frequency_unit\n'
    '     }))'
)

Q_BAND_STEERING = (
    'import "date"\n'
    'from(bucket: "eero")\n'
    '  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_client_timeseries" and r._field == "connected" and r._value == true)\n'
    '  |> filter(fn: (r) => r.frequency != "")\n'
    '  |> group(columns: ["frequency", "frequency_unit"])\n'
    '  |> aggregateWindow(every: v.windowPeriod, fn: count, createEmpty: false)\n'
    '  |> map(fn: (r) => ({\n'
    '       r with frequency:\n'
    '         if r.frequency == "wired" then "Wired"\n'
    '         else "Frequency " + r.frequency + " " + r.frequency_unit\n'
    '     }))'
)

Q_PEAK_HOURS = (
    'import "date"\n'
    'import "timezone"\n'
    'option location = timezone.location(name: "America/Los_Angeles")\n'
    'hours12 = ["12 AM","1 AM","2 AM","3 AM","4 AM","5 AM","6 AM","7 AM","8 AM","9 AM","10 AM","11 AM",\n'
    '           "12 PM","1 PM","2 PM","3 PM","4 PM","5 PM","6 PM","7 PM","8 PM","9 PM","10 PM","11 PM"]\n'
    'from(bucket: "eero")\n'
    '  |> range(start: -7d)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_client_timeseries" and r._field == "connected" and r._value == true)\n'
    '  |> group()\n'
    '  |> aggregateWindow(every: 3m, fn: count, createEmpty: false)\n'
    '  |> map(fn: (r) => ({\n'
    '       _time: r._time,\n'
    '       h: date.hour(t: r._time),\n'
    '       _value: r._value\n'
    '     }))\n'
    '  |> group(columns: ["h"])\n'
    '  |> mean()\n'
    '  |> group()\n'
    '  |> map(fn: (r) => ({\n'
    '       sort_key: (if r.h < 10 then "0" else "") + string(v: r.h),\n'
    '       hour: hours12[r.h],\n'
    '       avg_connected_devices: int(v: r._value)\n'
    '     }))\n'
    '  |> sort(columns: ["sort_key"])\n'
    '  |> drop(columns: ["sort_key"])'
)

Q_SIGNAL_HEATMAP = (
    'from(bucket: "eero")\n'
    '  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_client_timeseries" and r._field == "signal")\n'
    '  |> filter(fn: (r) => r.connection_type == "wireless")\n'
    '  |> filter(fn: (r) => r.node_name =~ /^${Node:regex}$/)\n'
    '  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)'
)

Q_GLOBAL_THROUGHPUT = (
    'from(bucket: "eero")\n'
    '  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_client_timeseries" and\n'
    '      (r._field == "rx_rate_bps" or r._field == "tx_rate_bps"))\n'
    '  |> filter(fn: (r) => r.frequency =~ /^${Frequency:regex}$/)\n'
    '  |> group(columns: ["device_name"])\n'
    '  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)\n'
    '  |> yield(name: "mean")'
)

# Device Deep Dive
Q_DEV_ROAMING = (
    'from(bucket: "eero")\n'
    '  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_client_timeseries" and r._field == "signal")\n'
    '  |> filter(fn: (r) => r.device_name =~ /^${Device:regex}$/)\n'
    '  |> keep(columns: ["_time", "_value", "device_name", "node_name"])\n'
    '  |> map(fn: (r) => ({_time: r._time, _value: r.node_name, device_name: r.device_name}))'
)

Q_DEV_SIGNAL = (
    'from(bucket: "eero")\n'
    '  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_client_timeseries" and r._field == "signal")\n'
    '  |> filter(fn: (r) => r.device_name =~ /^${Device:regex}$/)\n'
    '  |> group(columns: ["device_name"])\n'
    '  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)\n'
    '  |> yield(name: "mean")'
)

Q_DEV_META = (
    'from(bucket: "eero")\n'
    '  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_client_timeseries" and r._field == "connected")\n'
    '  |> filter(fn: (r) => r.device_name =~ /^${Device:regex}$/)\n'
    '  |> group(columns: ["device_name"])\n'
    '  |> last()\n'
    '  |> map(fn: (r) => ({\n'
    '       _time: r._time,\n'
    '       device: r.device_name,\n'
    '       mac: r.mac,\n'
    '       connected_to: r.node_name,\n'
    '       connection: r.connection_type,\n'
    '       band: if r.frequency == "" or r.frequency == "wired" then "Wired"\n'
    '             else "Frequency " + r.frequency + " " + r.frequency_unit\n'
    '     }))\n'
    '  |> group()'
)

# Alerts row
Q_STALE_DEVICES = (
    'from(bucket: "eero")\n'
    '  |> range(start: -25h)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_client_timeseries" and r._field == "connected")\n'
    '  |> group(columns: ["device_name"])\n'
    '  |> last()\n'
    '  |> filter(fn: (r) => r._value == false)\n'
    '  |> map(fn: (r) => ({_time: r._time, device: r.device_name, mac: r.mac,\n'
    '      last_seen: string(v: r._time), connected_to: r.node_name}))\n'
    '  |> group()'
)

Q_PAUSED_DEVICES = (
    'from(bucket: "eero")\n'
    '  |> range(start: -25h)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_client_timeseries" and r._field == "paused")\n'
    '  |> group(columns: ["device_name"])\n'
    '  |> last()\n'
    '  |> filter(fn: (r) => r._value == true)\n'
    '  |> map(fn: (r) => ({_time: r._time, device: r.device_name, mac: r.mac}))\n'
    '  |> group()'
)

Q_BLACKLISTED = (
    'from(bucket: "eero")\n'
    '  |> range(start: -25h)\n'
    '  |> filter(fn: (r) => r._measurement == "eero_client_timeseries" and r._field == "blacklisted")\n'
    '  |> group(columns: ["device_name"])\n'
    '  |> last()\n'
    '  |> filter(fn: (r) => r._value == true)\n'
    '  |> map(fn: (r) => ({_time: r._time, device: r.device_name, mac: r.mac}))\n'
    '  |> group()'
)

# ─────────────────────────────────────────────────────────────────────────────
# Build the Dashboard
# ─────────────────────────────────────────────────────────────────────────────

STATUS_MAPPINGS = [
    {"type": "value", "options": {"1": {"color": "green", "index": 0, "text": "✅ OK"},
                                   "0": {"color": "red",   "index": 1, "text": "❌ Down"}}}
]
DOUBLE_NAT_MAPPINGS = [
    {"type": "value", "options": {"1": {"color": "orange", "index": 0, "text": "⚠️ Double NAT"},
                                   "0": {"color": "green",  "index": 1, "text": "✅ Clean"}}}
]
FW_ALERT_MAPPINGS = [
    {"type": "value", "options": {"1": {"color": "orange", "index": 0, "text": "⚠️ Update Available"},
                                   "0": {"color": "green",  "index": 1, "text": "✅ Up to Date"}}}
]
UPTIME_MAPPINGS = [
    {"type": "value", "options": {"green": {"color": "green", "index": 0, "text": "Online"}}},
    {"type": "special", "options": {"match": "null+nan", "result": {"color": "red", "index": 1, "text": "Offline"}}}
]
UPTIME_THRESHOLDS = {"mode": "absolute", "steps": [{"color": "red", "value": None}, {"color": "green", "value": 1}]}

panels = []
pid = 2  # panel id counter

def next_id():
    global pid
    pid += 2
    return pid

# ── 1. EXECUTIVE SUMMARY ──────────────────────────────────────────────────────
y = 0
panels.append(row("📊 Executive Summary", y, next_id()))
y += 1

panels.append(stat("ISP Status", Q_ISP_STATUS, next_id(), 0, y, w=4, h=4,
    mappings=STATUS_MAPPINGS,
    thresholds={"mode": "absolute", "steps": [{"color": "red", "value": None}]},
    desc="Whether the Eero gateway has a working upstream internet connection."))

panels.append(stat("Mesh Status", Q_MESH_STATUS, next_id(), 4, y, w=4, h=4,
    mappings=STATUS_MAPPINGS,
    thresholds={"mode": "absolute", "steps": [{"color": "red", "value": None}]},
    desc="Overall Eero mesh network health status."))

panels.append(stat("Double NAT", Q_DOUBLE_NAT, next_id(), 8, y, w=4, h=4,
    mappings=DOUBLE_NAT_MAPPINGS,
    thresholds={"mode": "absolute", "steps": [{"color": "green", "value": None}]},
    desc="Double NAT detected means your Eero is behind another router. This can cause issues with port forwarding and game hosting."))

panels.append(stat("Firmware Status", Q_FIRMWARE_ALERT, next_id(), 12, y, w=4, h=4,
    mappings=FW_ALERT_MAPPINGS,
    thresholds={"mode": "absolute", "steps": [{"color": "green", "value": None}]},
    desc="Whether any Eero node has a firmware update available."))

panels.append(gauge("ISP Download Speed", Q_ISP_DL, next_id(), 16, y, w=4, h=4,
    unit="Mbps", min_val=0, max_val=1000,
    desc="Last measured ISP download speed from Eero's built-in speed test (not real-time)."))

panels.append(gauge("ISP Upload Speed", Q_ISP_UL, next_id(), 20, y, w=4, h=4,
    unit="Mbps", min_val=0, max_val=500,
    desc="Last measured ISP upload speed from Eero's built-in speed test (not real-time)."))

y += 4

# ── 2. ISP & CONNECTIVITY ─────────────────────────────────────────────────────
panels.append(row("🌐 ISP & Connectivity", y, next_id()))
y += 1

panels.append(table("🌍 Location & ISP", Q_NET_CONFIG_GEO, next_id(), 0, y, w=12, h=5,
    desc="GeoIP location data for the WAN public IP: city, region, ISP, and WAN type."))

panels.append(table("🔒 Security & Wireless", Q_NET_CONFIG_SECURITY, next_id(), 12, y, w=12, h=5,
    desc="Wireless security settings: WPA3, band steering, UPnP, NAT, and wireless mode."))
y += 5

panels.append(table("🔧 DNS, DHCP & Services", Q_NET_CONFIG_DNS, next_id(), 0, y, w=24, h=5,
    desc="DNS/DHCP configuration, guest network, Eero Secure status, and ad blocking settings."))
y += 5

# ── 3. MESH NODE HEALTH ───────────────────────────────────────────────────────
panels.append(row("📡 Eero Node Telemetry ($Node)", y, next_id()))
y += 1

panels.append(timeseries("Connected Clients per Node", Q_NODE_CLIENTS, next_id(), 0, y, w=12, h=8,
    display_name="${__field.labels.node_name}",
    desc="Number of Wi-Fi clients currently associated to each Eero node. Useful for spotting imbalanced load across your mesh."))

p_uptime = state_timeline("Node Uptime", Q_NODE_UPTIME, next_id(), 12, y, w=12, h=8,
    desc="Per-node online/offline status over time. 'green' from the eero API means fully operational.")

# Fix uptime mappings + displayName + suppress bar text
p_uptime["fieldConfig"]["defaults"]["mappings"] = UPTIME_MAPPINGS
p_uptime["fieldConfig"]["defaults"]["thresholds"] = UPTIME_THRESHOLDS
p_uptime["fieldConfig"]["defaults"]["displayName"] = "${__field.labels.node_name}"
p_uptime["options"]["showValue"] = "never"
panels.append(p_uptime)
y += 8

panels.append(bargauge("Mesh Backhaul Quality", Q_MESH_QUALITY, next_id(), 0, y, w=12, h=6,
    display_name="${__field.labels.node_name}",
    desc="Quality of wireless backhaul link between each node and the gateway (1–5 bars). 5=excellent, 1=critical. N/A for wired backhaul nodes."))

panels.append(table("Eero Hardware Details", Q_HW_DETAILS, next_id(), 12, y, w=12, h=6,
    desc="Static inventory: serial, model, IP, firmware version, and whether each node is the primary gateway."))
y += 6

panels.append(table("Firmware Status", Q_FIRMWARE_STATUS, next_id(), 0, y, w=24, h=5,
    desc="Current firmware on each node. 'update_available = true' rows indicate a pending update."))
y += 5

# ── 4. NODE DEEP DIVE ─────────────────────────────────────────────────────────
panels.append(row("🔬 Node Deep-Dive ($Node)", y, next_id()))
y += 1

panels.append(timeseries("Clients on Node (History)", Q_NODE_DIVE_CLIENTS_TS, next_id(), 0, y, w=12, h=8,
    display_name="${__field.labels.node_name}",
    desc="Historical client count for the selected node over the chosen time range."))

panels.append(timeseries("Mesh Quality History", Q_NODE_DIVE_MESH_TS, next_id(), 12, y, w=12, h=8,
    display_name="${__field.labels.node_name}",
    desc="Mesh backhaul quality bars (1–5) over time for the selected node. Drops may indicate interference or physical obstructions."))
y += 8

panels.append(table("Currently Connected Devices", Q_NODE_DIVE_DEVICES_TABLE, next_id(), 0, y, w=14, h=8,
    desc="Devices currently associated to the selected node, with their MAC address and Wi-Fi band."))

p_power = state_timeline("Node Power & Backhaul", Q_NODE_DIVE_POWER, next_id(), 14, y, w=10, h=8,
    desc="Power source (USB, PoE) and backhaul connection type (wired/wireless) for the selected node over time.")
p_power["fieldConfig"]["defaults"]["displayName"] = "${__field.labels.node_name} ${__field.labels._field}"
p_power["options"]["showValue"] = "never"
panels.append(p_power)
y += 8

# ── 5. CLIENT DEVICE OVERVIEW ─────────────────────────────────────────────────
panels.append(row("📱 Client Device Health ($Frequency)", y, next_id()))
y += 1

panels.append(piechart("Band Distribution", Q_BAND_DIST, next_id(), 0, y, w=6, h=8,
    display_name="${__field.labels.frequency}",
    desc="Breakdown of currently connected clients by Wi-Fi band. Wired = ethernet-connected clients."))

panels.append(timeseries("Band Steering Efficiency", Q_BAND_STEERING, next_id(), 6, y, w=10, h=8,
    desc="Number of connected devices on each frequency band over time. Shows how well devices are distributed across 2.4/5/6 GHz.",
    display_name="${__field.labels.frequency}"))

panels.append(table("Peak Hours (7-day avg)", Q_PEAK_HOURS, next_id(), 16, y, w=8, h=8,
    desc="Average number of devices connected per hour of day over the last 7 days. Useful for spotting peak network usage patterns."))
y += 8

panels.append(timeseries("Network-Wide Signal Strength", Q_SIGNAL_HEATMAP, next_id(), 0, y, w=24, h=16,
    display_name="${__field.labels.device_name} → ${__field.labels.node_name} (${__field.labels.connection_type})",
    unit="dBm",
    desc="RSSI (dBm) for every wireless client. Closer to 0 is better; below -75 dBm is poor."))
y += 16

gtp = timeseries("Global Throughput Rates ($Frequency)", Q_GLOBAL_THROUGHPUT, next_id(), 0, y, w=24, h=16,
    display_name="${__field.labels.device_name}",
    unit="bps",
    desc="⚠️ Negotiated Wi-Fi link (PHY) rates — NOT real-time throughput. Shows the max speed the radio association is capable of. A device shows a high rate even when idle.")
panels.append(gtp)
y += 16

# ── 6. DEVICE DEEP DIVE ───────────────────────────────────────────────────────
panels.append(row("🕵️ Specific Device Deep-Dive ($Device)", y, next_id()))
y += 1

panels.append(state_timeline("AP Roaming Events", Q_DEV_ROAMING, next_id(), 0, y, w=24, h=20,
    desc="Which Eero node the selected device is associated with over time. Changes indicate a roaming event."))
y += 20

panels.append(timeseries("Device Signal Strength", Q_DEV_SIGNAL, next_id(), 0, y, w=24, h=12,
    display_name="${__field.labels.device_name}",
    unit="dBm",
    desc="RSSI (dBm) for the selected device over time. Closer to 0 is better; below -75 dBm is poor."))
y += 12

panels.append(table("Device Metadata", Q_DEV_META, next_id(), 0, y, w=24, h=8,
    desc="Current connection snapshot for the selected device: connected node, MAC, frequency band, and connection type."))
y += 8

# ── 7. ALERTS & ANOMALIES ─────────────────────────────────────────────────────
panels.append(row("⚠️ Alerts & Anomalies", y, next_id()))
y += 1

panels.append(table("Offline / Stale Devices (>24h)", Q_STALE_DEVICES, next_id(), 0, y, w=24, h=8,
    desc="Devices that were last seen connected more than 24 hours ago. Useful for tracking unplugged or forgotten devices."))
y += 8

panels.append(table("Paused Devices", Q_PAUSED_DEVICES, next_id(), 0, y, w=12, h=8,
    desc="Devices currently paused by an Eero profile (parental controls or manual pause)."))

panels.append(table("Blocked Devices", Q_BLACKLISTED, next_id(), 12, y, w=12, h=8,
    desc="Devices that have been blocked/blacklisted from the network."))
y += 8

# ─────────────────────────────────────────────────────────────────────────────
# Variables
# ─────────────────────────────────────────────────────────────────────────────

variables = [
    {
        "current": {"selected": True, "text": ["All"], "value": ["$__all"]},
        "datasource": DS,
        "hide": 0, "includeAll": True, "multi": True, "name": "Node",
        "options": [], "query": 'import "influxdata/influxdb/schema"\nschema.measurementTagValues(bucket: "eero", measurement: "eero_node_timeseries", tag: "node_name")',
        "refresh": 1, "regex": "", "skipUrlSync": False, "sort": 1, "type": "query"
    },
    {
        "current": {"selected": True, "text": ["All"], "value": ["$__all"]},
        "datasource": DS,
        "hide": 0, "includeAll": True, "multi": True, "name": "Device",
        "options": [], "query": 'import "influxdata/influxdb/schema"\nschema.measurementTagValues(bucket: "eero", measurement: "eero_client_timeseries", tag: "device_name")',
        "refresh": 1, "regex": "", "skipUrlSync": False, "sort": 1, "type": "query"
    },
    {
        "current": {"selected": True, "text": ["All"], "value": ["$__all"]},
        "hide": 0, "includeAll": True, "multi": True, "name": "Frequency",
        "options": [{"selected": True, "text": "All", "value": "$__all"},
                    {"selected": False, "text": "2.4", "value": "2.4"},
                    {"selected": False, "text": "5", "value": "5"},
                    {"selected": False, "text": "6", "value": "6"},
                    {"selected": False, "text": "wired", "value": "wired"}],
        "query": "2.4, 5, 6, wired", "skipUrlSync": False, "type": "custom"
    }
]

# ─────────────────────────────────────────────────────────────────────────────
# Assemble Dashboard
# ─────────────────────────────────────────────────────────────────────────────

dashboard = {
    "annotations": {"list": []},
    "editable": True,
    "fiscalYearStartMonth": 0,
    "graphTooltip": 1,  # shared crosshair
    "id": None,
    "links": [],
    "panels": panels,
    "refresh": "1m",
    "schemaVersion": 38,
    "style": "dark",
    "tags": ["eero"],
    "templating": {"list": variables},
    "time": {"from": "now-6h", "to": "now"},
    "timepicker": {},
    "timezone": "",
    "title": "Eero Network Telemetry",
    "uid": "eero_dashboard_v1",
    "version": 1,
    "weekStart": ""
}

with open("grafana/dashboards/eero.json", "w") as f:
    json.dump(dashboard, f, indent=2)

print(f"Dashboard written with {len(panels)} panels across 7 sections (y_max={y}).")
