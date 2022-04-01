package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	networkPolicyConversion "github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/store"
	networkingV1 "k8s.io/api/networking/v1"
)

// networkPolicyDispatcher handles network policy resource events.
type networkPolicyDispatcher struct {
	netpolStore     store.NetworkPolicyStore
	deploymentStore *DeploymentStore
	reconciler      networkPolicyReconciler
	detector        detector.Detector
}

func newNetworkPolicyDispatcher(networkPolicyStore store.NetworkPolicyStore, deploymentStore *DeploymentStore, detector detector.Detector) *networkPolicyDispatcher {
	return &networkPolicyDispatcher{
		netpolStore:     networkPolicyStore,
		deploymentStore: deploymentStore,
		reconciler:      newNetworkPolicyReconciler(deploymentStore, networkPolicyStore),
		detector:        detector,
	}
}

// Process processes a network policy resource event, and returns the sensor events to generate.
func (h *networkPolicyDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) []*central.SensorEvent {
	np := obj.(*networkingV1.NetworkPolicy)

	roxNetpol := networkPolicyConversion.KubernetesNetworkPolicyWrap{NetworkPolicy: np}.ToRoxNetworkPolicy()

	// TODO: feature flag

	//netpolWrap := &networkPolicyWrap{
	//	NetworkPolicy: roxNetpol,
	//	selector:      SelectorFromMap(np.Spec.PodSelector.MatchLabels),
	//}

	var sel selector
	oldWrap := h.netpolStore.Get(roxNetpol.GetId())
	if oldWrap != nil {
		sel = SelectorFromMap(oldWrap.GetSpec().GetPodSelector().GetMatchLabels())
	}

	if action == central.ResourceAction_REMOVE_RESOURCE {
		h.netpolStore.Delete(roxNetpol.GetId(), roxNetpol.GetNamespace())
		h.updateDeploymentsFromStore(roxNetpol, sel, action)
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

	h.netpolStore.Upsert(roxNetpol)
	if action == central.ResourceAction_UPDATE_RESOURCE {
		if sel != nil {
			sel = or(sel, SelectorFromMap(roxNetpol.GetSpec().GetPodSelector().GetMatchLabels()))
		} else {
			sel = SelectorFromMap(roxNetpol.GetSpec().GetPodSelector().GetMatchLabels())
		}
	} else if action == central.ResourceAction_CREATE_RESOURCE {
		sel = SelectorFromMap(roxNetpol.GetSpec().GetPodSelector().GetMatchLabels())
	}
	h.updateDeploymentsFromStore(roxNetpol, sel, action)

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

func (h *networkPolicyDispatcher) updateDeploymentsFromStore(np *storage.NetworkPolicy, sel selector, action central.ResourceAction) {
	for _, deploymentWrap := range h.deploymentStore.getMatchingDeployments(np.GetNamespace(), sel) {
		h.reconciler.UpdateNetworkPolicyForDeployment(deploymentWrap)
		h.detector.ProcessDeployment(deploymentWrap.GetDeployment(), central.ResourceAction_UPDATE_RESOURCE)
	}
}
