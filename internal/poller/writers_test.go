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
			got := resolveDeviceName(tt.device)
			if got != tt.want {
				t.Errorf("resolveDeviceName() = %q, want %q", got, tt.want)
			}
		})
	}
}
