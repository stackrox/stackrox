package phonehome

import "github.com/stackrox/rox/pkg/set"

// Telemeter defines a common interface for telemetry gatherers.
//go:generate mockgen-wrapper
type Telemeter interface {
	Start()
	Stop()
	Identify(props map[string]any)
	Track(event, userID string, props map[string]any)
}

// Config represents the central instance telemetry configuration.
type Config struct {
	CentralID string
	// Cloud Services identity properties passed from the fleet manager.
	CSProperties map[string]string
	APIPaths     set.FrozenSet[string]
	Identity     map[string]any
}
