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

// MetricWriter defines the interface for writing metrics to a destination (e.g., InfluxDB).
type MetricWriter interface {
	WritePoint(point *write.Point)
}
