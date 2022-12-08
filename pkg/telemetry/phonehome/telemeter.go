package phonehome

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
