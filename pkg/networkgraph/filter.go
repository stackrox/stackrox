package networkgraph

import "github.com/stackrox/rox/generated/storage"

// FilterFlowsByPredicate applies the given predicate to the slice of flows.
func FilterFlowsByPredicate(flows []*storage.NetworkFlow, pred func(*storage.NetworkFlowProperties) bool) []*storage.NetworkFlow {
	if pred == nil {
		return flows
	}

	filtered := make([]*storage.NetworkFlow, 0, len(flows))
	for _, flow := range flows {
		if !pred(flow.GetProps()) {
			continue
		}
		filtered = append(filtered, flow)
	}
	return filtered
}
