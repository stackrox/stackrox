package pipeline

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/metrics"
)

// ActionToOperation converts a resource action to a metric op for recording.
func ActionToOperation(action central.ResourceAction) metrics.Op {
	switch action {
	case central.ResourceAction_CREATE_RESOURCE:
		return metrics.Add
	case central.ResourceAction_UPDATE_RESOURCE:
		return metrics.Update
	case central.ResourceAction_REMOVE_RESOURCE:
		return metrics.Remove
	case central.ResourceAction_SYNC_RESOURCE:
		return metrics.Sync
	}
	return 0
}
