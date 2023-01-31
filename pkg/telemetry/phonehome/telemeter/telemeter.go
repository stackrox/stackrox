package telemeter

type CallOptions struct {
	UserID     string
	ClientID   string
	ClientType string
}

type Option func(*CallOptions)

func ApplyOptions(opts []Option) *CallOptions {
	o := &CallOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func WithUserID(userID string) Option {
	return func(o *CallOptions) {
		o.UserID = userID
	}
}

func WithClient(clientID string, clientType string) Option {
	return func(o *CallOptions) {
		o.ClientID = clientID
		o.ClientType = clientType
	}
}

// Telemeter defines a common interface for telemetry gatherers.
//
//go:generate mockgen-wrapper
type Telemeter interface {
	// Stop gracefully shutdowns the implementation, potentially flushing
	// the buffers.
	Stop()
	// Identify updates the user traits.
	Identify(props map[string]any, opts ...Option)
	// Track registers an event, caused by the user.
	Track(event string, props map[string]any, opts ...Option)
	// Group adds the user to a group, supplying group specific properties.
	Group(groupID string, props map[string]any, opts ...Option)
}
