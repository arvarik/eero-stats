package poller

import (
	"context"
	"testing"
	"time"

	"github.com/arvarik/eero-go/eero"
)

// Mock client with delay to simulate network latency
type delayedClient struct {
	*MockEeroClient
	delay time.Duration
}

func (c *delayedClient) GetNetwork(ctx context.Context, url string) (*eero.NetworkDetails, error) {
	time.Sleep(c.delay)
	return c.MockEeroClient.GetNetwork(ctx, url)
}

func (c *delayedClient) ListDevices(ctx context.Context, url string) ([]eero.Device, error) {
	time.Sleep(c.delay)
	return c.MockEeroClient.ListDevices(ctx, url)
}

func BenchmarkPollFast(b *testing.B) {
	mockWriter := newMockMetricWriter()
	mockClient := &delayedClient{
		MockEeroClient: &MockEeroClient{
			GetNetworkFunc: func(_ context.Context, _ string) (*eero.NetworkDetails, error) {
				return &eero.NetworkDetails{}, nil
			},
			ListDevicesFunc: func(_ context.Context, _ string) ([]eero.Device, error) {
				return []eero.Device{}, nil
			},
		},
		delay: 10 * time.Millisecond,
	}

	p := NewPoller(mockClient, mockWriter, "/network/123")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.pollFast(ctx)
	}
}
