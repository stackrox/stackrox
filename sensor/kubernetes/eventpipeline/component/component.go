package component

import (
	"context"

	"github.com/stackrox/rox/sensor/common/message"
)

// PipelineComponent components that constitute the eventPipeline
type PipelineComponent interface {
	Start() error
	Stop(error)
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

// ContextListener is a component that listens but has a context in the messages
type ContextListener interface {
	PipelineComponent
	StartWithContext(context.Context) error
}
