package streamer

import (
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
)

// Pipeline represents a pipeline that reads and processes data from one channel, and outputs to the next.
type Pipeline interface {
	Start(eventsIn <-chan *central.SensorEvent, pl pipeline.Pipeline, enforcementInjector pipeline.EnforcementInjector)
}

// NewPipeline returns a new instance of a Pipeline using the given processing pipeline.
func NewPipeline(onFinish func()) Pipeline {
	return &channeledImpl{
		onFinish: onFinish,
	}
}
