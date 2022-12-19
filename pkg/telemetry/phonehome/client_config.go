package phonehome

import (
	"time"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/segment"
)

var (
	onceGatherer  sync.Once
	onceTelemeter sync.Once

	log = logging.LoggerForModule()
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

// Gatherer returns the telemetry gatherer instance.
func (cfg *Config) Gatherer() Gatherer {
	if cfg == nil {
		return &nilGatherer{}
	}
	onceGatherer.Do(func() {
		if cfg.Enabled() {
			period := cfg.GatherPeriod
			if cfg.GatherPeriod.Nanoseconds() == 0 {
				period = 1 * time.Hour
			}
			cfg.gatherer = newGatherer(cfg.ClientID, cfg.Telemeter(), period)
		} else {
			cfg.gatherer = &nilGatherer{}
		}
	})
	return cfg.gatherer
}

// Telemeter returns the instance of the telemeter.
func (cfg *Config) Telemeter() Telemeter {
	if cfg == nil {
		return &nilTelemeter{}
	}
	onceTelemeter.Do(func() {
		if cfg.Enabled() {
			cfg.telemeter = segment.NewTelemeter(
				cfg.StorageKey,
				cfg.Endpoint,
				cfg.ClientID,
				cfg.ClientName,
				cfg.PushInterval)
		} else {
			cfg.telemeter = &nilTelemeter{}
		}
	})
	return cfg.telemeter
}
