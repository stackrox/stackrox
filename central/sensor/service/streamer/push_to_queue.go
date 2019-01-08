package streamer

import (
	"github.com/stackrox/rox/generated/internalapi/central"
)

// PushToQueue provides an interface for push from a channel to a queue.
type PushToQueue interface {
	Start(inputChannel <-chan *central.MsgFromSensor, toPush pushable)
}

// NewPushToQueue returns a new instance of the PullFromQueue interface.
func NewPushToQueue(onPush func(), onFinish func()) PushToQueue {
	return &pushToQueueImpl{
		onPush:   onPush,
		onFinish: onFinish,
	}
}

type pushable interface {
	Push(*central.MsgFromSensor) error
}
