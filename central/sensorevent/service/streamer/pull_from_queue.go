package streamer

import (
	"github.com/stackrox/rox/generated/internalapi/central"
)

// PullFromQueue provides an interface for pulling from a queue to a channel.
type PullFromQueue interface {
	Start(toPull pullable, outputChannel chan<- *central.SensorEvent)
}

// NewPullFromQueue returns a new instance of the PullFromQueue interface.
func NewPullFromQueue(onEmpty func() bool, onFinish func()) PullFromQueue {
	return &pullFromQueueImpl{
		onEmpty:  onEmpty,
		onFinish: onFinish,
	}
}

type pullable interface {
	Pull() (*central.SensorEvent, error)
}
