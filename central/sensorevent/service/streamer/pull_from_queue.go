package streamer

import (
	"github.com/stackrox/rox/generated/api/v1"
)

// PullFromQueue provides an interface for pulling from a queue to a channel.
type PullFromQueue interface {
	Start(toPull pullable, outputChannel chan<- *v1.SensorEvent)
}

// NewPullFromQueue returns a new instance of the PullFromQueue interface.
func NewPullFromQueue(onEmpty func() bool, onFinish func()) PullFromQueue {
	return &pullFromQueueImpl{
		onEmpty:  onEmpty,
		onFinish: onFinish,
	}
}

type pullable interface {
	Pull() (*v1.SensorEvent, error)
}
