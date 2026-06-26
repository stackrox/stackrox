package events

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

// FileAccessEvent holds the enriched state for a file access event.
type FileAccessEvent struct {
	Ctx        context.Context
	Deployment *storage.Deployment
	Node       *storage.Node
	Access     *storage.FileAccess
	Netpols    *augmentedobjs.NetworkPoliciesApplied
}

func (e *FileAccessEvent) Topic() pubsub.Topic {
	return pubsub.DetectorFileAccessTopic
}

func (e *FileAccessEvent) Lane() pubsub.LaneID {
	return pubsub.DetectorFileAccessLane
}
