package marketing

// Telemeter defines a common interface for telemetry gatherers.
type Telemeter interface {
	Start()
	Stop()
	Identify(props map[string]any)
	Track(event, userID string)
	TrackProp(event, userID string, key string, value any)
	TrackProps(event, userID string, props map[string]any)
}

// Config represents the central instance telemetry configuration.
type Config struct {
	ID       string
	APIPaths []string
	Identity map[string]any
}
