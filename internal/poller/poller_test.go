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
	writeReady chan struct{}
	points     []*write.Point
	once       sync.Once
	mu         sync.Mutex
}

func newMockMetricWriter() *MockMetricWriter {
	return &MockMetricWriter{
		writeReady: make(chan struct{}),
	}
}

func (m *MockMetricWriter) WritePoint(point *write.Point) {
	m.mu.Lock()
	m.points = append(m.points, point)
	m.mu.Unlock()

	// Signal that at least one write has occurred.
	m.once.Do(func() { close(m.writeReady) })
}

func (m *MockMetricWriter) pointCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.points)
}

func TestPoller_Start(t *testing.T) {
	var mu sync.Mutex
	calledNetwork := false
	calledDevices := false
	calledProfiles := false

	mockClient := &MockEeroClient{
		GetNetworkFunc: func(_ context.Context, _ string) (*eero.NetworkDetails, error) {
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
		ListDevicesFunc: func(_ context.Context, _ string) ([]eero.Device, error) {
			mu.Lock()
			calledDevices = true
			mu.Unlock()
			return []eero.Device{}, nil
		},
		ListProfilesFunc: func(_ context.Context, _ string) ([]eero.Profile, error) {
			mu.Lock()
			calledProfiles = true
			mu.Unlock()
			return []eero.Profile{}, nil
		},
	}

	mockWriter := newMockMetricWriter()
	p := NewPoller(mockClient, mockWriter, "/network/123")

	ctx, cancel := context.WithCancel(context.Background())

	// Start poller in a goroutine.
	done := make(chan struct{})
	go func() {
		p.Start(ctx)
		close(done)
	}()

	// Wait for at least one metric write (proves initial poll completed)
	// instead of using time.Sleep, which is flaky on CI.
	select {
	case <-mockWriter.writeReady:
		// Success — initial poll wrote at least one point.
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for initial poll to write metrics")
	}

	cancel()

	select {
	case <-done:
		// Poller shut down cleanly.
	case <-time.After(5 * time.Second):
		t.Fatal("Poller did not stop within timeout")
	}

	mu.Lock()
	defer mu.Unlock()

	if !calledNetwork {
		t.Error("Expected GetNetwork to be called")
	}
	if !calledDevices {
		t.Error("Expected ListDevices to be called")
	}
	if !calledProfiles {
		t.Error("Expected ListProfiles to be called")
	}
	if mockWriter.pointCount() == 0 {
		t.Error("Expected some points to be written")
	}
}
