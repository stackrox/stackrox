package telemeter

// CallOptions defines optional features for a Telemeter call.
type CallOptions struct {
	UserID      string
	AnonymousID string
	ClientID    string
	ClientType  string

	// [group type: [group id]]
	Groups map[string][]string
	// User properties to be updated:
	Traits map[string]any
}

// Option modifies the provided CallOptions structure.
type Option func(*CallOptions)

// ApplyOptions returns an instance of CallOptions modified by provided opts.
func ApplyOptions(opts []Option) *CallOptions {
	o := &CallOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// WithUserID allows for modifying the UserID call option.
func WithUserID(userID string) Option {
	return func(o *CallOptions) {
		o.UserID = userID
		o.AnonymousID = ""
	}
}

// WithClient allows for modifying the ClientID and ClientType call options.
func WithClient(clientID string, clientType string) Option {
	return func(o *CallOptions) {
		if o.UserID == "" {
			o.AnonymousID = clientID
		}
		o.ClientID = clientID
		o.ClientType = clientType
	}
}

// WithGroups appends the groups for an event.
func WithGroups(groupType string, groupID string) Option {
	return func(o *CallOptions) {
		if o.Groups == nil {
			o.Groups = make(map[string][]string, 1)
		}
		o.Groups[groupType] = append(o.Groups[groupType], groupID)
	}
}

// WithTraits sets the user properties to be updated with the call.
func WithTraits(traits map[string]any) Option {
	return func(o *CallOptions) {
		o.Traits = traits
	}
}

// Telemeter defines a common interface for telemetry gatherers.
//
//go:generate mockgen-wrapper
type Telemeter interface {
	// Stop gracefully shutdowns the implementation, potentially flushing
	// the buffers.
	Stop()
	// Identify updates user traits.
	Identify(props map[string]any, opts ...Option)
	// Track registers an event, caused by a user.
	Track(event string, props map[string]any, opts ...Option)
	// Group adds a user to a group, supplying group specific properties.
	// The group must be provided with a WithGroups option.
	Group(props map[string]any, opts ...Option)
}
