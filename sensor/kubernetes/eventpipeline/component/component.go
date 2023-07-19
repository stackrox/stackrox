package component

import (
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
)

// PipelineComponent components that constitute the eventPipeline
type PipelineComponent interface {
	Start() error
	Stop(error)
	common.Notifier
}

// Resolver component that performs the dependency resolution
//
//go:generate mockgen-wrapper
type Resolver interface {
	PipelineComponent
	Send(event *ResourceEvent)
}

// OutputQueue component that redirects Resource Events and Alerts to the output channel
//
//go:generate mockgen-wrapper
type OutputQueue interface {
	PipelineComponent
	Send(event *ResourceEvent)
	ResponsesC() <-chan *message.ExpiringMessage
}
