package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	networkPolicyConversion "github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	networkingV1 "k8s.io/api/networking/v1"
)

type Detection interface {
	ProcessDeployment(deployment *storage.Deployment, action central.ResourceAction)
}

// networkPolicyDispatcher handles network policy resource events.
type networkPolicyDispatcher struct {
	npStore   *NetworkPolicyStore
	detection Detection
}

func newNetworkPolicyDispatcher(store *NetworkPolicyStore, detection Detection) *networkPolicyDispatcher {
	return &networkPolicyDispatcher{
		npStore:   store,
		detection: detection,
	}
}

// ProcessEvent processes a network policy resource event, and returns the sensor events to generate.
func (h *networkPolicyDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) []*central.SensorEvent {
	np := obj.(*networkingV1.NetworkPolicy)
	netPolicy := networkPolicyConversion.KubernetesNetworkPolicyWrap{NetworkPolicy: np}.ToRoxNetworkPolicy()

	switch action {
	case central.ResourceAction_CREATE_RESOURCE:
		h.npStore.addNetPolicy(netPolicy)
	case central.ResourceAction_REMOVE_RESOURCE:
		h.npStore.deleteNetPolicy(netPolicy)
	case central.ResourceAction_UPDATE_RESOURCE:
		h.npStore.update(netPolicy)
	}

	for _, deployment := range DeploymentStoreSingleton().GetAll() {
		if deployment.Namespace == netPolicy.GetNamespace() {
			h.detection.ProcessDeployment(deployment, action)
		}
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
