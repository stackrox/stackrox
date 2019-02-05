package streamer

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
)

// EnqueueDequeue provides an interface for that pulls from and input channel, and pushes to an output channel, storing
// items that have been pulled but not yet pushed in memory.
type EnqueueDequeue interface {
	Start(inputChannel <-chan *central.MsgFromSensor, dependents ...Stoppable)
	Stop(err error) bool
	Stopped() concurrency.ReadOnlyErrorSignal

	Output() <-chan *central.MsgFromSensor
}

// NewEnqueueDequeue returns a new instance of the EnqueueDequeue interface.
func NewEnqueueDequeue() EnqueueDequeue {
	return &enqueueDequeueImpl{
		queue: newQueue(),

		output: make(chan *central.MsgFromSensor),

		stopC:    concurrency.NewErrorSignal(),
		stoppedC: concurrency.NewErrorSignal(),
	}
}
