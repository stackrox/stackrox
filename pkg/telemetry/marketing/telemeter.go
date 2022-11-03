package marketing

import (
	"github.com/stackrox/rox/pkg/env"
)

// Telemeter defines a common interface for telemetry gatherers.
type Telemeter interface {
	Start()
	Stop()
	Identify(props map[string]any)
	Track(userAgent, event string)
	TrackProp(userAgent, event string, key string, value any)
	TrackProps(userAgent, event string, props map[string]any)
}

// Enabled tells whether telemetry data collection is enabled.
func Enabled() bool {
	return env.AmplitudeAPIKey.Setting() != ""
}

// Config represents the central instance telemetry configuration.
type Config struct {
	ID       string
	Version  string
	APIPaths []string
}
