package queue

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
)

// IndicatorQueueItem defines a item for the Queue
type IndicatorQueueItem struct {
	Ctx          context.Context
	Deployment   *storage.Deployment
	Indicator    *storage.ProcessIndicator
	Netpols      *augmentedobjs.NetworkPoliciesApplied
	IsInBaseline bool
}
