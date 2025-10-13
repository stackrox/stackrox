package queue

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
)

// FlowQueueItem defines a item for the Queue
type FlowQueueItem struct {
	Ctx        context.Context
	Deployment *storage.Deployment
	Flow       *augmentedobjs.NetworkFlowDetails
	Netpols    *augmentedobjs.NetworkPoliciesApplied
}
