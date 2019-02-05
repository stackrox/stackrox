package streamer

import (
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
)

// Pipeline represents a pipeline that reads and processes data from one channel, and outputs to the next.
type Pipeline interface {
	Start(eventsIn <-chan *central.MsgFromSensor, enforcementInjector pipeline.MsgInjector, dependents ...Stoppable)
	Stop(err error) bool
	Stopped() concurrency.ReadOnlyErrorSignal
}

// NewPipeline returns a new instance of a Pipeline using the given processing pipeline.
func NewPipeline(pl pipeline.Pipeline) Pipeline {
	return &channeledImpl{
		pl: pl,

		stopC:    concurrency.NewErrorSignal(),
		stoppedC: concurrency.NewErrorSignal(),
	}
}
