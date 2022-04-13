package pipeline

import (
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/pkg/metrics"
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
	}
	return 0
}
