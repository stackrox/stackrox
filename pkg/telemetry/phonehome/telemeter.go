package phonehome

// Telemeter defines a common interface for telemetry gatherers.
//
//go:generate mockgen-wrapper
type Telemeter interface {
	// Stop gracefully shutdowns the implementation, potentially flushing
	// the buffers.
	Stop()
	// Identify updates the user traits.
	Identify(userID, userKind string, props map[string]any)
	// Track registers an event, caused by the user.
	Track(event, userID string, props map[string]any)
	// Group adds the user to a group, supplying group specific properties.
	Group(groupID, userID string, props map[string]any)
}

type nilTelemeter struct{}

func (t *nilTelemeter) Stop()                                                  {}
func (t *nilTelemeter) Identify(userID, userKind string, props map[string]any) {}
func (t *nilTelemeter) Track(event, userID string, props map[string]any)       {}
func (t *nilTelemeter) Group(groupID, userID string, props map[string]any)     {}
