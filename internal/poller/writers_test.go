package poller

import (
	"testing"

	"github.com/arvarik/eero-go/eero"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

func strPtr(s string) *string { return &s }

func TestResolveDeviceName(t *testing.T) {
	tests := []struct {
		name   string
		device eero.Device
		want   string
	}{
		{
			name: "prefer nickname",
			device: eero.Device{
				MAC:      "AA:BB:CC:DD:EE:FF",
				Nickname: strPtr("Living Room TV"),
				Hostname: strPtr("LG-TV"),
			},
			want: "Living Room TV",
		},
		{
			name: "fallback to hostname when no nickname",
			device: eero.Device{
				MAC:      "AA:BB:CC:DD:EE:FF",
				Nickname: nil,
				Hostname: strPtr("LG-TV"),
			},
			want: "LG-TV",
		},
		{
			name: "fallback to hostname when nickname is empty",
			device: eero.Device{
				MAC:      "AA:BB:CC:DD:EE:FF",
				Nickname: strPtr(""),
				Hostname: strPtr("LG-TV"),
			},
			want: "LG-TV",
		},
		{
			name: "fallback to MAC when both nil",
			device: eero.Device{
				MAC:      "AA:BB:CC:DD:EE:FF",
				Nickname: nil,
				Hostname: nil,
			},
			want: "AA:BB:CC:DD:EE:FF",
		},
		{
			name: "fallback to MAC when both empty",
			device: eero.Device{
				MAC:      "AA:BB:CC:DD:EE:FF",
				Nickname: strPtr(""),
				Hostname: strPtr(""),
			},
			want: "AA:BB:CC:DD:EE:FF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveDeviceName(&tt.device)
			if got != tt.want {
				t.Errorf("resolveDeviceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWriteISPSpeeds(t *testing.T) {
	mockWriter := newMockMetricWriter()
	p := &Poller{influx: mockWriter}

	net := &eero.NetworkDetails{
		Name: "Test Network",
		Speed: eero.NetworkSpeed{
			Down: eero.SpeedInfo{Value: 100.5},
			Up:   eero.SpeedInfo{Value: 50.2},
		},
	}

	p.writeISPSpeeds(net)

	if mockWriter.pointCount() != 1 {
		t.Fatalf("expected 1 point, got %d", mockWriter.pointCount())
	}

	pt := mockWriter.points[0]
	if pt.Name() != "eero_isp_speed" {
		t.Errorf("expected measurement eero_isp_speed, got %s", pt.Name())
	}

	// Verify tags
	expectedTags := map[string]string{
		"network_name": "Test Network",
	}
	for k, v := range expectedTags {
		found := false
		for _, tag := range pt.TagList() {
			if tag.Key == k && tag.Value == v {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing or incorrect tag %s", k)
		}
	}

	// Verify fields
	expectedFields := map[string]interface{}{
		"speed_down_mbps": 100.5,
		"speed_up_mbps":   50.2,
	}
	for k, v := range expectedFields {
		found := false
		for _, field := range pt.FieldList() {
			if field.Key == k && field.Value == v {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing or incorrect field %s", k)
		}
	}
}
