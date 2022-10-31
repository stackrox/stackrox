package message

import "github.com/stackrox/rox/generated/internalapi/central"

// PipelineComponent components that constitute the eventPipeline
type PipelineComponent interface {
	Start() error
	Stop(error)
	ResponsesC() <-chan *central.MsgFromSensor
}

// OutputQueue component that redirects Resource Events and Alerts to the output channel
type OutputQueue interface {
	PipelineComponent
	Send(detectionObject *ResourceEvent)
}
