package monoclock

import "time"

var (
	instance = New().(monoClock)
)

// MonoClock is a monotonic clock, that only allows for retrieving the duration since some unspecified epoch.
type MonoClock interface {
	SinceEpoch() time.Duration
}

type monoClock struct {
	epoch time.Time
}

// New creates a new monotonic clock.
func New() MonoClock {
	return monoClock{
		epoch: time.Now(),
	}
}

// SinceEpoch returns the elapsed time since this clock's epoch.
func (m monoClock) SinceEpoch() time.Duration {
	return time.Now().Sub(m.epoch)
}

// SinceEpoch returns the elapsed time since the default monotonic clocks epoch.
func SinceEpoch() time.Duration {
	return instance.SinceEpoch()
}
