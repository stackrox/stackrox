package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	networkPolicyConversion "github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/store"
	networkingV1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// networkPolicyDispatcher handles network policy resource events.
type networkPolicyDispatcher struct {
	netpolStore     store.NetworkPolicyStore
	deploymentStore *DeploymentStore
	detector        detector.Detector
}

func newNetworkPolicyDispatcher(networkPolicyStore store.NetworkPolicyStore, deploymentStore *DeploymentStore, detector detector.Detector) *networkPolicyDispatcher {
	return &networkPolicyDispatcher{
		netpolStore:     networkPolicyStore,
		deploymentStore: deploymentStore,
		detector:        detector,
	}
}

// ProcessEvent processes a network policy resource event and returns the sensor events to generate.
func (h *networkPolicyDispatcher) ProcessEvent(obj, old interface{}, action central.ResourceAction) []*central.SensorEvent {
	np := obj.(*networkingV1.NetworkPolicy)

	roxNetpol := networkPolicyConversion.KubernetesNetworkPolicyWrap{NetworkPolicy: np}.ToRoxNetworkPolicy()

	if features.NetworkPolicySystemPolicy.Enabled() {
		var roxOldNetpol *storage.NetworkPolicy
		oldNp := old.(*networkingV1.NetworkPolicy)
		if oldNp != nil {
			roxOldNetpol = networkPolicyConversion.KubernetesNetworkPolicyWrap{NetworkPolicy: oldNp}.ToRoxNetworkPolicy()
		}
		sel := h.getSelector(roxNetpol, roxOldNetpol)
		if action == central.ResourceAction_REMOVE_RESOURCE {
			h.netpolStore.Delete(roxNetpol.GetId(), roxNetpol.GetNamespace())
		} else {
			h.netpolStore.Upsert(roxNetpol)
		}

		h.updateDeploymentsFromStore(roxNetpol, sel)
	}

	return []*central.SensorEvent{
		{
			Id:     string(np.UID),
			Action: action,
			Resource: &central.SensorEvent_NetworkPolicy{
				NetworkPolicy: roxNetpol,
			},
		},
	}
}

func (h *networkPolicyDispatcher) getSelector(np, oldNp *storage.NetworkPolicy) selector {
	var sel selector
	if oldNp != nil {
		sel = MatcherOrEverything(oldNp.GetSpec().GetPodSelector().GetMatchLabels())
	} else {
		sel = labels.Nothing()
	}
	return or(sel, MatcherOrEverything(np.GetSpec().GetPodSelector().GetMatchLabels()))
}

func (h *networkPolicyDispatcher) updateDeploymentsFromStore(np *storage.NetworkPolicy, sel selector) {
	for _, deploymentWrap := range h.deploymentStore.getMatchingDeployments(np.GetNamespace(), sel) {
		h.detector.ProcessDeployment(deploymentWrap.GetDeployment(), central.ResourceAction_UPDATE_RESOURCE)
	}
}
