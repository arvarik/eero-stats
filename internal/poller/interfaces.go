package poller

import (
	"context"

	"github.com/arvarik/eero-go/eero"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

// EeroClient defines the interface for interacting with the Eero API.
type EeroClient interface {
	GetNetwork(ctx context.Context, url string) (*eero.NetworkDetails, error)
	ListDevices(ctx context.Context, url string) ([]eero.Device, error)
	ListProfiles(ctx context.Context, url string) ([]eero.Profile, error)
}

// MetricWriter defines the interface for writing metrics to a time-series
// database (e.g., InfluxDB). The signature matches the InfluxDB non-blocking
// write API (api.WriteAPI) so it can be used directly.
type MetricWriter interface {
	WritePoint(point *write.Point)
}
