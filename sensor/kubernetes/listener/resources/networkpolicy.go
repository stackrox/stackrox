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
	deplStore *DeploymentStore
	detection Detection
}

func newNetworkPolicyDispatcher(store *NetworkPolicyStore, deplStore *DeploymentStore, detection Detection) *networkPolicyDispatcher {
	return &networkPolicyDispatcher{
		npStore:   store,
		deplStore: deplStore,
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
	log.Infof("networkPolicyDispatcher.ProcessEvent: got %d network polisies in the store.\n", len(h.npStore.GetAll()))
	deployments := h.deplStore.GetAll()

	log.Infof("networkPolicyDispatcher.ProcessEvent: pinging up to %d deployments to refresh.\n", len(deployments))
	for _, deployment := range deployments {
		if deployment.GetNamespace() == netPolicy.GetNamespace() {
			log.Infof("networkPolicyDispatcher.ProcessEvent: triggering detection for depl: %s\n", deployment.Name)
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
