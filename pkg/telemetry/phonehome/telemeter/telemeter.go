package telemeter

// CallOptions defines optional features for a Telemeter call.
type CallOptions struct {
	UserID     string
	ClientID   string
	ClientType string

	// [group name: group id]
	Groups map[string][]string
	// [group id: [key: value]]
	GroupProperties map[string]map[string]any
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
	}
}

// WithClient allows for modifying the ClientID and ClientType call options.
func WithClient(clientID string, clientType string) Option {
	return func(o *CallOptions) {
		o.ClientID = clientID
		o.ClientType = clientType
	}
}

// WithGroup appends a group association to an event.
func WithGroup(groupName string, groupID string) Option {
	return func(o *CallOptions) {
		if o.Groups == nil {
			o.Groups = make(map[string][]string, 1)
		}
		o.Groups[groupName] = append(o.Groups[groupName], groupID)
	}
}

// WithGroupProperties sets the group properties of an event.
func WithGroupProperties(groupID string, groupProperties map[string]any) Option {
	return func(o *CallOptions) {
		if o.GroupProperties == nil {
			o.GroupProperties = make(map[string]map[string]any)
		}
		o.GroupProperties[groupID] = groupProperties
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
	Group(groupID string, props map[string]any, opts ...Option)
}
