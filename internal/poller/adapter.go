package poller

import (
	"context"

	"github.com/arvarik/eero-go/eero"
)

// EeroClientAdapter adapts *eero.Client to the EeroClient interface,
// keeping the concrete dependency at the application boundary.
type EeroClientAdapter struct {
	client *eero.Client
}

// NewEeroClientAdapter creates a new adapter for the given eero client.
func NewEeroClientAdapter(client *eero.Client) *EeroClientAdapter {
	return &EeroClientAdapter{client: client}
}

// GetNetwork fetches network details.
func (a *EeroClientAdapter) GetNetwork(ctx context.Context, url string) (*eero.NetworkDetails, error) {
	return a.client.Network.Get(ctx, url)
}

// ListDevices lists devices on the network.
func (a *EeroClientAdapter) ListDevices(ctx context.Context, url string) ([]eero.Device, error) {
	return a.client.Device.List(ctx, url)
}

// ListProfiles lists profiles on the network.
func (a *EeroClientAdapter) ListProfiles(ctx context.Context, url string) ([]eero.Profile, error) {
	return a.client.Profile.List(ctx, url)
}
