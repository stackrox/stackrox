package telemeter

// Telemeter defines a common interface for telemetry gatherers.
//
//go:generate mockgen-wrapper
type Telemeter interface {
	// Stop gracefully shutdowns the implementation, potentially flushing
	// the buffers.
	Stop()
	// Identify updates the user traits.
	Identify(props map[string]any)
	// Track registers an event, caused by the user.
	Track(event string, props map[string]any)
	// Group adds the user to a group, supplying group specific properties.
	Group(groupID string, props map[string]any)

	With(userID string) Telemeter
	As(clientID string, clientType string) Telemeter
}
