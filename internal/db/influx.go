// Package db provides a thin wrapper around the InfluxDB v2 client, configured
// with aggressive write batching to minimize SSD write amplification on NVMe
// storage (ideal for TrueNAS SCALE, Unraid, or Proxmox deployments).
package db

import (
	"context"
	"log/slog"

	"github.com/arvarik/eero-stats/internal/config"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

// InfluxClient wraps the InfluxDB client and its non-blocking write API.
type InfluxClient struct {
	Client   influxdb2.Client
	WriteAPI api.WriteAPI
}

// NewInfluxClient creates an InfluxDB client with NVMe-optimized batching.
// Points are buffered in memory and flushed asynchronously:
//   - BatchSize:     100 points per flush
//   - FlushInterval: 60 seconds
func NewInfluxClient(cfg *config.Config) *InfluxClient {
	options := influxdb2.DefaultOptions().
		SetBatchSize(100).
		SetFlushInterval(60000)

	client := influxdb2.NewClientWithOptions(cfg.InfluxURL, cfg.InfluxToken, options)

	// Use the non-blocking, asynchronous Write API.
	writeAPI := client.WriteAPI(cfg.InfluxOrg, cfg.InfluxBucket)

	// Log async write errors in the background.
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

// Shutdown flushes all buffered writes to disk and closes the InfluxDB
// connection. The provided context can be used for timeout control.
func (i *InfluxClient) Shutdown(ctx context.Context) {
	slog.Info("Flushing InfluxDB memory buffers to disk...")
	i.WriteAPI.Flush()
	i.Client.Close()
	slog.Info("InfluxDB connection closed")
}
