// Package version holds build-time metadata injected via ldflags.
// These variables are set during compilation and default to "dev"
// for local development builds.
package version

// Build-time variables, populated by:
//
//	go build -ldflags "-X .../version.Version=v1.0.0 -X .../version.Commit=abc123 -X .../version.BuildDate=2024-01-01"
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)
