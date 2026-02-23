package poller

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/arvarik/eero-go/eero"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

// MockEeroClient implements EeroClient interface for testing.
type MockEeroClient struct {
	GetNetworkFunc   func(ctx context.Context, url string) (*eero.NetworkDetails, error)
	ListDevicesFunc  func(ctx context.Context, url string) ([]eero.Device, error)
	ListProfilesFunc func(ctx context.Context, url string) ([]eero.Profile, error)
}

func (m *MockEeroClient) GetNetwork(ctx context.Context, url string) (*eero.NetworkDetails, error) {
	if m.GetNetworkFunc != nil {
		return m.GetNetworkFunc(ctx, url)
	}
	return &eero.NetworkDetails{}, nil
}

func (m *MockEeroClient) ListDevices(ctx context.Context, url string) ([]eero.Device, error) {
	if m.ListDevicesFunc != nil {
		return m.ListDevicesFunc(ctx, url)
	}
	return []eero.Device{}, nil
}

func (m *MockEeroClient) ListProfiles(ctx context.Context, url string) ([]eero.Profile, error) {
	if m.ListProfilesFunc != nil {
		return m.ListProfilesFunc(ctx, url)
	}
	return []eero.Profile{}, nil
}

// MockMetricWriter implements MetricWriter interface for testing.
type MockMetricWriter struct {
	WritePointFunc func(point *write.Point)
}

func (m *MockMetricWriter) WritePoint(point *write.Point) {
	if m.WritePointFunc != nil {
		m.WritePointFunc(point)
	}
}

func TestPoller_Start(t *testing.T) {
	var mu sync.Mutex
	calledNetwork := false
	mockClient := &MockEeroClient{
		GetNetworkFunc: func(ctx context.Context, url string) (*eero.NetworkDetails, error) {
			mu.Lock()
			calledNetwork = true
			mu.Unlock()
			return &eero.NetworkDetails{
				Name: "Test Network",
				Health: eero.Health{
					Internet: eero.InternetHealth{
						Status: "connected",
						ISPUp:  true,
					},
				},
			}, nil
		},
	}

	writeCount := 0
	mockWriter := &MockMetricWriter{
		WritePointFunc: func(point *write.Point) {
			mu.Lock()
			writeCount++
			mu.Unlock()
		},
	}

	p := NewPoller(mockClient, mockWriter, "/network/123")

	ctx, cancel := context.WithCancel(context.Background())

	// Start poller in a goroutine
	done := make(chan struct{})
	go func() {
		p.Start(ctx)
		close(done)
	}()

	// Let it run for a bit (enough for initial poll)
	time.Sleep(100 * time.Millisecond)

	cancel()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Poller did not stop within timeout")
	}

	mu.Lock()
	defer mu.Unlock()
	if !calledNetwork {
		t.Error("Expected GetNetwork to be called")
	}
	if writeCount == 0 {
		t.Error("Expected some points to be written")
	}
}
