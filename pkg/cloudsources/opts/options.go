package opts

import "time"

// Option holds configurable options for cloud source clients.
type Option struct {
	Retries int
	Timeout time.Duration
}

// ClientOpts is the option function for cloud source clients.
type ClientOpts func(o *Option)

// WithRetries sets the number of max retries for a client.
// In case it is set to 0, no retries will be attempted.
func WithRetries(retries int) ClientOpts {
	return func(o *Option) {
		o.Retries = retries
	}
}

// WithTimeout sets the timeout for the client.
func WithTimeout(timeout time.Duration) ClientOpts {
	return func(o *Option) {
		o.Timeout = timeout
	}
}

// DefaultOpts returns the default options used for clients.
func DefaultOpts() *Option {
	return &Option{
		Retries: 3,
		Timeout: 30 * time.Second,
	}
}
