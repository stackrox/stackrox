package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	networkPolicyConversion "github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/store"
	networkingV1 "k8s.io/api/networking/v1"
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

// Process processes a network policy resource event, and returns the sensor events to generate.
func (h *networkPolicyDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) []*central.SensorEvent {
	np := obj.(*networkingV1.NetworkPolicy)

	roxNetpol := networkPolicyConversion.KubernetesNetworkPolicyWrap{NetworkPolicy: np}.ToRoxNetworkPolicy()

	if features.NetworkPolicySystemPolicy.Enabled() {
		sel, isEmpty := h.getSelector(roxNetpol, action)
		if action == central.ResourceAction_REMOVE_RESOURCE {
			h.netpolStore.Delete(roxNetpol.GetId(), roxNetpol.GetNamespace())
		} else {
			h.netpolStore.Upsert(roxNetpol)
		}

		h.updateDeploymentsFromStore(roxNetpol, sel, isEmpty)

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

func (h *networkPolicyDispatcher) getSelector(np *storage.NetworkPolicy, action central.ResourceAction) (selector, bool) {
	var sel selector
	isEmpty := true
	oldWrap := h.netpolStore.Get(np.GetId())
	if oldWrap != nil {
		sel = SelectorFromMap(oldWrap.GetSpec().GetPodSelector().GetMatchLabels())
		isEmpty = len(oldWrap.GetSpec().GetPodSelector().GetMatchLabels()) == 0
	}

	if action == central.ResourceAction_UPDATE_RESOURCE {
		if sel != nil {
			sel = or(sel, SelectorFromMap(np.GetSpec().GetPodSelector().GetMatchLabels()))
		} else {
			sel = SelectorFromMap(np.GetSpec().GetPodSelector().GetMatchLabels())
			isEmpty = len(np.GetSpec().GetPodSelector().GetMatchLabels()) == 0
		}
	} else if action == central.ResourceAction_CREATE_RESOURCE {
		sel = SelectorFromMap(np.GetSpec().GetPodSelector().GetMatchLabels())
		isEmpty = len(np.GetSpec().GetPodSelector().GetMatchLabels()) == 0
	}
	return sel, isEmpty
}

func (h *networkPolicyDispatcher) updateDeploymentsFromStore(np *storage.NetworkPolicy, sel selector, isEmptySelector bool) {
	if !isEmptySelector {
		for _, deploymentWrap := range h.deploymentStore.getMatchingDeployments(np.GetNamespace(), sel) {
			h.detector.ProcessDeployment(deploymentWrap.GetDeployment(), central.ResourceAction_UPDATE_RESOURCE)
		}
	} else {
		// Network Policies with no selector match with all the deployments in the namespace
		for _, deploymentWrap := range h.deploymentStore.getAllDeploymentsInNamespace(np.GetNamespace()) {
			h.detector.ProcessDeployment(deploymentWrap.GetDeployment(), central.ResourceAction_UPDATE_RESOURCE)
		}
	}
}
