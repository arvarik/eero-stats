1. **Define Constants in `internal/poller/writers.go`**:
   - Create a `const` block at the top of the file to define the string literals used for InfluxDB measurement names.
   - Constants to define:
     - `MeasurementClientTimeSeries` = "eero_client_timeseries"
     - `MeasurementNodeTimeSeries` = "eero_node_timeseries"
     - `MeasurementNetworkHealth` = "eero_network_health"
     - `MeasurementNodeMetadata` = "eero_node_metadata"
     - `MeasurementClientMetadata` = "eero_client_metadata"
     - `MeasurementProfileMappings` = "eero_profile_mappings"
     - `MeasurementISPSpeed` = "eero_isp_speed"
     - `MeasurementNetworkConfig` = "eero_network_config"

2. **Refactor `internal/poller/writers.go`**:
   - Replace all magic strings passed to `influxdb2.NewPointWithMeasurement()` with the corresponding newly defined constants.

3. **Refactor `internal/poller/writers_test.go`**:
   - Replace magic string assertions in `writers_test.go` with the newly defined constants. For example, replacing `"eero_node_timeseries"` with `MeasurementNodeTimeSeries` and `"eero_isp_speed"` with `MeasurementISPSpeed`.

4. **Complete pre-commit steps**:
   - Complete pre-commit steps to ensure proper testing, verification, review, and reflection are done.

5. **Submit the change**:
   - Commit the changes and open a pull request.
