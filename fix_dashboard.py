import json

with open("grafana/dashboards/eero.json", "r") as f:
    dashboard = json.load(f)

# 1. Variables: Update $Node and $Device
for t in dashboard.get("templating", {}).get("list", []):
    if t["name"] == "Node":
        t["query"] = 'import "influxdata/influxdb/schema"\nschema.measurementTagValues(bucket: "eero", measurement: "eero_node_timeseries", tag: "node_name")'
    elif t["name"] == "Device":
        t["query"] = 'import "influxdata/influxdb/schema"\nschema.measurementTagValues(bucket: "eero", measurement: "eero_client_timeseries", tag: "device_name")'

for panel in dashboard.get("panels", []):
    title = panel.get("title", "")
    
    # Update filters in all panels replacing location and mac with node_name and device_name
    for target in panel.get("targets", []):
        if "query" in target:
            target["query"] = target["query"].replace('r.location =~ /^${Node:regex}$/', 'r.node_name =~ /^${Node:regex}$/')
            target["query"] = target["query"].replace('r.mac == "${Device}"', 'r.device_name == "${Device}"')
            
            # Fix No Data for ISP Status
            if title == "ISP Status":
                target["query"] = 'from(bucket: "eero")\n  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)\n  |> filter(fn: (r) => r._measurement == "eero_network_health" and r._field == "isp_up")\n  |> aggregateWindow(every: v.windowPeriod, fn: last, createEmpty: false)\n  |> yield(name: "last")'
            # Fix No Data for Eero Mesh Status
            elif title == "Eero Mesh Status":
                target["query"] = 'from(bucket: "eero")\n  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)\n  |> filter(fn: (r) => r._measurement == "eero_network_health" and r._field == "eero_network_status")\n  |> aggregateWindow(every: v.windowPeriod, fn: last, createEmpty: false)\n  |> yield(name: "last")'

    # Fix Node Uptime Colors
    if title == "Node Uptime":
        if "fieldConfig" in panel and "defaults" in panel["fieldConfig"]:
            panel["fieldConfig"]["defaults"]["mappings"] = [
                {
                    "options": {
                        "match": "exact",
                        "pattern": "green"
                    },
                    "type": "regex",
                    "text": "Online",
                    "color": "green"
                }
            ]

    # Fix Network Configuration Fields
    elif title == "Network Configuration":
        for target in panel.get("targets", []):
            if "query" in target:
                target["query"] = 'from(bucket: "eero")\n  |> range(start: -24h)\n  |> filter(fn: (r) => r._measurement == "eero_network_config")\n  |> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")\n  |> keep(columns: ["_time", "dhcp_mode", "dns_mode", "geoip_isp", "geoip_city", "geoip_region_name", "geoip_timezone", "geoip_org"])\n  |> last(column: "_time")'

    # Fix Eero Hardware Details Fields
    elif title == "Eero Hardware Details":
        for target in panel.get("targets", []):
            if "query" in target:
                target["query"] = 'from(bucket: "eero")\n  |> range(start: -2h)\n  |> filter(fn: (r) => r._measurement == "eero_node_metadata")\n  |> pivot(rowKey:["serial"], columnKey: ["_field"], valueColumn: "_value")\n  |> keep(columns: ["node_name", "serial", "ip_address", "os_version", "wired", "is_primary_node", "ethernet_addresses"])\n  |> group()'

    # Fix Band Distribution Legend
    elif title == "Band Distribution":
        if "options" not in panel:
            panel["options"] = {}
        if "legend" not in panel["options"]:
            panel["options"]["legend"] = {}
        # standard grafana piechart doesn't easily rename via legend config in pure UI without overrides,
        # but we can do it via renaming in the flux query:
        for target in panel.get("targets", []):
            if "query" in target:
                target["query"] = target["query"] + '\n  |> map(fn: (r) => ({ r with frequency: "Frequency " + r.frequency }))'

    # Fix Device Metadata
    elif title == "Device Metadata":
        for target in panel.get("targets", []):
            if "query" in target:
                target["query"] = 'from(bucket: "eero")\n  |> range(start: -24h)\n  |> filter(fn: (r) => r._measurement == "eero_client_metadata")\n  |> filter(fn: (r) => r.device_name == "${Device}")\n  |> pivot(rowKey:["mac"], columnKey: ["_field"], valueColumn: "_value")\n  |> keep(columns: ["mac", "device_name", "device_type", "first_active", "last_active", "ipv4"])\n  |> last(column: "mac")'


with open("grafana/dashboards/eero.json", "w") as f:
    json.dump(dashboard, f, indent=2)
