package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	networkPolicyConversion "github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	"github.com/stackrox/rox/sensor/common/store"
	networkingV1 "k8s.io/api/networking/v1"
)

// networkPolicyDispatcher handles network policy resource events.
type networkPolicyDispatcher struct {
	store store.NetworkPolicyStore
}

func newNetworkPolicyDispatcher(nps store.NetworkPolicyStore) *networkPolicyDispatcher {
	return &networkPolicyDispatcher{
		store: nps,
	}
}

// ProcessEvent processes a network policy resource event, and returns the sensor events to generate.
func (h *networkPolicyDispatcher) ProcessEvent(newObj, _ interface{}, action central.ResourceAction) []*central.SensorEvent {
	np := newObj.(*networkingV1.NetworkPolicy)
	netPolicy := networkPolicyConversion.KubernetesNetworkPolicyWrap{NetworkPolicy: np}.ToRoxNetworkPolicy()

	switch action {
	case central.ResourceAction_CREATE_RESOURCE, central.ResourceAction_UPDATE_RESOURCE:
		h.store.Upsert(netPolicy)
	case central.ResourceAction_REMOVE_RESOURCE:
		h.store.Delete(netPolicy.GetId(), netPolicy.GetNamespace())
	}

	return []*central.SensorEvent{
		{
			Id:     string(np.UID),
			Action: action,
			Resource: &central.SensorEvent_NetworkPolicy{
				NetworkPolicy: networkPolicyConversion.KubernetesNetworkPolicyWrap{NetworkPolicy: np}.ToRoxNetworkPolicy(),
			},
		},
	}
}
