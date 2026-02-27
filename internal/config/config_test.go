package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Save and clear environment to avoid leaking real values into tests.
	envKeys := []string{"EERO_LOGIN", "EERO_SESSION_PATH", "INFLUX_URL", "INFLUX_TOKEN", "INFLUX_ORG", "INFLUX_BUCKET"}
	saved := make(map[string]string, len(envKeys))
	for _, k := range envKeys {
		saved[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	t.Cleanup(func() {
		for k, v := range saved {
			if v != "" {
				os.Setenv(k, v)
			}
		}
	})

	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{
			name:    "missing EERO_LOGIN",
			env:     map[string]string{},
			wantErr: "EERO_LOGIN",
		},
		{
			name: "missing INFLUX_URL",
			env: map[string]string{
				"EERO_LOGIN": "test@example.com",
			},
			wantErr: "INFLUX_URL",
		},
		{
			name: "missing INFLUX_TOKEN",
			env: map[string]string{
				"EERO_LOGIN": "test@example.com",
				"INFLUX_URL": "http://localhost:8086",
			},
			wantErr: "INFLUX_TOKEN",
		},
		{
			name: "missing INFLUX_ORG",
			env: map[string]string{
				"EERO_LOGIN":   "test@example.com",
				"INFLUX_URL":   "http://localhost:8086",
				"INFLUX_TOKEN": "token123",
			},
			wantErr: "INFLUX_ORG",
		},
		{
			name: "missing INFLUX_BUCKET",
			env: map[string]string{
				"EERO_LOGIN":   "test@example.com",
				"INFLUX_URL":   "http://localhost:8086",
				"INFLUX_TOKEN": "token123",
				"INFLUX_ORG":   "my-org",
			},
			wantErr: "INFLUX_BUCKET",
		},
		{
			name: "all required vars present",
			env: map[string]string{
				"EERO_LOGIN":    "test@example.com",
				"INFLUX_URL":    "http://localhost:8086",
				"INFLUX_TOKEN":  "token123",
				"INFLUX_ORG":    "my-org",
				"INFLUX_BUCKET": "eero",
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars for this subtest.
			for _, k := range envKeys {
				os.Unsetenv(k)
			}
			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			cfg, err := Load()

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if got := err.Error(); !contains(got, tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.EeroLogin != tt.env["EERO_LOGIN"] {
				t.Errorf("EeroLogin = %q, want %q", cfg.EeroLogin, tt.env["EERO_LOGIN"])
			}
			if cfg.InfluxURL != tt.env["INFLUX_URL"] {
				t.Errorf("InfluxURL = %q, want %q", cfg.InfluxURL, tt.env["INFLUX_URL"])
			}

			// Verify EeroSessionPath defaults correctly.
			expectedSessionPath := tt.env["EERO_SESSION_PATH"]
			if expectedSessionPath == "" {
				expectedSessionPath = "data/app/.eero_session.json"
			}
			if cfg.EeroSessionPath != expectedSessionPath {
				t.Errorf("EeroSessionPath = %q, want %q", cfg.EeroSessionPath, expectedSessionPath)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
