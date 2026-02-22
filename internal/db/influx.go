package db

import (
	"context"
	"log/slog"

	"github.com/arvarik/eero-stats/internal/config"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

type InfluxClient struct {
	Client   influxdb2.Client
	WriteAPI api.WriteAPI
}

// NewInfluxClient initializes an NVMe-optimized InfluxDB client.
func NewInfluxClient(cfg *config.Config) *InfluxClient {
	// Enforce heavy batching to protect SSD TBW (Write Amplification)
	// BatchSize: 100 points
	// FlushInterval: 60000ms (60 seconds)
	options := influxdb2.DefaultOptions().
		SetBatchSize(100).
		SetFlushInterval(60000)

	client := influxdb2.NewClientWithOptions(cfg.InfluxURL, cfg.InfluxToken, options)

	// Instantiate non-blocking, asynchronous Write API
	writeAPI := client.WriteAPI(cfg.InfluxOrg, cfg.InfluxBucket)

	// Attach background error logging for the async writer
	errorsCh := writeAPI.Errors()
	go func() {
		for err := range errorsCh {
			slog.Error("InfluxDB async write error", "error", err)
		}
	}()

	return &InfluxClient{
		Client:   client,
		WriteAPI: writeAPI,
	}
}

// Shutdown gracefully flushes all memory buffers to disk on exit
func (i *InfluxClient) Shutdown(ctx context.Context) {
	slog.Info("Flushing InfluxDB memory buffers to disk...")
	i.WriteAPI.Flush()
	i.Client.Close()
	slog.Info("InfluxDB connection closed")
}
