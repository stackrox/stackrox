package message

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

type DeploymentRef struct {
	Namespace, Id string
	Action        central.ResourceAction
	Subject       string
}

type ResourceEvent struct {
	ForwardMessages []*central.SensorEvent

	// DeploymentRefs is an experimental field to provide a new way of resolving
	// deployment dependencies. The objective of this field is to reduce (and eventually remove)
	// the usage of resource re-sync.
	// NOTE: After implementing a prototype of this, I don't thing a slice is a good idea.
	// It makes more sense for the handler that requires multiple deployment updates to send
	// the deployment reference fields. E.g. selectors, subjects etc...
	// The reference can then be a single slice.
	// Additionally, this could be implemented in a way that the reference is parsed by
	// different resolvers which will query the deployment store differently.
	DeploymentRefs []DeploymentRef

	// CompatibilityDetectionDeployment should be used by old handlers
	// and it's here for retrocompatibility reasons.
	// This property should be removed in the future and only the
	// DetectionObject should be sent
	CompatibilityDetectionDeployment []CompatibilityDetectionMessage

	// ReprocessDeployments is also used for compatibility reasons with Network Policy handlers
	// in the future this will not be needed as the dependencies are taken care by the resolvers
	ReprocessDeployments []string
}

type CompatibilityDetectionMessage struct {
	Object *storage.Deployment
	Action central.ResourceAction
}
