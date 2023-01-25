package phonehome

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

	// IdentifyUserAs updates the user traits.
	IdentifyUserAs(userID, clientID, clientType string, props map[string]any)
	// TrackUserAs registers an event, caused by the user.
	TrackUserAs(userID, clientID, clientType, event string, props map[string]any)
	// GroupUserAs adds the user to a group, supplying group specific properties.
	GroupUserAs(userID, clientID, clientType, groupID string, props map[string]any)
}

type nilTelemeter struct{}

func (t *nilTelemeter) Stop()                            {}
func (t *nilTelemeter) Identify(_ map[string]any)        {}
func (t *nilTelemeter) Track(_ string, _ map[string]any) {}
func (t *nilTelemeter) Group(_ string, _ map[string]any) {}

func (t *nilTelemeter) IdentifyUserAs(_, _, _ string, _ map[string]any) {}
func (t *nilTelemeter) TrackUserAs(_, _, _, _ string, _ map[string]any) {}
func (t *nilTelemeter) GroupUserAs(_, _, _, _ string, _ map[string]any) {}
