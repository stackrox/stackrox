package marketing

// Telemeter defines a common interface for telemetry gatherers.
type Telemeter interface {
	Start()
	Stop()
	Identify(props map[string]any)
	Track(event string)
	TrackProp(event string, key string, value any)
	TrackProps(event string, props map[string]any)
}

// Config represents the central instance telemetry configuration.
type Config struct {
	ID           string
	Orchestrator string
	Version      string
	APIPaths     []string
}
