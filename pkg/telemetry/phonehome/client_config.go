package phonehome

import "time"

// Config represents a telemetry client instance configuration.
type Config struct {
	// ClientID identifies an entity that reports telemetry data.
	ClientID string
	// ClientName tells what kind of client is sending data.
	ClientName string
	// GroupID identifies the main group to which the client belongs.
	GroupID string

	StorageKey   string
	Endpoint     string
	PushInterval time.Duration
}

// Enabled tells whether telemetry data collection is enabled.
func (cfg *Config) Enabled() bool {
	return cfg.StorageKey != ""
}
