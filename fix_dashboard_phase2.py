import json
import logging

logging.basicConfig(level=logging.INFO)

with open("grafana/dashboards/eero.json", "r") as f:
    dashboard = json.load(f)

for panel in dashboard.get("panels", []):
    title = panel.get("title", "")
    
    # 1. Global Throughput Rates ($Frequency)
    if title.startswith("Global Throughput Rates"):
        for target in panel.get("targets", []):
            if "query" in target:
                q = target["query"]
                if 'group(columns: ["device_name"])' not in q:
                    target["query"] = q.replace(
                        '|> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)',
                        '|> group(columns: ["device_name"])\n  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)'
                    )

    # 2. Device State
    elif title == "Device State":
        for target in panel.get("targets", []):
            if "query" in target:
                target["query"] = 'from(bucket: "eero")\n  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)\n  |> filter(fn: (r) => r._measurement == "eero_client_timeseries" and r._field == "connected")\n  |> filter(fn: (r) => r.device_name == "${Device}")\n  |> aggregateWindow(every: v.windowPeriod, fn: last, createEmpty: false)\n  |> yield(name: "last")'

    # 3. AP Roaming Events
    elif title == "AP Roaming Events":
        for target in panel.get("targets", []):
            if "query" in target:
                target["query"] = 'from(bucket: "eero")\n  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)\n  |> filter(fn: (r) => r._measurement == "eero_client_timeseries" and r._field == "signal")\n  |> filter(fn: (r) => r.device_name == "${Device}")\n  |> map(fn: (r) => ({ _time: r._time, _value: r.node_name }))'

    # 4. Device Signal Strength
    elif title == "Device Signal Strength":
        for target in panel.get("targets", []):
            if "query" in target:
                q = target["query"]
                if 'group(columns: ["device_name"])' not in q:
                    target["query"] = q.replace(
                        '|> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)',
                        '|> group(columns: ["device_name"])\n  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)'
                    )

    # 5. Negotiated Link Rate
    elif title == "Negotiated Link Rate":
        for target in panel.get("targets", []):
            if "query" in target:
                q = target["query"]
                if 'group(columns: ["_field"])' not in q:
                    target["query"] = q.replace(
                        '|> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)',
                        '|> group(columns: ["_field"])\n  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)'
                    )

    # 6. Device Metadata
    elif title == "Device Metadata":
        for target in panel.get("targets", []):
            if "query" in target:
                target["query"] = 'from(bucket: "eero")\n  |> range(start: -24h)\n  |> filter(fn: (r) => r._measurement == "eero_client_timeseries" and r._field == "connected")\n  |> filter(fn: (r) => r.device_name == "${Device}")\n  |> group()\n  |> keep(columns: ["_time", "device_name", "mac", "node_name", "connection_type", "frequency", "frequency_unit"])\n  |> last(column: "_time")'

with open("grafana/dashboards/eero.json", "w") as f:
    json.dump(dashboard, f, indent=2)

logging.info("Dashboard fixed successfully.")
