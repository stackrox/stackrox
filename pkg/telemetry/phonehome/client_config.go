package phonehome

import (
	"context"
	"time"
)

// TenantIDLabel is the name of the k8s object label that holds the cloud
// services tenant ID. The value of the label becomes the group ID if not empty.
const TenantIDLabel = "rhacs.redhat.com/tenant"

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

	// The period of identity gathering. Default is 1 hour.
	GatherPeriod time.Duration

	telemeter Telemeter
	gatherer  Gatherer
}

// Enabled tells whether telemetry data collection is enabled.
func (cfg *Config) Enabled() bool {
	return cfg != nil && cfg.StorageKey != ""
}

// GatherFunc returns properties gathered by a data source.
type GatherFunc func(context.Context) (map[string]any, error)
