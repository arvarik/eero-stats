package poller

import (
	"testing"

	"github.com/arvarik/eero-go/eero"
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

func TestParseSignalDBm(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{
			name:    "valid negative signal",
			input:   "-65 dBm",
			want:    -65,
			wantErr: false,
		},
		{
			name:    "valid positive signal",
			input:   "20 dBm",
			want:    20,
			wantErr: false,
		},
		{
			name:    "missing space",
			input:   "-65dBm",
			want:    0,
			wantErr: true,
		},
		{
			name:    "missing suffix",
			input:   "-65",
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid number",
			input:   "abc dBm",
			want:    0,
			wantErr: true,
		},
		{
			name:    "only suffix",
			input:   " dBm",
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSignalDBm(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSignalDBm() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseSignalDBm() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWriteNodeTimeSeries(t *testing.T) {
	mockWriter := newMockMetricWriter()
	p := &Poller{influx: mockWriter}

	net := &eero.NetworkDetails{
		Eeros: eero.NetworkEeros{
			Data: []eero.EeroNode{
				{
					Serial:                "A123",
					Location:              "Living Room",
					Model:                 "eero Pro 6",
					ConnectedClientsCount: 5,
					MeshQualityBars:       4,
					HeartbeatOK:           true,
					Status:                "online",
					State:                 "connected",
					UsingWan:              true,
					PowerInfo: eero.PowerInfo{
						PowerSource: "ac",
					},
					ConnectionType: "wired",
					LedOn:          true,
				},
				{
					Serial:                "B456",
					Location:              "Bedroom",
					Model:                 "eero 6",
					ConnectedClientsCount: 2,
					MeshQualityBars:       3,
					HeartbeatOK:           false,
					Status:                "offline",
					State:                 "disconnected",
					UsingWan:              false,
					PowerInfo: eero.PowerInfo{
						PowerSource: "battery",
					},
					ConnectionType: "wireless",
					LedOn:          false,
				},
			},
		},
	}

	p.writeNodeTimeSeries(net)

	if mockWriter.pointCount() != 2 {
		t.Fatalf("expected 2 points, got %d", mockWriter.pointCount())
	}

	// Verify Node 1
	pt1 := mockWriter.points[0]
	if pt1.Name() != MeasurementNodeTimeSeries {
		t.Errorf("expected measurement %s, got %s", MeasurementNodeTimeSeries, pt1.Name())
	}

	expectedTags1 := map[string]string{
		"serial":    "A123",
		"location":  "Living Room",
		"model":     "eero Pro 6",
		"node_name": "Living Room - eero Pro 6",
	}
	for k, v := range expectedTags1 {
		found := false
		for _, tag := range pt1.TagList() {
			if tag.Key == k && tag.Value == v {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing or incorrect tag %s in pt1", k)
		}
	}

	expectedFields1 := map[string]interface{}{
		"connected_clients_count": int64(5),
		"mesh_quality_bars":       int64(4),
		"heartbeat_ok":            true,
		"status":                  "online",
		"state":                   "connected",
		"using_wan":               true,
		"power_source":            "ac",
		"connection_type":         "wired",
		"led_on":                  true,
	}
	for k, v := range expectedFields1 {
		found := false
		for _, field := range pt1.FieldList() {
			if field.Key == k && field.Value == v {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing or incorrect field %s in pt1", k)
		}
	}

	// Verify Node 2
	pt2 := mockWriter.points[1]
	if pt2.Name() != MeasurementNodeTimeSeries {
		t.Errorf("expected measurement %s, got %s", MeasurementNodeTimeSeries, pt2.Name())
	}

	expectedTags2 := map[string]string{
		"serial":    "B456",
		"location":  "Bedroom",
		"model":     "eero 6",
		"node_name": "Bedroom - eero 6",
	}
	for k, v := range expectedTags2 {
		found := false
		for _, tag := range pt2.TagList() {
			if tag.Key == k && tag.Value == v {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing or incorrect tag %s in pt2", k)
		}
	}

	expectedFields2 := map[string]interface{}{
		"connected_clients_count": int64(2),
		"mesh_quality_bars":       int64(3),
		"heartbeat_ok":            false,
		"status":                  "offline",
		"state":                   "disconnected",
		"using_wan":               false,
		"power_source":            "battery",
		"connection_type":         "wireless",
		"led_on":                  false,
	}
	for k, v := range expectedFields2 {
		found := false
		for _, field := range pt2.FieldList() {
			if field.Key == k && field.Value == v {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing or incorrect field %s in pt2", k)
		}
	}
}

func TestWriteISPSpeeds(t *testing.T) {
	mockWriter := newMockMetricWriter()
	p := &Poller{influx: mockWriter}

	net := &eero.NetworkDetails{
		Name: "Test Network",
	}
	net.Speed.Down.Value = 100.5
	net.Speed.Up.Value = 50.2

	p.writeISPSpeeds(net)

	if mockWriter.pointCount() != 1 {
		t.Fatalf("expected 1 point, got %d", mockWriter.pointCount())
	}

	pt := mockWriter.points[0]
	if pt.Name() != MeasurementISPSpeed {
		t.Errorf("expected measurement %s, got %s", MeasurementISPSpeed, pt.Name())
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
