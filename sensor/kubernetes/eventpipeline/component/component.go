package component

import (
	"context"

	"github.com/stackrox/rox/sensor/common"
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
	ResponsesC() <-chan common.ExpiringSensorMessage
}

// Listener component contains all the Kubernetes informers and processes incoming events.
type Listener interface {
	PipelineComponent
	SetContext(ctx context.Context)
}
