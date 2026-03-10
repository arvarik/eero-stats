package poller

import (
	"context"
	"testing"
	"time"

	"github.com/arvarik/eero-go/eero"
)

func TestPollMedium_Performance(t *testing.T) {
	// Simulate slow API calls
	apiDelay := 50 * time.Millisecond

	mockClient := &MockEeroClient{
		GetNetworkFunc: func(ctx context.Context, url string) (*eero.NetworkDetails, error) {
			time.Sleep(apiDelay)
			return &eero.NetworkDetails{}, nil
		},
		ListDevicesFunc: func(ctx context.Context, url string) ([]eero.Device, error) {
			time.Sleep(apiDelay)
			return []eero.Device{}, nil
		},
		ListProfilesFunc: func(ctx context.Context, url string) ([]eero.Profile, error) {
			time.Sleep(apiDelay)
			return []eero.Profile{}, nil
		},
	}

	mockWriter := newMockMetricWriter()
	p := NewPoller(mockClient, mockWriter, "/network/123")

	ctx := context.Background()

	start := time.Now()
	p.pollMedium(ctx)
	duration := time.Since(start)

	t.Logf("pollMedium took: %v", duration)

	// Since there are 3 API calls taking 50ms each, sequential would take ~150ms.
	// Parallel should take ~50ms.
	if duration > 100*time.Millisecond {
		t.Logf("pollMedium runs sequentially, duration: %v", duration)
	} else {
		t.Logf("pollMedium runs concurrently, duration: %v", duration)
	}
}
