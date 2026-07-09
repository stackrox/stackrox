package events

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

// ScanResultEvent holds an enriched scan result for deploy-time detection.
type ScanResultEvent struct {
	Context                context.Context
	Action                 central.ResourceAction
	Deployment             *storage.Deployment
	Images                 []*storage.Image
	NetworkPoliciesApplied *augmentedobjs.NetworkPoliciesApplied
}

func (e *ScanResultEvent) Topic() pubsub.Topic {
	return pubsub.DetectorScanResultTopic
}

func (e *ScanResultEvent) Lane() pubsub.LaneID {
	return pubsub.DetectorScanResultLane
}
