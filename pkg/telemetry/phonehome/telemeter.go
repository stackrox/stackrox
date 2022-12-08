package phonehome

import (
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/segment"
)

var (
	onceTelemeter sync.Once
)

// Telemeter defines a common interface for telemetry gatherers.
//go:generate mockgen-wrapper
type Telemeter interface {
	Start()
	Stop()
	Identify(userID string, props map[string]any)
	Track(event, userID string, props map[string]any)
	Group(groupID, userID string, props map[string]any)
}

type nilTelemeter struct{}

func (t *nilTelemeter) Start()                                             {}
func (t *nilTelemeter) Stop()                                              {}
func (t *nilTelemeter) Identify(userID string, props map[string]any)       {}
func (t *nilTelemeter) Track(event, userID string, props map[string]any)   {}
func (t *nilTelemeter) Group(groupID, userID string, props map[string]any) {}

// Telemeter returns the instance of the telemeter.
func (cfg *Config) Telemeter() Telemeter {
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
