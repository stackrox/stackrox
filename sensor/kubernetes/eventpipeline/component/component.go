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

// Resolver component that performs the dependency resolution. This is the core component of the pipeline, and the reason
// why re-sync is no longer required in kubernetes informers. It receives inputs from any kubernetes handler (from
// listeners in sensor/kubernetes/listener/resources) in the form of a ResourceEvent object. Its main responsibility
// is to look at the DeploymentReference slice, and process any deployments that need to be updated. Deployment references
// are based on resolver.DeploymentResolution callback functions, which are defined by the handlers.
//
// For example: given deployment-A which has a service account (sa-A), and RBAC resources attached to it. When any of the
// RBAC resources bound to sa-A change (created, updated or deleted), deployment-A has to be reprocessed. This is required
// because the storage.Deployment object contains properties that are based off RBAC resources. They also need reprocessing,
// because the state of violations might change. Rather than reprocessing every deployment every minute, the Resolver component
// will look at any DeploymentReference entries that handlers might have added, and enqueue those deployments for reprocessing.
//
//go:generate mockgen-wrapper
type Resolver interface {
	PipelineComponent
	Send(event *ResourceEvent)
}

// OutputQueue component that redirects Resource Events to the output channel. This component is the last step
// before dispatching Kubernetes events to other components (detector or to the Sensor gRPC connection). It converts component.ResourceEvent
// messages to protobuf messages, and communicates with the detector for any event that requires processing.
//
// Messages written in the .ResponsesC channel here will be copied to the parent component .ResponsesC (which will eventually be picked up by
// the gRPC component)
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
