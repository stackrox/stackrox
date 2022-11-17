package marketing

// Telemeter defines a common interface for telemetry gatherers.
//go:generate mockgen-wrapper
type Telemeter interface {
	Start()
	Stop()
	Identify(props map[string]any)
	TrackProps(event, userID string, props map[string]any)
}

// Config represents the central instance telemetry configuration.
type Config struct {
	ID       string
	OrgID    string
	APIPaths []string
	Identity map[string]any
}
