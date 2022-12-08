package phonehome

import "github.com/stackrox/rox/pkg/env"

// Config represents a telemetry client instance configuration.
type Config struct {
	// ClientID identifies an entity that reports telemetry data.
	ClientID string
	// ClientName tells what kind of client is sending data.
	ClientName string
	// GroupID identifies the main group to which the client belongs.
	GroupID string
}

// Enabled tells whether telemetry data collection is enabled.
func Enabled() bool {
	return env.TelemetryStorageKey.Setting() != ""
}
