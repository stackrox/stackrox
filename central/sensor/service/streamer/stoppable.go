package streamer

import (
	"github.com/stackrox/rox/pkg/concurrency"
)

// Stoppable represents an object that can be stopped, and will then stop asynchronously.
type Stoppable interface {
	Stop(err error) bool
	Stopped() concurrency.ReadOnlyErrorSignal
}

// StopAll signals all input stoppables with the input error.
func StopAll(err error, stoppables ...Stoppable) {
	for _, stoppable := range stoppables {
		stoppable.Stop(err)
	}
}
