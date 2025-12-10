package telemeter

import "maps"

// CallOptions defines optional features for a Telemeter call.
type CallOptions struct {
	UserID          string
	AnonymousID     string
	ClientID        string
	ClientType      string
	ClientVersion   string
	MessageIDPrefix string

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
func WithClient(clientID string, clientType, clientVersion string) Option {
	return func(o *CallOptions) {
		if o.UserID == "" {
			o.AnonymousID = clientID
		}
		o.ClientID = clientID
		o.ClientType = clientType
		o.ClientVersion = clientVersion
	}
}

// WithGroup appends the provided group to the list of client groups.
func WithGroup(groupType string, groupID string) Option {
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
		if o.Traits == nil {
			o.Traits = traits
		} else {
			maps.Copy(o.Traits, traits)
		}
	}
}

// WithNoDuplicates enables the use of the expiring cache to check for
// previously sent messages.
// The message ID is cached since the first use of WithNoDuplicates, meaning if
// some message has been sent without this option, it won't be considered in the
// consequent call with the option.
// If messageIDPrefix is empty, the option is ignored.
// Check the cache implementation for details for how long the ID is stored.
// Note: on the Segment server side a similar cache is used for deduplication.
func WithNoDuplicates(messageIDPrefix string) Option {
	return func(o *CallOptions) {
		o.MessageIDPrefix = messageIDPrefix
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
	Identify(opts ...Option)
	// Track registers an event, caused by a user.
	Track(event string, props map[string]any, opts ...Option)
	// Group adds a user to a group, supplying group specific traits.
	// The groups must be provided with a WithGroup options.
	Group(opts ...Option)
}
