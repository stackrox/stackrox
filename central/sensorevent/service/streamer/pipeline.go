package streamer

import (
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
	"github.com/stackrox/rox/generated/api/v1"
)

// Pipeline represents a pipeline that reads and processes data from one channel, and outputs to the next.
type Pipeline interface {
	Start(eventsIn <-chan *v1.SensorEvent, pl pipeline.Pipeline, enforcementsOut chan<- *v1.SensorEnforcement)
}

// NewPipeline returns a new instance of a Pipeline using the given processing pipeline.
func NewPipeline(onFinish func()) Pipeline {
	return &channeledImpl{
		onFinish: onFinish,
	}
}
